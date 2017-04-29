package gannoy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"
)

type File struct {
	tree       int
	dim        int
	K          int
	file       *os.File
	appendFile *os.File
	createChan chan createArgs
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
		appendFile: appendFile,
		createChan: make(chan createArgs, 1),
	}
	go f.creator()
	return f
}

func (f *File) Create(n Node) (int, error) {
	args := createArgs{node: n, result: make(chan createResult)}
	f.createChan <- args
	result := <-args.result
	return result.index, result.err
}

func (f *File) create(n Node) (int, error) {
	buf := &bytes.Buffer{}
	f.nodeToBuf(buf, n)
	id := f.nodeCount()
	_, err := f.appendFile.Write(buf.Bytes())
	return id, err
}

func (f *File) Find(index int) Node {
	node := Node{}
	node.id = index
	node.storage = f
	err := syscall.FcntlFlock(f.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  f.offset(index),
		Len:    f.nodeSize(),
		Type:   syscall.F_RDLCK,
		Whence: io.SeekStart,
	})
	if err != nil {
		fmt.Printf("fcntl error %v\n", err)
	}
	defer syscall.FcntlFlock(f.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  f.offset(index),
		Len:    f.nodeSize(),
		Type:   syscall.F_UNLCK,
		Whence: io.SeekStart,
	})

	b := make([]byte, f.nodeSize())
	syscall.Pread(int(f.file.Fd()), b, f.offset(index))

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
		vec := make([]float64, f.dim)
		binary.Read(buf, binary.BigEndian, &vec)
		node.v = vec
	} else if node.nDescendants <= f.K {
		// bucket node
		buf.Seek(int64(8*f.dim), io.SeekCurrent) // skip v
		children := make([]int32, nDescendants)
		binary.Read(buf, binary.BigEndian, &children)
		nodeChildren := make([]int, nDescendants)
		for i, child := range children {
			nodeChildren[i] = int(child)
		}
		node.children = nodeChildren

	} else {
		// other node
		vec := make([]float64, f.dim)
		binary.Read(buf, binary.BigEndian, &vec)
		node.v = vec

		children := make([]int32, 2)
		binary.Read(buf, binary.BigEndian, &children)
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

	err := syscall.FcntlFlock(f.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  f.offset(n.id),
		Len:    f.nodeSize(),
		Type:   syscall.F_WRLCK,
		Whence: io.SeekStart,
	})
	if err != nil {
		return err
	}
	defer syscall.FcntlFlock(f.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  f.offset(n.id),
		Len:    f.nodeSize(),
		Type:   syscall.F_UNLCK,
		Whence: io.SeekStart,
	})
	_, err = syscall.Pwrite(int(f.file.Fd()), buf.Bytes(), f.offset(n.id))
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

	err := syscall.FcntlFlock(f.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  offset,
		Len:    4,
		Type:   syscall.F_WRLCK,
		Whence: io.SeekStart,
	})
	if err != nil {
		return err
	}
	defer syscall.FcntlFlock(f.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  offset,
		Len:    4,
		Type:   syscall.F_UNLCK,
		Whence: io.SeekStart,
	})
	_, err = syscall.Pwrite(int(f.file.Fd()), buf.Bytes(), offset)
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
		c <- f.Find(i)
	}
	close(c)
}

func (f File) offset(item int) int64 {
	return (int64(item) * f.nodeSize())
}

func (f File) nodeCount() int {
	stat, _ := f.file.Stat()
	size := stat.Size()
	return int(size / f.nodeSize())
}

func (f File) nodeSize() int64 {
	return int64(1 + // free
		4 + // nDescendants
		4 + // key
		4*f.tree + // parents
		8*f.dim + // v
		4*f.K) // children
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

	// 8bytes v in f
	vec := make([]float64, f.dim)
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

type createArgs struct {
	node   Node
	result chan createResult
}

type createResult struct {
	index int
	err   error
}

func (f *File) creator() {
	for args := range f.createChan {
		index, err := f.create(args.node)
		args.result <- createResult{
			index: index,
			err:   err,
		}
	}
}
