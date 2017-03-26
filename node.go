package annoy

type Vector []float64

type Nodes struct {
	nodes []*Node
}

func (ns Nodes) get(item int) *Node {
	if item >= ns.size() {
		return nil
	}
	return ns.nodes[item]
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
