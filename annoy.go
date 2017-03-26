package annoy

import (
	"fmt"
	"math"
	"sort"

	"github.com/gansidui/priority_queue"
	"github.com/k0kubun/pp"
)

type AnnoyIndex struct {
	f      int
	D      Distance
	random Random
	nodes  Nodes
	nItems int
	loaded bool
	roots  []int
	K      int
}

func NewAnnoyIndex(f int, distance Distance, random Random) AnnoyIndex {
	return AnnoyIndex{
		f:      f,
		D:      distance,
		random: random,
		nodes:  Nodes{},
		loaded: false,
		roots:  []int{},
		// K:      52,
		K: 3,
	}
}

func (a *AnnoyIndex) AddItem(id int, w []float64) {
	_, n := a.nodes.newNode()
	n.id = id
	n.v = w
	a.nItems++
}

func (a *AnnoyIndex) AddNode(id int, w []float64) {

	idx, n := a.nodes.newNode()
	n.id = id
	n.v = w
	a.nItems++

	// 所属ノードを見つける
	for _, root := range a.roots {
		item := a.findBranchByVector(root, w)
		found := a.nodes.get(item)
		fmt.Printf("Found %d\n", item)
		pp.Println(found)

		org_parent := found.parents[root]

		if found.isBucket() && len(found.children) < a.K {
			// ノードに余裕があれば追加
			fmt.Printf("pattern bucket\n")
			n.parents[root] = item
			found.nDescendants++
			found.children = append(found.children, idx)
		} else {
			// ノードが上限またはリーフノードであれば新しいノードを追加
			willDelete := false
			var indices []int
			if found.isLeaf() {
				fmt.Printf("pattern leaf node\n")
				indices = []int{item, idx}
			} else {
				fmt.Printf("pattern full backet\n")
				indices = append(found.children, idx)
				willDelete = true
			}

			m := a.makeTree(root, org_parent, indices)
			parent := a.nodes.get(org_parent)
			parent.nDescendants++
			children := []int{}
			for _, child := range parent.children {
				if child != item {
					children = append(children, child)
				}
			}
			parent.children = append(children, m)

			if willDelete {
				found.release()
			}
		}
	}
}

func (a *AnnoyIndex) DeleteNode(item int) {
	node := a.nodes.get(item)
	for _, root := range a.roots {
		parent := a.nodes.get(node.parents[root])

		if parent.isBucket() && len(parent.children) > 2 {
			fmt.Printf("pattern bucket\n")
			children := []int{}
			for _, child := range parent.children {
				if child != item {
					children = append(children, child)
				}
			}
			parent.nDescendants--
			parent.children = children
		} else {
			fmt.Printf("pattern leaf node\n")
			var other int
			for _, child := range parent.children {
				if child != item {
					other = child
				}
			}
			grandParent := a.nodes.get(parent.parents[root])
			children := []int{}
			for _, child := range grandParent.children {
				if child == node.parents[root] {
					children = append(children, other)
				} else {
					children = append(children, child)
				}
			}
			grandParent.nDescendants--
			grandParent.children = children
			a.nodes.get(other).parents[root] = parent.parents[root]
			parent.ref = false
		}
	}
	node.ref = false
}

func (a AnnoyIndex) findBranchByVector(item int, v []float64) int {
	node := a.nodes.get(item)
	if node.isLeaf() || node.isBucket() {
		return item
	}
	side := a.D.side(node, v, a.f, a.random)
	return a.findBranchByVector(node.children[side], v)
}

func (a *AnnoyIndex) Build(q int) {
	if a.loaded {
		return
	}

	for {
		if q == -1 && a.nodes.size() >= a.nItems*2 {
			break
		}
		if q != -1 && len(a.roots) >= q {
			break
		}

		indices := make([]int, a.nItems)
		for i := 0; i < a.nItems; i++ {
			indices[i] = i
		}
		a.roots = append(a.roots, a.makeTree(-1, -1, indices))
	}
}

