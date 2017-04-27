package gannoy

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/gansidui/priority_queue"
)

type GannoyIndex struct {
	meta      meta
	maps      Maps
	tree      int
	dim       int
	distance  Distance
	random    Random
	nodes     Nodes
	K         int
	buildChan chan buildArgs
}

func NewGannoyIndex(metaFile string, distance Distance, random Random) (GannoyIndex, error) {

	meta, err := loadMeta(metaFile)
	if err != nil {
		return GannoyIndex{}, err
	}
	tree := meta.tree
	dim := meta.dim

	ann := meta.treePath()
	maps := meta.mapPath()

	// K := 3
	K := 50
	gannoy := GannoyIndex{
		meta:      meta,
		maps:      newMaps(maps),
		tree:      tree,
		dim:       dim,
		distance:  distance,
		random:    random,
		K:         K,
		nodes:     newNodes(ann, tree, dim, K),
		buildChan: make(chan buildArgs, 1),
	}
	go gannoy.builder()
	return gannoy, nil
}

func (g GannoyIndex) Tree() {
	for i, root := range g.meta.roots() {
		g.walk(i, g.nodes.getNode(root), root, 0)
	}
}

func (g *GannoyIndex) AddItem(id int, w []float64) error {
	args := buildArgs{action: ADD, id: id, w: w, result: make(chan error)}
	g.buildChan <- args
	return <-args.result
}

func (g *GannoyIndex) RemoveItem(id int) error {
	args := buildArgs{action: DELETE, id: id, result: make(chan error)}
	g.buildChan <- args
	return <-args.result
}

func (g *GannoyIndex) UpdateItem(id int, w []float64) error {
	args := buildArgs{action: UPDATE, id: id, w: w, result: make(chan error)}
	g.buildChan <- args
	return <-args.result
}

func (g GannoyIndex) GetNnsByItem(id, n, searchK int) []int {
	m := g.nodes.getNode(g.maps.getIndex(id))
	if !m.isLeaf() {
		return []int{}
	}
	indices := g.getAllNns(m.v, n, searchK)
	ids := make([]int, len(indices))
	for i, index := range indices {
		ids[i] = g.maps.getId(index)
	}
	return ids
}

func (g GannoyIndex) getAllNns(v []float64, n, searchK int) []int {
	if searchK == -1 {
		searchK = n * g.tree
	}

	q := priority_queue.New()
	for _, root := range g.meta.roots() {
		q.Push(&Queue{priority: math.Inf(1), value: root})
	}

	nns := []int{}
	for len(nns) < searchK && q.Len() > 0 {
		top := q.Top().(*Queue)
		d := top.priority
		i := top.value

		nd := g.nodes.getNode(i)
		q.Pop()
		if nd.isLeaf() {
			nns = append(nns, i)
		} else if nd.nDescendants <= g.K {
			dst := nd.children
			nns = append(nns, dst...)
		} else {
			margin := g.distance.margin(nd, v, g.dim)
			q.Push(&Queue{priority: math.Min(d, +margin), value: nd.children[1]})
			q.Push(&Queue{priority: math.Min(d, -margin), value: nd.children[0]})
		}
	}

	sort.Ints(nns)
	nnsDist := []sorter{}
	last := -1
	for _, j := range nns {
		if j == last {
			continue
		}
		last = j
		nnsDist = append(nnsDist, sorter{value: g.distance.distance(v, g.nodes.getNode(j).v, g.dim), id: j})
	}

	m := len(nnsDist)
	p := m
	if n < m {
		p = n
	}

	HeapSort(nnsDist, DESC, p)

	result := make([]int, p)
	for i := 0; i < p; i++ {
		result[i] = nnsDist[m-1-i].id
	}

	return result
}

func (g *GannoyIndex) addItem(id int, w []float64) error {
	n := g.nodes.newNode()
	n.v = w
	n.parents = make([]int, g.tree)
	err := n.save()
	if err != nil {
		return err
	}
	// fmt.Printf("id %d\n", n.id)

	var wg sync.WaitGroup
	wg.Add(g.tree)
	buildChan := make(chan int, g.tree)
	worker := func(n Node) {
		for index := range buildChan {
			// fmt.Printf("root: %d\n", g.meta.roots()[index])
			g.build(index, g.meta.roots()[index], n)
			wg.Done()
		}
	}

	for i := 0; i < 3; i++ {
		go worker(n)
	}

	for index, _ := range g.meta.roots() {
		buildChan <- index
	}

	wg.Wait()
	close(buildChan)
	g.maps.add(n.id, id)

	return nil
}

func (g *GannoyIndex) build(index, root int, n Node) {
	if root == -1 {
		// 最初のノード
		n.parents[index] = -1
		n.save()
		g.meta.updateRoot(index, n.id)
		return
	}
	item := g.findBranchByVector(root, n.v)
	found := g.nodes.getNode(item)
	// fmt.Printf("Found %d\n", item)

	org_parent := found.parents[index]
	if found.isBucket() && len(found.children) < g.K {
		// ノードに余裕があれば追加
		// fmt.Printf("pattern bucket\n")
		n.updateParents(index, item)
		found.nDescendants++
		found.children = append(found.children, n.id)
		found.save()
	} else {
		// ノードが上限またはリーフノードであれば新しいノードを追加
		willDelete := false
		var indices []int
		if found.isLeaf() {
			// fmt.Printf("pattern leaf node\n")
			indices = []int{item, n.id}
		} else {
			// fmt.Printf("pattern full backet\n")
			indices = append(found.children, n.id)
			willDelete = true
		}

		m := g.makeTree(index, org_parent, indices)
		// fmt.Printf("m: %d, org_parent: %d\n", m, org_parent)
		if org_parent == -1 {
			// rootノードの入れ替え
			g.meta.updateRoot(index, m)
		} else {
			parent := g.nodes.getNode(org_parent)
			parent.nDescendants++
			children := make([]int, len(parent.children))
			for i, child := range parent.children {
				if child == item {
					// 新しいノードに変更
					children[i] = m
				} else {
					// 既存のノードのまま
					children[i] = child
				}
			}
			parent.children = children
			parent.save()

		}
		if willDelete {
			found.destroy()
		}
	}
}

