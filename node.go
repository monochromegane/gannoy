package gannoy

type Nodes struct {
	Storage
	free Free
	maps Maps
}

func newNodes(filename string, tree, dim, K int) Nodes {
	// TODO Switch storage by parameter
	nodes := Nodes{
		Storage: newFile(filename, tree, dim, K),
	}
	// initialize free and maps
	nodes.initialize()
	return nodes
}

func (n *Nodes) initialize() {
	n.free = newFree()
	n.maps = newMaps()

	iterator := make(chan Node)
	go n.Iterate(iterator)

	for node := range iterator {
		if node.free {
			n.free.push(node.id)
		} else {
			if node.isLeaf() {
				n.maps.add(node.id, node.key)
			}
		}
	}
}

func (ns *Nodes) newNode() Node {
	node := Node{
		storage: ns.Storage,

		nDescendants: 1,
		id:           -1,
		key:          -1,
		parents:      []int{},
		children:     []int{0, 0},
		v:            []float64{},
		free:         false,

		isNewRecord: true,
	}
	if free, err := ns.free.pop(); err == nil {
		node.id = free
		node.isNewRecord = false
	}
	return node
}

func (ns Nodes) getNode(id int) (Node, error) {
	return ns.Storage.Find(id)
}

func (ns *Nodes) getNodeByKey(key int) (Node, error) {
	id, err := ns.maps.getId(key)
	if err != nil {
		return Node{}, err
	}
	return ns.getNode(id)
}

type Node struct {
	storage Storage

	nDescendants int
	id           int
	key          int
	parents      []int
	children     []int
	v            []float64
	free         bool
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

func (n *Node) destroy() error {
	err := n.storage.Delete(*n)
	if err != nil {
		return err
	}
	n.free = true
	return nil
}