func (a *AnnoyIndex) makeTree(root, parent int, indices []int) int {
	r := root
	if r == -1 {
		r = parent
	}
	if len(indices) == 1 {
		n := a.nodes.get(indices[0])
		n.parents[r] = parent
		return indices[0]
	}

	if len(indices) <= a.K {
		item, m := a.nodes.newNode()
		m.nDescendants = len(indices)
		m.parents[r] = parent
		m.children = indices
		for _, child := range indices {
			c := a.nodes.get(child)
			c.parents[r] = item
		}
		return item
	}

	children := make([]*Node, len(indices))
	for i, idx := range indices {
		children[i] = a.nodes.get(idx)
	}

	childrenIndices := [2][]int{[]int{}, []int{}}

	item, m := a.nodes.newNode()
	m.nDescendants = len(indices)
	if root == -1 {
		root = item
	}
	m.parents[root] = parent

	a.D.createSplit(children, a.f, a.random, m)
	for _, idx := range indices {
		n := a.nodes.get(idx)
		side := a.D.side(m, n.v, a.f, a.random)
		childrenIndices[side] = append(childrenIndices[side], idx)
	}

	for len(childrenIndices[0]) == 0 || len(childrenIndices[1]) == 0 {
		childrenIndices[0] = []int{}
		childrenIndices[1] = []int{}
		for z := 0; z < a.f; z++ {
			m.v[z] = 0.0
		}
		for _, idx := range indices {
			side := a.random.flip()
			childrenIndices[side] = append(childrenIndices[side], idx)
		}
	}

	var flip int
	if len(childrenIndices[0]) > len(childrenIndices[1]) {
		flip = 1
	}
	for side := 0; side < 2; side++ {
		m.children[side^flip] = a.makeTree(root, item, childrenIndices[side^flip])
	}

	return item
}

func (a AnnoyIndex) GetNnsByItem(item, n, search_k int) []int {
	m := a.nodes.get(item)
	return a.getAllNns(m.v, n, search_k)
}

type Queue struct {
	priority float64
	value    int
}

func (q *Queue) Less(other interface{}) bool {
	return q.priority < other.(*Queue).priority
}

func (a AnnoyIndex) Tree() {
	for _, root := range a.roots {
		a.tree(root, a.nodes.get(root), root, 0)
	}
}

func (a AnnoyIndex) tree(root int, node *Node, id, tab int) {
	for i := 0; i < tab*2; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%d (%d) [nDescendants: %d, v: %v]\n", id, node.parents[root], node.nDescendants, node.v)
	if !node.isLeaf() {
		for _, child := range node.children {
			a.tree(root, a.nodes.get(child), child, tab+1)
		}
	}
}

func (a AnnoyIndex) getAllNns(v []float64, n, search_k int) []int {
	if search_k == -1 {
		search_k = n * len(a.roots)
	}

	q := priority_queue.New()
	for _, root := range a.roots {
		q.Push(&Queue{priority: math.Inf(1), value: root})
	}

	nns := []int{}
	for len(nns) < search_k && q.Len() > 0 {
		top := q.Top().(*Queue)
		d := top.priority
		i := top.value

		nd := a.nodes.get(i)
		q.Pop()
		if nd.isLeaf() && i < a.nItems {
			nns = append(nns, i)
		} else if nd.nDescendants <= a.K {
			dst := nd.children
			nns = append(nns, dst...)
		} else {
			margin := a.D.margin(nd, v, a.f)
			q.Push(&Queue{priority: math.Min(d, +margin), value: nd.children[1]})
			q.Push(&Queue{priority: math.Min(d, -margin), value: nd.children[0]})
		}
	}

	type Dist struct {
		distance float64
		item     int
	}

	sort.Ints(nns)
	nnsDist := []Dist{}
	last := -1
	for _, j := range nns {
		if j == last {
			continue
		}
		last = j
		nnsDist = append(nnsDist, Dist{distance: a.D.distance(v, a.nodes.get(j).v, a.f), item: j})
	}

	m := len(nnsDist)
	p := m
	if n < m {
		p = n
	}

	result := []int{}
	sort.Slice(nnsDist, func(i, j int) bool {
		return nnsDist[i].distance < nnsDist[j].distance
	})
	for i := 0; i < p; i++ {
		result = append(result, nnsDist[i].item)
	}

	return result
}
