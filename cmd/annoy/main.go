package main

import (
	"math/rand"
	"time"

	"github.com/monochromegane/annoy"
)

func main() {
	annoy := annoy.NewAnnoyIndex(3, annoy.Angular{}, annoy.RandRandom{})
	// annoy.AddItem(0, []float64{0.1, 0.1, 0.0})
	// annoy.AddItem(1, []float64{0.1, 0.1, 0.1})
	// annoy.AddItem(2, []float64{0.1, 0.1, 0.1})
	// annoy.AddItem(3, []float64{0.5, 0.5, 0.0})
	// annoy.AddItem(4, []float64{0.5, 0.5, 0.1})
	// annoy.AddItem(5, []float64{0.5, 0.5, 0.1})
	// annoy.AddItem(6, []float64{0.5, 0.5, 0.2})
	// annoy.AddItem(7, []float64{0.5, 0.5, 0.2})
	// annoy.AddItem(8, []float64{0.1, 0.1, 0.0})
	// annoy.AddItem(9, []float64{0.5, 0.5, 0.0})
	// annoy.Build(1)

	// annoy.Tree()

	// pp.Print(annoy.GetNnsByItem(8, 5, -1))
	// annoy.AddNode(10, []float64{0.5, 0.5, 0.2})

	// annoy.DeleteNode(9)

	// fmt.Printf("------------\n")

	// annoy.Save("test2.ann")
	annoy.Load("test2.ann")
	// annoy.Tree()
	// pp.Print(annoy.GetNnsByItem(8, 5, -1))
	// annoy.AddItem(10, []float64{0.5, 0.5, 0.2})
	// annoy.Tree()

	rand.Seed(time.Now().UnixNano())
	// var wg sync.WaitGroup
	// go func() {
	for i := 0; i < 10; i++ {
		// wg.Add(1)
		annoy.AddItem(10, []float64{rand.Float64(), rand.Float64(), rand.Float64()})
	}
	// wg.Done()
	// }()
	// go func() {
	// 	for i := 0; i < 1000; i++ {
	// 		annoy.AddItem(20, []float64{rand.Float64(), rand.Float64(), rand.Float64()})
	// 	}
	// 	wg.Done()
	// }()
	// wg.Wait()
	annoy.Tree()

}
