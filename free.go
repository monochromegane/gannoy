package gannoy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"syscall"
)

type Free struct {
	mu   sync.Mutex
	free []int
	file *os.File
}

func newFree(filename string) Free {
	_, err := os.Stat(filename)
	if err != nil {
		initializeFree(filename)
	}
	f, _ := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0)
	return Free{
		mu:   sync.Mutex{},
		free: []int{},
		file: f,
	}
}

func initializeFree(filename string) {
	f, _ := os.Create(filename)
	defer f.Close()
}

func (f *Free) push(index int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, int32(index))
	f.file.Write(buf.Bytes())
}

func (f *Free) pop() (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	b := make([]byte, 4)
	stat, _ := f.file.Stat()
	size := stat.Size()
	if size == 0 {
		return 0, fmt.Errorf("free list is empty.")
	}
	syscall.Pread(int(f.file.Fd()), b, size-4)

	buf := bytes.NewReader(b)

	var free int32
	binary.Read(buf, binary.BigEndian, &free)

	err := syscall.Ftruncate(int(f.file.Fd()), size-4)
	if err != nil {
		return 0, err
	}

	return int(free), nil
}
