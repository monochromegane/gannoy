package main

import (
	"math/rand"
	"sync"
	"time"

	"github.com/monochromegane/gannoy"
)

func main() {
	gannoy := gannoy.NewGannoyIndex(2, 3, "test.ann", gannoy.Angular{}, gannoy.RandRandom{})
	gannoy.Tree()

	// r := gannoy.GetNnsByItem(100000, 10, -1)
	// fmt.Printf("%v\n", r)

	rand.Seed(time.Now().UnixNano())
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			gannoy.AddItem(0, []float64{float64(i), rand.Float64(), rand.Float64()})
			wg.Done()
		}(i)
	}

	wg.Wait()
	gannoy.Tree()
}
