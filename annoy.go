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
	nodes  []*Node
	nItems int
	loaded bool
	nNodes int
	roots  []int
	K      int
}

func NewAnnoyIndex(f int, distance Distance, random Random) AnnoyIndex {
	return AnnoyIndex{
		f:      f,
		D:      distance,
		random: random,
		nodes:  []*Node{},
		loaded: false,
		nNodes: 0,
		roots:  []int{},
		// K:      52,
		K: 3,
	}
}

func (a *AnnoyIndex) AddItem(item int, w []float64) {
	n := a.get(item)

	n.children[0] = 0
	n.children[1] = 0
	n.nDescendants = 1
	n.v = w

	a.nItems = item + 1
}

func (a *AnnoyIndex) AddNode(w []float64) {

	// 所属ノードを見つける
	for _, root := range a.roots {
		item := a.findBranchByVector(root, w)
		found := a.get(item)
		fmt.Printf("Found %d\n", item)
		pp.Println(found)
		// リーフノードであれば新しいノードを追加
		if found.nDescendants == 1 {
			fmt.Printf("pattern %s\n", "A")
			n := a.get(-1)

			n.children[0] = 0
			n.children[1] = 0
			n.nDescendants = 1
			n.v = w

			a.nItems++
			a.nNodes++

			a.makeTree([]int{item, len(a.nodes) - 1})
		} else {
			if len(found.children) < a.K {
				fmt.Printf("pattern %s\n", "B")
				// ノードに余裕があれば追加
				n := a.get(-1)

				n.children[0] = 0
				n.children[1] = 0
				n.nDescendants = 1
				n.v = w

				a.nItems++
				a.nNodes++

				found.nDescendants += 1
				found.children = append(found.children, len(a.nodes)-1)
			} else {
				fmt.Printf("pattern %s\n", "C")
				// ノードが最大であれば新しいノードを追加
			}
		}
	}
}

func (a AnnoyIndex) findBranchByVector(item int, v []float64) int {
	node := a.get(item)
	if node.nDescendants == 1 || len(node.v) == 0 {
		return item
	}
	side := a.D.side(node, v, a.f, a.random)
	return a.findBranchByVector(node.children[side], v)
}

func (a *AnnoyIndex) Build(q int) {
	if a.loaded {
		return
	}
	a.nNodes = a.nItems

	for {
		if q == -1 && a.nNodes >= a.nItems*2 {
			break
		}
		if q != -1 && len(a.roots) >= q {
			break
		}

		indices := []int{}
		for i := 0; i < a.nItems; i++ {
			indices = append(indices, i)
		}
		a.roots = append(a.roots, a.makeTree(indices))
	}

	for i := 0; i < len(a.roots); i++ {
		d := a.get(a.nNodes + i)
		s := a.get(a.roots[i])
		d.nDescendants = s.nDescendants
		d.children = s.children
		d.v = s.v
	}
	a.nNodes += len(a.roots)
}

func (a *AnnoyIndex) get(item int) *Node {
	var node *Node
	if len(a.nodes) <= item || item == -1 {
		node = NewNode()
		a.nodes = append(a.nodes, node)
	} else {
		node = a.nodes[item]
	}
	return node
}

func (a *AnnoyIndex) makeTree(indices []int) int {
	if len(indices) == 1 {
		return indices[0]
	}

	if len(indices) <= a.K {
		item := a.nNodes
		a.nNodes++

		m := a.get(item)
		m.nDescendants = len(indices)
		m.children = indices
		return item
	}

	children := []*Node{}
	for i := 0; i < len(indices); i++ {
		j := indices[i]
		n := a.get(j)
		children = append(children, n)
	}

	childrenIndices := [2][]int{[]int{}, []int{}}
	m := NewNode()
	a.D.create_split(children, a.f, a.random, m)
	for i := 0; i < len(indices); i++ {
		j := indices[i]
		n := a.get(j)
		side := a.D.side(m, n.v, a.f, a.random)
		childrenIndices[side] = append(childrenIndices[side], j)
	}

	for len(childrenIndices[0]) == 0 || len(childrenIndices[1]) == 0 {
		childrenIndices[0] = []int{}
		childrenIndices[1] = []int{}
		for z := 0; z < a.f; z++ {
			m.v[z] = 0.0
		}
		for i := 0; i < len(indices); i++ {
			j := indices[i]
			side := a.random.flip()
			childrenIndices[side] = append(childrenIndices[side], j)
		}
	}

	var flip int
	if len(childrenIndices[0]) > len(childrenIndices[1]) {
		flip = 1
	}
	m.nDescendants = len(indices)
	for side := 0; side < 2; side++ {
		m.children[side^flip] = a.makeTree(childrenIndices[side^flip])
	}
	item := a.nNodes
	a.nNodes++
	node := a.get(item)
	node.nDescendants = m.nDescendants
	node.children = m.children
	node.v = m.v

	return item
}

func (a AnnoyIndex) GetNnsByItem(item, n, search_k int) []int {
	m := a.get(item)
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
		a.tree(a.get(root), root, 0)
	}
}

func (a AnnoyIndex) tree(node *Node, id, tab int) {
	for i := 0; i < tab*2; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%d [nDescendants: %d, v: %v]\n", id, node.nDescendants, node.v)
	if node.nDescendants == 1 {
	} else {
		for _, child := range node.children {
			a.tree(a.get(child), child, tab+1)
		}
	}
}

func (a AnnoyIndex) getAllNns(v []float64, n, search_k int) []int {
	if search_k == -1 {
		search_k = n * len(a.roots)
	}

	q := priority_queue.New()
	for i := 0; i < len(a.roots); i++ {
		q.Push(&Queue{priority: math.Inf(1), value: a.roots[i]})
	}

	nns := []int{}

	for len(nns) < search_k && q.Len() > 0 {
		top := q.Top().(*Queue)
		d := top.priority
		i := top.value

		nd := a.get(i)
		q.Pop()
		if nd.nDescendants == 1 && i < a.nItems {
			nns = append(nns, i)
		} else if nd.nDescendants <= a.K {
			dst := nd.children
			nns = append(nns, dst...)
		} else {
			margin := a.D.margin(nd, v, a.f)
			fmt.Printf("%f:%d = %f\n", d, i, margin)
			fmt.Printf("children[1] = %d\n", nd.children[1])
			fmt.Printf("children[0] = %d\n", nd.children[0])
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
	for i := 0; i < len(nns); i++ {
		j := nns[i]
		if j == last {
			continue
		}
		last = j
		nnsDist = append(nnsDist, Dist{distance: a.D.distance(v, a.get(j).v, a.f), item: j})
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
