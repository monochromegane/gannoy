package annoy

import (
	"bytes"
	"encoding/binary"
	"os"
)

type Storage interface {
	Create(Node) (int, error)
	Find(int) Node
	Update(Node) error
	Delete(Node) error
}

type Memory struct {
	records []Node
}

func (m *Memory) Create(n Node) (int, error) {
	id := len(m.records)
	n.id = id
	n.isNewRecord = false
	m.records = append(m.records, n)
	return id, nil
}

func (m *Memory) Find(index int) Node {
	return m.records[index]
}

func (m *Memory) Update(n Node) error {
	m.records[n.id] = n
	return nil
}

func (m *Memory) Delete(n Node) error {
	n.ref = false
	m.records[n.id] = n
	return nil
}

func (m *Memory) Save(f, K, q int, roots []int, name string) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := &bytes.Buffer{}

	// roots
	for _, root := range roots {
		binary.Write(buf, binary.BigEndian, int32(root))
	}
	binary.Write(buf, binary.BigEndian, int32(-1))
	file.Write(buf.Bytes())
	buf.Reset()

	storage := &File{
		f:    f,
		K:    K,
		q:    q,
		file: file,
	}
	for _, node := range m.records {
		storage.Create(node)
	}
	return nil
}

type File struct {
	file *os.File
	f    int
	K    int
	q    int
}

func (f *File) Create(n Node) (int, error) {
	buf := &bytes.Buffer{}
	f.nodeToBuf(buf, n)
	f.file.Seek(0, os.SEEK_END)
	id := f.nodeCount()
	f.file.Write(buf.Bytes())
	buf.Reset()
	return id, nil
}

func (f *File) Find(index int) Node {
	node := Node{}
	node.id = index
	node.storage = f
	f.file.Seek(f.offset(index), 0)

	var ref bool
	b := make([]byte, 1)
	f.file.Read(b)
	binary.Read(bytes.NewBuffer(b), binary.BigEndian, &ref)
	node.ref = ref

	var fk int32
	b = make([]byte, 4)
	f.file.Read(b)
	binary.Read(bytes.NewBuffer(b), binary.BigEndian, &fk)
	node.fk = int(fk)

	var nDescendants int32
	b = make([]byte, 4)
	f.file.Read(b)
	binary.Read(bytes.NewBuffer(b), binary.BigEndian, &nDescendants)
	node.nDescendants = int(nDescendants)

	parents := make([]int32, f.q)
	b = make([]byte, 4*f.q)
	f.file.Read(b)
	binary.Read(bytes.NewBuffer(b), binary.BigEndian, &parents)
	nodeParents := make([]int, f.q)
	for i, parent := range parents {
		nodeParents[i] = int(parent)
	}
	node.parents = nodeParents

	if node.nDescendants == 1 {
		// leaf node
		vec := make([]float64, f.f)
		b = make([]byte, 8*f.f)
		f.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &vec)
		node.v = vec
	} else if node.nDescendants <= f.K {
		// bucket node
		f.file.Seek(int64(8*f.f), 1) // skip v
		children := make([]int32, nDescendants)
		b = make([]byte, 4*nDescendants)
		f.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &children)
		nodeChildren := make([]int, nDescendants)
		for i, child := range children {
			nodeChildren[i] = int(child)
		}
		node.children = nodeChildren
	} else {
		// other node
		vec := make([]float64, f.f)
		b = make([]byte, 8*f.f)
		f.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &vec)
		node.v = vec

		children := make([]int32, 2)
		b = make([]byte, 4*2)
		f.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &children)
		nodeChildren := make([]int, 2)
		for i, child := range children {
			nodeChildren[i] = int(child)
		}
		node.children = nodeChildren
	}
	return node
}

func (f *File) Update(n Node) error {
	buf := &bytes.Buffer{}
	f.nodeToBuf(buf, n)
	f.file.Seek(f.offset(n.id), 0)
	f.file.Write(buf.Bytes())
	buf.Reset()
	return nil
}

func (f *File) Delete(n Node) error {
	n.ref = false
	f.Update(n)
	return nil
}

func (f File) offset(item int) int64 {
	return int64((f.q+1)*4) + (int64(item) * f.nodeSize())
}

func (f File) nodeCount() int {
	stat, _ := f.file.Stat()
	size := stat.Size()
	return int((size - int64((f.q+1)*4)) / f.nodeSize())
}

func (f File) nodeSize() int64 {
	return int64(1 + 4 + 4 + 4*f.q + 4*f.K + 8*f.f)
}

func (f File) nodeToBuf(buf *bytes.Buffer, node Node) {
	// 1bytes ref
	binary.Write(buf, binary.BigEndian, node.ref)

	// 4bytes foreign key
	binary.Write(buf, binary.BigEndian, int32(node.fk))

	// 4bytes nDescendants
	binary.Write(buf, binary.BigEndian, int32(node.nDescendants))

	// 4bytes parents
	parents := make([]int32, len(node.parents))
	for i, parent := range node.parents {
		parents[i] = int32(parent)
	}
	binary.Write(buf, binary.BigEndian, parents)

	// 8bytes v in f
	vec := make([]float64, f.f)
	for i, v := range node.v {
		vec[i] = float64(v)
	}
	binary.Write(buf, binary.BigEndian, vec)

	// 4bytes children in K
	children := make([]int32, f.K)
	for i, child := range node.children {
		children[i] = int32(child)
	}
	binary.Write(buf, binary.BigEndian, children)
}

func (f *File) Load(f_, k int, name string) []int {
	file, _ := os.OpenFile(name, os.O_RDWR, 0)
	f.file = file
	f.f = f_
	f.K = k

	roots := []int{}
	for {
		var root int32
		b := make([]byte, 4)
		_, err := f.file.Read(b)
		if err != nil {
			break
		}
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &root)
		if root == -1 {
			break
		}
		roots = append(roots, int(root))
	}
	f.q = len(roots)
	return roots
}
