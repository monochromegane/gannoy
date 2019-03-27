package gannoy

import (
	"fmt"
	"testing"
)
//
//func TestAngularMargin(t *testing.T) {
//	angular := Angular{}
//	node := Node{v: []float64{1, 2, 3}}
//	y := []float64{1, 2, 3}
//	dot := angular.margin(node, y)
//	expect := 14.0
//	if dot != expect {
//		t.Errorf("Angular margin should return %d, but %d", expect, dot)
//	}
//}

func TestAngularSide(t *testing.T) {
	angular := Angular{}

	// dot is plus (14.0)
	node := Node{v: []float64{1, 2, 3}}
	y := []float64{1, 2, 3}
	if side := angular.side(node, y, RandRandom{}); side != 1 {
		t.Errorf("Angular side should return 1, but %d", side)
	}

	// dot is minus (-14.0)
	node = Node{v: []float64{1, 2, 3}}
	y = []float64{-1, -2, -3}
	if side := angular.side(node, y, RandRandom{}); side != 0 {
		t.Errorf("Angular side should return 0, but %d", side)
	}
}

func TestAngularDistance(t *testing.T) {
	angular := Angular{}

	x := []float64{1, 2, 3}
	y := []float64{-1, -2, -3}
	expect := 4.0
	if distance := angular.distance(x, y); distance != expect {
		t.Errorf("Angular distance should return %f, but %f.", expect, distance)
	}
}

func TestAngularCreateSplit(t *testing.T) {
	angular := Angular{}
	nodes := []Node{
		{v: []float64{0.1, 0.1}},
		{v: []float64{1.1, 1.1}},
		{v: []float64{0.1, 1.1}},
		{v: []float64{1.1, 0.1}},
	}
	n := angular.createSplit(nodes, &TestLoopRandom{max: len(nodes)}, Node{})
	expect := []string{"0.822251", "-0.569124"}
	for i, v := range n.v {
		if strv := fmt.Sprintf("%f", v); strv != expect[i] {
			t.Errorf("Create split should return node.v %s, but %s", strv, expect[i])
		}
	}
}

type TestLoopRandom struct {
	max         int
	current     int
	flipCurrent int
}

func (r *TestLoopRandom) index(n int) int {
	index := r.current % r.max
	r.current++
	return index
}

func (r *TestLoopRandom) flip() int {
	r.flipCurrent++
	if r.flipCurrent%2 == 0 {
		return 0
	} else {
		return 1
	}
}
