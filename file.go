package gannoy

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"syscall"
)

type File struct {
	tree       int
	dim        int
	K          int
	file       *os.File
	filename   string
	appendFile *os.File
	createChan chan createArgs
	locker     Locker
	nodeSize   int64
}

func newFile(filename string, tree, dim, K int) *File {
	_, err := os.Stat(filename)
	if err != nil {
		f, _ := os.Create(filename)
		f.Close()
	}

	file, _ := os.OpenFile(filename, os.O_RDWR, 0)
	appendFile, _ := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0)

	f := &File{
		tree:       tree,
		dim:        dim,
		K:          K,
		file:       file,
		filename:   filename,
		appendFile: appendFile,
		createChan: make(chan createArgs, 1),
		locker:     newLocker(),
		nodeSize: int64(1 + // free
			4 + // nDescendants
			4 + // key
			4*tree + // parents
			4*2 + // children
			8*dim), // v
	}
	go f.creator()
	return f
}

func (f *File) Create(n Node) (int, error) {
	args := createArgs{node: n, result: make(chan createResult)}
	f.createChan <- args
	result := <-args.result
	return result.id, result.err
}

func (f *File) create(n Node) (int, error) {
	buf := &bytes.Buffer{}
	f.nodeToBuf(buf, n)
	id := f.nodeCount()
	_, err := f.appendFile.Write(buf.Bytes())
	return id, err
}

func (f *File) Find(id int) (Node, error) {
	node := Node{}
	node.id = id
	node.storage = f
	offset := f.offset(id)
	err := f.locker.ReadLock(f.file.Fd(), offset, f.nodeSize)
	if err != nil {
		return node, err
	}
	defer f.locker.UnLock(f.file.Fd(), offset, f.nodeSize)

	b := make([]byte, f.nodeSize)
	_, err = syscall.Pread(int(f.file.Fd()), b, offset)
	if err != nil {
		return node, err
	}

	buf := bytes.NewReader(b)

	var free bool
	binary.Read(buf, binary.BigEndian, &free)
	node.free = free

	var nDescendants int32
	binary.Read(buf, binary.BigEndian, &nDescendants)
	node.nDescendants = int(nDescendants)

	var key int32
	binary.Read(buf, binary.BigEndian, &key)
	node.key = int(key)

	parents := make([]int32, f.tree)
	binary.Read(buf, binary.BigEndian, &parents)
	nodeParents := make([]int, f.tree)
	for i, parent := range parents {
		nodeParents[i] = int(parent)
	}
	node.parents = nodeParents

	if node.nDescendants == 1 {
		// leaf node
		buf.Seek(int64(4*2), io.SeekCurrent) // skip children
		node.children = []int{0, 0}

		vec := make([]float64, f.dim)
		binary.Read(buf, binary.BigEndian, &vec)
		node.v = vec
	} else if node.nDescendants <= f.K {
		// bucket node
		children := make([]int32, nDescendants)
		binary.Read(buf, binary.BigEndian, &children)
		nodeChildren := make([]int, nDescendants)
		for i, child := range children {
			nodeChildren[i] = int(child)
		}
		node.children = nodeChildren
	} else {
		// other node
		children := make([]int32, 2)
		binary.Read(buf, binary.BigEndian, &children)
		nodeChildren := make([]int, 2)
		for i, child := range children {
			nodeChildren[i] = int(child)
		}
		node.children = nodeChildren

		vec := make([]float64, f.dim)
		binary.Read(buf, binary.BigEndian, &vec)
		node.v = vec
	}
	return node, nil
}

func (f *File) Update(n Node) error {
	buf := &bytes.Buffer{}
	f.nodeToBuf(buf, n)
	offset := f.offset(n.id)
	file, _ := os.OpenFile(f.filename, os.O_RDWR, 0)
	defer file.Close()

	err := f.locker.WriteLock(file.Fd(), offset, f.nodeSize)
	if err != nil {
		return err
	}
	defer f.locker.UnLock(file.Fd(), offset, f.nodeSize)

	_, err = syscall.Pwrite(int(file.Fd()), buf.Bytes(), offset)
	return err
}

func (f *File) UpdateParent(id, rootIndex, parent int) error {
	offset := f.offset(id) +
		int64(1+ // free
			4+ // nDescendants
			4+ // key
			4*rootIndex) // parents
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, int32(parent))

	file, _ := os.OpenFile(f.filename, os.O_RDWR, 0)
	defer file.Close()

	err := f.locker.WriteLock(file.Fd(), offset, 4)
	if err != nil {
		return err
	}
	defer f.locker.UnLock(file.Fd(), offset, 4)

	_, err = syscall.Pwrite(int(file.Fd()), buf.Bytes(), offset)
	return err
}

func (f *File) Delete(n Node) error {
	n.free = true
	return f.Update(n)
}

func (f *File) Iterate(c chan Node) {
	count := f.nodeCount()
	// TODO: Use goroutine
	for i := 0; i < count; i++ {
		n, err := f.Find(i)
		if err != nil {
			close(c)
		}
		c <- n
	}
	close(c)
}

func (f File) offset(id int) int64 {
	return (int64(id) * f.nodeSize)
}

func (f File) nodeCount() int {
	stat, _ := f.file.Stat()
	size := stat.Size()
	return int(size / f.nodeSize)
}

func (f File) nodeToBuf(buf *bytes.Buffer, node Node) {
	// 1bytes free
	binary.Write(buf, binary.BigEndian, node.free)

	// 4bytes nDescendants
	binary.Write(buf, binary.BigEndian, int32(node.nDescendants))

	// 4bytes key
	binary.Write(buf, binary.BigEndian, int32(node.key))

	// 4bytes parents
	parents := make([]int32, len(node.parents))
	for i, parent := range node.parents {
		parents[i] = int32(parent)
	}
	binary.Write(buf, binary.BigEndian, parents)

	if node.isBucket() {
		// 4bytes children in K
		children := make([]int32, f.K)
		for i, child := range node.children {
			children[i] = int32(child)
		}
		binary.Write(buf, binary.BigEndian, children)

		// padding by zero
		remainingSize := ((2*4 + 8*f.dim) - (4 * f.K))
		binary.Write(buf, binary.BigEndian, make([]int32, remainingSize/4))
	} else {
		// 4bytes children in K
		children := make([]int32, 2)
		for i, child := range node.children {
			children[i] = int32(child)
		}
		binary.Write(buf, binary.BigEndian, children)

		// 8bytes v in f
		vec := make([]float64, f.dim)
		for i, v := range node.v {
			vec[i] = float64(v)
		}
		binary.Write(buf, binary.BigEndian, vec)
	}
}

type createArgs struct {
	node   Node
	result chan createResult
}

type createResult struct {
	id  int
	err error
}

func (f *File) creator() {
	for args := range f.createChan {
		id, err := f.create(args.node)
		args.result <- createResult{
			id:  id,
			err: err,
		}
	}
}

func (f File) size() int64 {
	info, _ := f.file.Stat()
	return info.Size()
}
