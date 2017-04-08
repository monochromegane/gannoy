package annoy

type Nodes struct {
	Storage
}

type Node struct {
	storage Storage

	nDescendants int
	id           int
	fk           int
	parents      []int
	children     []int
	v            []float64
	ref          bool

	isNewRecord bool
}

func (n Node) isLeaf() bool {
	return n.nDescendants == 1
}

func (n Node) isBucket() bool {
	return len(n.v) == 0
}

func (ns Nodes) NewNode() Node {
	return Node{
		storage: ns.Storage,

		nDescendants: 1,
		id:           -1,
		fk:           -1,
		parents:      []int{},
		children:     []int{0, 0},
		v:            []float64{},
		ref:          true,

		isNewRecord: true,
	}
}

func (ns Nodes) GetNode(index int) Node {
	return ns.Storage.Find(index)
}

func (n *Node) Save() bool {
	if n.isNewRecord {
		id, _ := n.storage.Create(*n)
		n.id = id
		n.isNewRecord = false
		return true
	} else {
		n.storage.Update(*n)
		return true
	}
	return true
}

func (n *Node) Destroy() bool {
	n.storage.Delete(*n)
	n.ref = false
	return true
}
