package annoy

import (
	"math"
)

type Distance interface {
	createSplit([]Node, int, Random, Node) Node
	distance([]float64, []float64, int) float64
	side(Node, []float64, int, Random) int
	margin(Node, []float64, int) float64
}

type Angular struct {
}

func (a Angular) createSplit(nodes []Node, f int, random Random, n Node) Node {
	bestIv, bestJv := twoMeans(a, nodes, f, random, true)
	for z := 0; z < f; z++ {
		n.v = append(n.v, bestIv[z]-bestJv[z])
	}
	n.v = normalize(n.v, f)
	return n
}

func (a Angular) distance(x, y []float64, f int) float64 {
	var pp, qq, pq float64
	for z := 0; z < f; z++ {
		pp += x[z] * x[z]
		qq += y[z] * y[z]
		pq += x[z] * y[z]
	}
	ppqq := pp * qq
	if ppqq > 0 {
		return 2.0 - 2.0*pq/math.Sqrt(ppqq)
	}
	return 2.0
}

func (a Angular) side(n Node, y []float64, f int, random Random) int {
	dot := a.margin(n, y, f)
	if dot != 0.0 {
		if dot > 0 {
			return 1
		} else {
			return 0
		}
	}
	return random.flip()
}

func (a Angular) margin(n Node, y []float64, f int) float64 {
	dot := 0.0
	for z := 0; z < f; z++ {
		dot += n.v[z] * y[z]
	}
	return dot
}
