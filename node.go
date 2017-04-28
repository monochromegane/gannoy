package gannoy

type Nodes struct {
	Storage
}

func newNodes(filename string, tree, dim, K int) Nodes {
	// TODO Switch storage by parameter
	return Nodes{
		Storage: newFile(filename, tree, dim, K),
	}
}

func (ns Nodes) newNode() Node {
	return Node{
		storage: ns.Storage,

		nDescendants: 1,
		id:           -1,
		parents:      []int{},
		children:     []int{0, 0},
		v:            []float64{},

		isNewRecord: true,
	}
}

func (ns Nodes) getNode(index int) Node {
	return ns.Storage.Find(index)
}

type Node struct {
	storage Storage

	nDescendants int
	id           int
	parents      []int
	children     []int
	v            []float64
	isNewRecord  bool
}

func (n Node) isLeaf() bool {
	return n.nDescendants == 1
}

func (n Node) isBucket() bool {
	return len(n.v) == 0
}

func (n Node) isRoot(index int) bool {
	return n.parents[index] == -1
}

func (n *Node) save() error {
	if n.isNewRecord {
		id, err := n.storage.Create(*n)
		if err != nil {
			return err
		}
		n.id = id
		n.isNewRecord = false
		return nil
	} else {
		return n.storage.Update(*n)
	}
}

func (n *Node) updateParents(index, parent int) error {
	return n.storage.UpdateParent(n.id, index, parent)
}
