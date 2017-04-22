package gannoy

import (
	"bytes"
	"encoding/binary"
	"os"
	"sync"
)

type Maps struct {
	file      *os.File
	idToIndex map[int]int
	indexToId map[int]int

	mu sync.RWMutex
}

func newMaps(filename string) Maps {
	_, err := os.Stat(filename)
	if err != nil {
		initializeMaps(filename)
	}
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0)
	idToIndex, indexToId := loadMaps(filename)
	return Maps{
		file:      file,
		idToIndex: idToIndex,
		indexToId: indexToId,
		mu:        sync.RWMutex{},
	}
}

func initializeMaps(filename string) {
	f, _ := os.Create(filename)
	defer f.Close()
}

func loadMaps(filename string) (map[int]int, map[int]int) {
	f, _ := os.Open(filename)
	defer f.Close()

	stat, _ := f.Stat()
	size := stat.Size()
	length := int(size / (4 * 2))

	idToIndex := map[int]int{}
	indexToId := map[int]int{}

	b := make([]byte, size)
	f.Read(b)
	buf := bytes.NewReader(b)
	for i := 0; i < length; i++ {
		var id int32
		binary.Read(buf, binary.BigEndian, &id)
		var index int32
		binary.Read(buf, binary.BigEndian, &index)

		idToIndex[int(id)] = int(index)
		indexToId[int(index)] = int(id)
	}
	return idToIndex, indexToId
}

func (m *Maps) add(index, id int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, int32(id))
	binary.Write(buf, binary.BigEndian, int32(index))
	m.file.Write(buf.Bytes())

	m.idToIndex[id] = index
	m.indexToId[index] = id
}

func (m Maps) getId(index int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if id, ok := m.indexToId[index]; !ok {
		return -1
	} else {
		return id
	}
}

func (m Maps) getIndex(id int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if index, ok := m.idToIndex[id]; !ok {
		return -1
	} else {
		return index
	}
}

func (m Maps) offset(index int) int64 {
	return int64(index * 4)
}
