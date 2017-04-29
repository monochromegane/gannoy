package gannoy

import (
	"sync"
)

type Maps struct {
	mu sync.Mutex

	idToIndex map[int]int
}

func newMaps() Maps {
	return Maps{
		mu:        sync.Mutex{},
		idToIndex: map[int]int{},
	}
}

func (m *Maps) add(index, id int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idToIndex[id] = index
}

func (m *Maps) remove(key int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.idToIndex, key)
}

func (m *Maps) getIndex(id int) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index, ok := m.idToIndex[id]; !ok {
		return -1
	} else {
		return index
	}
}

func (m Maps) offset(index int) int64 {
	return int64(index * 4)
}
