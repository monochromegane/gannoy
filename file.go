package gannoy

import (
	"bytes"
	"encoding/binary"
	"math"
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
	offsetOfV  int64
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
		offsetOfV: int64(1 + // free
			4 + // nDescendants
			4 + // key
			4*tree + // parents
			4*2), // children
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
	id := f.nodeCount()
	_, err := f.appendFile.Write(f.nodeToBytes(n))
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

	node.free = b[0] != 0
	node.nDescendants = int(int32(binary.BigEndian.Uint32(b[1:5])))
	node.key = int(int32(binary.BigEndian.Uint32(b[5:9])))

	node.parents = make([]int, f.tree)
	for i := 0; i < f.tree; i++ {
		node.parents[i] = int(int32(binary.BigEndian.Uint32(b[9+i*4 : 9+i*4+4])))
	}

	if node.nDescendants == 1 {
		// leaf node
		node.children = []int{0, 0} // skip children
		node.v = bytesToFloat64s(b[f.offsetOfV:])
	} else if node.nDescendants <= f.K {
		// bucket node
		node.children = make([]int, node.nDescendants)
		offsetOfChildren := int(f.offsetOfV - (4 * 2))
		for i := 0; i < node.nDescendants; i++ {
			node.children[i] = int(int32(binary.BigEndian.Uint32(b[offsetOfChildren+i*4 : offsetOfChildren+i*4+4])))
		}
	} else {
		// other node
		node.children = make([]int, 2)
		offsetOfChildren := int(f.offsetOfV - (4 * 2))
		for i := 0; i < 2; i++ {
			node.children[i] = int(int32(binary.BigEndian.Uint32(b[offsetOfChildren+i*4 : offsetOfChildren+i*4+4])))
		}
		node.v = bytesToFloat64s(b[f.offsetOfV:])
	}
	return node, nil
}

func (f *File) Update(n Node) error {
	bytes := f.nodeToBytes(n)
	offset := f.offset(n.id)
	file, _ := os.OpenFile(f.filename, os.O_RDWR, 0)
	defer file.Close()

	err := f.locker.WriteLock(file.Fd(), offset, f.nodeSize)
	if err != nil {
		return err
	}
	defer f.locker.UnLock(file.Fd(), offset, f.nodeSize)

	_, err = syscall.Pwrite(int(file.Fd()), bytes, offset)
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

func (f File) nodeToBytes(node Node) []byte {
	bytes := make([]byte, f.nodeSize)

	// 1bytes free
	if node.free {
		bytes[0] = 1
	} else {
		bytes[0] = 0
	}
	// 4bytes nDescendants
	binary.BigEndian.PutUint32(bytes[1:5], uint32(node.nDescendants))
	// 4bytes key
	binary.BigEndian.PutUint32(bytes[5:9], uint32(node.key))
	// 4bytes parents
	for i := 0; i < f.tree; i++ {
		binary.BigEndian.PutUint32(bytes[9+i*4:9+i*4+4], uint32(node.parents[i]))
	}
	if node.isBucket() {
		// 4bytes children in K
		offsetOfChildren := int(f.offsetOfV - (4 * 2))
		for i, child := range node.children {
			binary.BigEndian.PutUint32(bytes[offsetOfChildren+i*4:offsetOfChildren+i*4+4], uint32(child))
		}
		// padding by zero (nothing to do)
	} else {
		offsetOfV := int(f.offsetOfV)
		offsetOfChildren := int(f.offsetOfV - (4 * 2))
		// 4bytes 2 children
		for i, child := range node.children {
			binary.BigEndian.PutUint32(bytes[offsetOfChildren+i*4:offsetOfChildren+i*4+4], uint32(child))
		}
		// 8bytes v in f
		for i, v := range node.v {
			binary.BigEndian.PutUint64(bytes[offsetOfV+i*8:offsetOfV+i*8+8], math.Float64bits(v))
		}
	}
	return bytes
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

func bytesToFloat64s(bytes []byte) []float64 {
	size := len(bytes) / 8
	floats := make([]float64, size)
	for i := 0; i < size; i++ {
		floats[i] = math.Float64frombits(binary.BigEndian.Uint64(bytes[0:8]))
		bytes = bytes[8:]
	}
	return floats
}
