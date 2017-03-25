package annoy

type Node struct {
	nDescendants int
	children     []int
	v            []float64
}

func NewNode() *Node {
	node := &Node{}
	node.children = []int{0, 0}
	node.v = []float64{}
	return node
}
