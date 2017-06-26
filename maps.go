package gannoy

import (
	"fmt"
	"sync"
)

type Maps struct {
	mu *sync.RWMutex

	keyToId map[int]int
}

func newMaps() Maps {
	return Maps{
		mu:      &sync.RWMutex{},
		keyToId: map[int]int{},
	}
}

func (m *Maps) add(id, key int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keyToId[key] = id
}

func (m *Maps) remove(key int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.keyToId, key)
}

func (m Maps) getId(key int) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if id, ok := m.keyToId[key]; !ok {
		return -1, fmt.Errorf("not found")
	} else {
		return id, nil
	}
}
