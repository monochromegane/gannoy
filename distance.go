package gannoy

import (
	"math"
)

type Distance interface {
	createSplit([]Node, Random, Node) Node
	distance([]float64, []float64) float64
	side(Node, []float64, Random) int
	margin(Node, []float64) float64
}

type Angular struct {
}

func (a Angular) createSplit(nodes []Node, random Random, n Node) Node {
	bestIv, bestJv := twoMeans(a, nodes, random, true)
	v := make([]float64, len(nodes[0].v))
	for z, _ := range v {
		v[z] = bestIv[z] - bestJv[z]
	}
	n.v = normalize(n.v)
	return n
}

func (a Angular) distance(x, y []float64) float64 {
	var pp, qq, pq float64
	for z, xz := range x {
		pp += xz * xz
		qq += y[z] * y[z]
		pq += xz * y[z]
	}
	ppqq := pp * qq
	if ppqq > 0 {
		return 2.0 - 2.0*pq/math.Sqrt(ppqq)
	}
	return 2.0
}

func (a Angular) side(n Node, y []float64, random Random) int {
	dot := a.margin(n, y)
	if dot != 0.0 {
		if dot > 0 {
			return 1
		} else {
			return 0
		}
	}
	return random.flip()
}

func (a Angular) margin(n Node, y []float64) float64 {
	dot := 0.0
	for z, v := range n.v {
		dot += v * y[z]
	}
	return dot
}
