package main

import (
	"github.com/k0kubun/pp"
	"github.com/monochromegane/annoy"
)

func main() {
	annoy := annoy.NewAnnoyIndex(3, annoy.Angular{}, annoy.RandRandom{})
	annoy.AddItem(0, []float64{0.0, 0.1, 0.0})
	annoy.AddItem(1, []float64{0.0, 0.1, 0.1})
	annoy.AddItem(2, []float64{0.0, 0.1, 0.2})
	annoy.AddItem(3, []float64{0.0, 0.1, 0.3})
	annoy.AddItem(4, []float64{0.0, 0.1, 0.4})
	annoy.Build(1)

	pp.Print(annoy)
	pp.Print(annoy.GetNnsByItem(2, 2, -1))
}