func (g *GannoyIndex) removeItem(id int) error {
	index := g.maps.getIndex(id)
	n := g.nodes.getNode(index)

	var wg sync.WaitGroup
	wg.Add(g.tree)
	buildChan := make(chan int, g.tree)
	worker := func(n Node) {
		for root := range buildChan {
			g.remove(root, n)
			wg.Done()
		}
	}

	for i := 0; i < 3; i++ {
		go worker(n)
	}
	for index, _ := range g.meta.roots() {
		buildChan <- index
	}

	wg.Wait()
	close(buildChan)

	g.maps.remove(n.id, id)
	n.ref = false
	n.save()

	return nil
}

func (g *GannoyIndex) remove(root int, node Node) {
	if node.isRoot(root) {
		g.meta.updateRoot(root, -1)
		return
	}
	parent := g.nodes.getNode(node.parents[root])
	if parent.isBucket() && len(parent.children) > 2 {
		// fmt.Printf("pattern bucket\n")
		target := -1
		for i, child := range parent.children {
			if child == node.id {
				target = i
			}
		}
		if target == -1 {
			return
		}
		children := append(parent.children[:target], parent.children[(target+1):]...)
		parent.nDescendants--
		parent.children = children
		parent.save()
	} else {
		// fmt.Printf("pattern leaf node\n")
		var other int
		for _, child := range parent.children {
			if child != node.id {
				other = child
			}
		}
		if parent.isRoot(root) {
			g.meta.updateRoot(root, other)
		} else {
			grandParent := g.nodes.getNode(parent.parents[root])
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
			grandParent.save()
		}

		otherNode := g.nodes.getNode(other)
		otherNode.updateParents(root, parent.parents[root])

		parent.ref = false
		parent.save()
	}
}

func (g GannoyIndex) findBranchByVector(index int, v []float64) int {
	node := g.nodes.getNode(index)
	if node.isLeaf() || node.isBucket() {
		return index
	}
	side := g.distance.side(node, v, g.dim, g.random)
	return g.findBranchByVector(node.children[side], v)
}

func (g *GannoyIndex) makeTree(root, parent int, indices []int) int {
	if len(indices) == 1 {
		n := g.nodes.getNode(indices[0])
		if len(n.parents) == 0 {
			n.parents = make([]int, g.tree)
		}
		n.updateParents(root, parent)
		return indices[0]
	}

	if len(indices) <= g.K {
		m := g.nodes.newNode()
		m.parents = make([]int, g.tree)
		m.nDescendants = len(indices)
		m.parents[root] = parent
		m.children = indices
		m.save()
		for _, child := range indices {
			c := g.nodes.getNode(child)
			if len(c.parents) == 0 {
				c.parents = make([]int, g.tree)
			}
			c.updateParents(root, m.id)
		}
		return m.id
	}

	children := make([]Node, len(indices))
	for i, idx := range indices {
		children[i] = g.nodes.getNode(idx)
	}

	childrenIndices := [2][]int{[]int{}, []int{}}

	m := g.nodes.newNode()
	m.parents = make([]int, g.tree)
	m.nDescendants = len(indices)
	m.parents[root] = parent

	m = g.distance.createSplit(children, g.dim, g.random, m)
	for _, idx := range indices {
		n := g.nodes.getNode(idx)
		side := g.distance.side(m, n.v, g.dim, g.random)
		childrenIndices[side] = append(childrenIndices[side], idx)
	}

	for len(childrenIndices[0]) == 0 || len(childrenIndices[1]) == 0 {
		childrenIndices[0] = []int{}
		childrenIndices[1] = []int{}
		for z := 0; z < g.dim; z++ {
			m.v[z] = 0.0
		}
		for _, idx := range indices {
			side := g.random.flip()
			childrenIndices[side] = append(childrenIndices[side], idx)
		}
	}

	var flip int
	if len(childrenIndices[0]) > len(childrenIndices[1]) {
		flip = 1
	}

	m.save()
	for side := 0; side < 2; side++ {
		m.children[side^flip] = g.makeTree(root, m.id, childrenIndices[side^flip])
	}
	m.save()

	return m.id
}

type buildArgs struct {
	action int
	id     int
	w      []float64
	result chan error
}

func (g *GannoyIndex) builder() {
	for args := range g.buildChan {
		switch args.action {
		case ADD:
			args.result <- g.addItem(args.id, args.w)
		case DELETE:
			args.result <- g.removeItem(args.id)
		case UPDATE:
			err := g.removeItem(args.id)
			if err != nil {
				args.result <- err
			} else {
				args.result <- g.addItem(args.id, args.w)
			}
		}
	}
}

func (g GannoyIndex) walk(root int, node Node, id, tab int) {
	for i := 0; i < tab*2; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%d [%d] (%d) [nDescendants: %d, v: %v]\n", id, g.maps.getId(id), node.parents[root], node.nDescendants, node.v)
	if !node.isLeaf() {
		for _, child := range node.children {
			g.walk(root, g.nodes.getNode(child), child, tab+1)
		}
	}
}
