package annoy

import (
	"bytes"
	"encoding/binary"
	"os"
)

type Vector []float64

type Nodes struct {
	nodes []*Node
	file  *os.File
	f     int
	k     int
}

func (ns Nodes) get(item int) *Node {
	if ns.file == nil {
		if item >= ns.size() {
			return nil
		}
		return ns.nodes[item]
	} else {
		return ns.getFromFile(item)
	}
}

func (ns Nodes) getFromFile(item int) *Node {
	node := newNode()
	ns.file.Seek(ns.offset(item), 0)
	var nDescendants int32
	b := make([]byte, 4)
	ns.file.Read(b)
	binary.Read(bytes.NewBuffer(b), binary.BigEndian, &nDescendants)
	node.nDescendants = int(nDescendants)
	if node.nDescendants == 1 {
		// leaf node
		vec := make([]float64, ns.f)
		b = make([]byte, 8*ns.f)
		ns.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &vec)
		node.v = vec
	} else if node.nDescendants <= ns.k {
		// bucket node
		ns.file.Seek(int64(8*ns.f), 1) // skip v
		children := make([]int32, nDescendants)
		b = make([]byte, 4*nDescendants)
		ns.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &children)
		nodeChildren := make([]int, nDescendants)
		for i, child := range children {
			nodeChildren[i] = int(child)
		}
		node.children = nodeChildren
	} else {
		// other node
		vec := make([]float64, ns.f)
		b = make([]byte, 8*ns.f)
		ns.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &vec)
		node.v = vec

		children := make([]int32, 2)
		b = make([]byte, 4*2)
		ns.file.Read(b)
		binary.Read(bytes.NewBuffer(b), binary.BigEndian, &children)
		nodeChildren := make([]int, 2)
		for i, child := range children {
			nodeChildren[i] = int(child)
		}
		node.children = nodeChildren
	}
	return node
}

func (ns *Nodes) load(file *os.File, f, k int) {
	ns.file = file
	ns.f = f
	ns.k = k
}

func (ns Nodes) offset(item int) int64 {
	return int64(item) * ns.nodeSize()
}

func (ns Nodes) nodeSize() int64 {
	return int64(4 + 4*ns.k + 8*ns.f)
}

func (ns *Nodes) newNode() (int, *Node) {
	node := newNode()
	ns.nodes = append(ns.nodes, node)
	return ns.size() - 1, node
}

func (ns Nodes) size() int {
	return len(ns.nodes)
}

type Node struct {
	nDescendants int
	id           int
	parents      map[int]int
	children     []int
	ref          bool
	v            []float64
}

func newNode() *Node {
	return &Node{
		nDescendants: 1,
		id:           0,
		parents:      map[int]int{},
		children:     []int{0, 0},
		ref:          true,
		v:            []float64{},
	}
}

func (n *Node) release() {
	n.ref = false
}

func (n Node) isRoot(root int) bool {
	return n.parents[root] == -1
}

func (n Node) isLeaf() bool {
	return n.nDescendants == 1
}

func (n Node) isBucket() bool {
	return len(n.v) == 0
}
