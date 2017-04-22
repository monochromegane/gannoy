package gannoy

import (
	"math/rand"
	"time"
)

type Random interface {
	index(int) int
	flip() int
}

type RandRandom struct {
}

func (r RandRandom) index(n int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(n)
}

func (r RandRandom) flip() int {
	return r.index(2)
}
