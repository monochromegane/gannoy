package gannoy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func CreateMeta(path, file string, tree, dim, K int) error {
	database := filepath.Join(path, file+".meta")
	_, err := os.Stat(database)
	if err == nil {
		return fmt.Errorf("Already exist database: %s.", database)
	}

	f, err := os.Create(database)
	if err != nil {
		return err
	}
	defer f.Close()

	binary.Write(f, binary.BigEndian, int32(tree))
	binary.Write(f, binary.BigEndian, int32(dim))
	binary.Write(f, binary.BigEndian, int32(K))
	roots := make([]int32, tree)
	for i, _ := range roots {
		roots[i] = int32(-1)
	}
	binary.Write(f, binary.BigEndian, roots)

	return nil
}

type meta struct {
	path string
	file *os.File
	tree int
	dim  int
	K    int
}

func loadMeta(filename string) (meta, error) {
	_, err := os.Stat(filename)
	if err != nil {
		return meta{}, err
	}
	file, _ := os.OpenFile(filename, os.O_RDWR, 0)

	b := make([]byte, 4*3)
	syscall.Pread(int(file.Fd()), b, 0)

	buf := bytes.NewReader(b)
	var tree, dim, K int32
	binary.Read(buf, binary.BigEndian, &tree)
	binary.Read(buf, binary.BigEndian, &dim)
	binary.Read(buf, binary.BigEndian, &K)

	return meta{
		path: filename,
		file: file,
		tree: int(tree),
		dim:  int(dim),
		K:    int(K),
	}, nil
}

func (m meta) rootOffset(index int) int64 {
	return int64(4 + // tree
		4 + // dim
		4 + // K
		4*index) // roots
}

func (m meta) roots() []int {
	err := syscall.FcntlFlock(m.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  m.rootOffset(0),
		Len:    int64(m.tree * 4),
		Type:   syscall.F_RDLCK,
		Whence: io.SeekStart,
	})
	if err != nil {
		return []int{}
	}
	defer syscall.FcntlFlock(m.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  m.rootOffset(0),
		Len:    int64(m.tree * 4),
		Type:   syscall.F_UNLCK,
		Whence: io.SeekStart,
	})

	b := make([]byte, m.tree*4)
	syscall.Pread(int(m.file.Fd()), b, m.rootOffset(0))
	buf := bytes.NewReader(b)
	roots := make([]int32, m.tree)
	binary.Read(buf, binary.BigEndian, &roots)
	result := make([]int, m.tree)
	for i, r := range roots {
		result[i] = int(r)
	}
	return result
}

func (m meta) updateRoot(index, root int) error {
	offset := m.rootOffset(index)
	err := syscall.FcntlFlock(m.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  offset,
		Len:    4,
		Type:   syscall.F_WRLCK,
		Whence: io.SeekStart,
	})
	if err != nil {
		return err
	}
	defer syscall.FcntlFlock(m.file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Start:  offset,
		Len:    4,
		Type:   syscall.F_UNLCK,
		Whence: io.SeekStart,
	})
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, int32(root))
	_, err = syscall.Pwrite(int(m.file.Fd()), buf.Bytes(), offset)
	if err != nil {
		return err
	}

	return err
}

func (m meta) treePath() string {
	return m.filePath("tree")
}

func (m meta) filePath(newExt string) string {
	ext := filepath.Ext(m.path)
	return fmt.Sprintf("%s.%s", strings.Split(m.path, ext)[0], newExt)
}
