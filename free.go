package gannoy

import "sync"

type Free struct {
	mu   sync.Mutex
	free []int
}

func newFree() Free {
	return Free{
		mu:   sync.Mutex{},
		free: []int{},
	}
}

func (f *Free) push(index int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.free = append(f.free, index)
}

func (f *Free) pop() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	x, newFree := f.free[len(f.free)-1], f.free[:len(f.free)-1]
	f.free = newFree
	return x
}
