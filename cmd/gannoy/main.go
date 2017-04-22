package main

import (
	"fmt"

	"github.com/monochromegane/gannoy"
)

func main() {
	gannoy := gannoy.NewGannoyIndex(1, 3, "test", gannoy.Angular{}, gannoy.RandRandom{})

	r := gannoy.GetNnsByItem(3, 10, -1)
	fmt.Printf("%v\n", r)

	// rand.Seed(time.Now().UnixNano())
	// var wg sync.WaitGroup
	// wg.Add(10)
	// for i := 0; i < 10; i++ {
	// 	go func(i int) {
	// 		gannoy.AddItem(i, []float64{float64(i), rand.Float64(), rand.Float64()})
	// 		wg.Done()
	// 	}(i)
	// }
	//
	// wg.Wait()
	// gannoy.Tree()
}
