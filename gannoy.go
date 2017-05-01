package gannoy

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"

	"github.com/gansidui/priority_queue"
)

type GannoyIndex struct {
	meta      meta
	tree      int
	dim       int
	distance  Distance
	random    Random
	nodes     Nodes
	K         int
	numWorker int
	buildChan chan buildArgs
}

func NewGannoyIndex(metaFile string, distance Distance, random Random) (GannoyIndex, error) {

	meta, err := loadMeta(metaFile)
	if err != nil {
		return GannoyIndex{}, err
	}
	tree := meta.tree
	dim := meta.dim
	K := meta.K

	ann := meta.treePath()

	gannoy := GannoyIndex{
		meta:      meta,
		tree:      tree,
		dim:       dim,
		distance:  distance,
		random:    random,
		K:         K,
		nodes:     newNodes(ann, tree, dim, K),
		numWorker: numWorker(tree),
		buildChan: make(chan buildArgs, 1),
	}
	go gannoy.builder()
	return gannoy, nil
}

func (g GannoyIndex) Tree() {
	for i, root := range g.meta.roots() {
		n, err := g.nodes.getNode(root)
		if err != nil {
			fmt.Printf("%v\n", err)
			break
		}
		g.printTree(i, n, root, 0)
	}
}

func (g *GannoyIndex) AddItem(key int, w []float64) error {
	args := buildArgs{action: ADD, key: key, w: w, result: make(chan error)}
	g.buildChan <- args
	return <-args.result
}

func (g *GannoyIndex) RemoveItem(key int) error {
	args := buildArgs{action: DELETE, key: key, result: make(chan error)}
	g.buildChan <- args
	return <-args.result
}

func (g *GannoyIndex) UpdateItem(key int, w []float64) error {
	args := buildArgs{action: UPDATE, key: key, w: w, result: make(chan error)}
	g.buildChan <- args
	return <-args.result
}

func (g *GannoyIndex) GetNnsByKey(key, n, searchK int) ([]int, error) {
	m, err := g.nodes.getNodeByKey(key)
	if err != nil || !m.isLeaf() {
		return []int{}, fmt.Errorf("Not found")
	}
	return g.getAllNns(m.v, n, searchK)
}

func (g *GannoyIndex) getAllNns(v []float64, n, searchK int) ([]int, error) {
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

		nd, err := g.nodes.getNode(i)
		if err != nil {
			return []int{}, err
		}
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
		node, err := g.nodes.getNode(j)
		if err != nil {
			return []int{}, err
		}
		nnsDist = append(nnsDist, sorter{value: g.distance.distance(v, node.v, g.dim), id: node.key})
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

	return result, nil
}

func (g *GannoyIndex) addItem(key int, w []float64) error {
	n := g.nodes.newNode()
	n.key = key
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

	for i := 0; i < g.numWorker; i++ {
		go worker(n)
	}

	for index, _ := range g.meta.roots() {
		buildChan <- index
	}

	wg.Wait()
	close(buildChan)
	g.nodes.maps.add(n.id, key)

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
	id := g.findBranchByVector(root, n.v)
	found, _ := g.nodes.getNode(id)
	// fmt.Printf("Found %d\n", item)

	org_parent := found.parents[index]
	if found.isBucket() && len(found.children) < g.K {
		// ノードに余裕があれば追加
		// fmt.Printf("pattern bucket\n")
		n.updateParents(index, id)
		found.nDescendants++
		found.children = append(found.children, n.id)
		found.save()
	} else {
		// ノードが上限またはリーフノードであれば新しいノードを追加
		willDelete := false
		var indices []int
		if found.isLeaf() {
			// fmt.Printf("pattern leaf node\n")
			indices = []int{id, n.id}
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
			parent, _ := g.nodes.getNode(org_parent)
			parent.nDescendants++
			children := make([]int, len(parent.children))
			for i, child := range parent.children {
				if child == id {
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
			g.nodes.free.push(found.id)
		}
	}
}

func (g *GannoyIndex) removeItem(key int) error {
	n, err := g.nodes.getNodeByKey(key)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(g.tree)
	buildChan := make(chan int, g.tree)
	worker := func(n Node) {
		for root := range buildChan {
			g.remove(root, n)
			wg.Done()
		}
	}

	for i := 0; i < g.numWorker; i++ {
		go worker(n)
	}
	for index, _ := range g.meta.roots() {
		buildChan <- index
	}

	wg.Wait()
	close(buildChan)

	g.nodes.maps.remove(key)
	n.destroy()
	g.nodes.free.push(n.id)

	return nil
}

func (g *GannoyIndex) remove(root int, node Node) {
	if node.isRoot(root) {
		g.meta.updateRoot(root, -1)
		return
	}
	parent, _ := g.nodes.getNode(node.parents[root])
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
			grandParent, _ := g.nodes.getNode(parent.parents[root])
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

		otherNode, _ := g.nodes.getNode(other)
		otherNode.updateParents(root, parent.parents[root])

		parent.destroy()
		g.nodes.free.push(parent.id)
	}
}

func (g GannoyIndex) findBranchByVector(id int, v []float64) int {
	node, _ := g.nodes.getNode(id)
	if node.isLeaf() || node.isBucket() {
		return id
	}
	side := g.distance.side(node, v, g.dim, g.random)
	return g.findBranchByVector(node.children[side], v)
}

func (g *GannoyIndex) makeTree(root, parent int, ids []int) int {
	if len(ids) == 1 {
		n, _ := g.nodes.getNode(ids[0])
		if len(n.parents) == 0 {
			n.parents = make([]int, g.tree)
		}
		n.updateParents(root, parent)
		return ids[0]
	}

	if len(ids) <= g.K {
		m := g.nodes.newNode()
		m.parents = make([]int, g.tree)
		m.nDescendants = len(ids)
		m.parents[root] = parent
		m.children = ids
		m.save()
		for _, child := range ids {
			c, _ := g.nodes.getNode(child)
			if len(c.parents) == 0 {
				c.parents = make([]int, g.tree)
			}
			c.updateParents(root, m.id)
		}
		return m.id
	}

	children := make([]Node, len(ids))
	for i, id := range ids {
		children[i], _ = g.nodes.getNode(id)
	}

	childrenIds := [2][]int{[]int{}, []int{}}

	m := g.nodes.newNode()
	m.parents = make([]int, g.tree)
	m.nDescendants = len(ids)
	m.parents[root] = parent

	m = g.distance.createSplit(children, g.dim, g.random, m)
	for _, id := range ids {
		n, _ := g.nodes.getNode(id)
		side := g.distance.side(m, n.v, g.dim, g.random)
		childrenIds[side] = append(childrenIds[side], id)
	}

	for len(childrenIds[0]) == 0 || len(childrenIds[1]) == 0 {
		childrenIds[0] = []int{}
		childrenIds[1] = []int{}
		for z := 0; z < g.dim; z++ {
			m.v[z] = 0.0
		}
		for _, id := range ids {
			side := g.random.flip()
			childrenIds[side] = append(childrenIds[side], id)
		}
	}

	var flip int
	if len(childrenIds[0]) > len(childrenIds[1]) {
		flip = 1
	}

	m.save()
	for side := 0; side < 2; side++ {
		m.children[side^flip] = g.makeTree(root, m.id, childrenIds[side^flip])
	}
	m.save()

	return m.id
}

type buildArgs struct {
	action int
	key    int
	w      []float64
	result chan error
}

func (g *GannoyIndex) builder() {
	for args := range g.buildChan {
		switch args.action {
		case ADD:
			args.result <- g.addItem(args.key, args.w)
		case DELETE:
			args.result <- g.removeItem(args.key)
		case UPDATE:
			err := g.removeItem(args.key)
			if err != nil {
				args.result <- err
			} else {
				args.result <- g.addItem(args.key, args.w)
			}
		}
	}
}

func (g GannoyIndex) printTree(root int, node Node, id, tab int) {
	for i := 0; i < tab*2; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%d [%d] (%d) [nDescendants: %d, v: %v]\n", id, node.key, node.parents[root], node.nDescendants, node.v)
	if !node.isLeaf() {
		for _, child := range node.children {
			n, err := g.nodes.getNode(child)
			if err != nil {
				fmt.Printf("%v\n", err)
				break
			}
			g.printTree(root, n, child, tab+1)
		}
	}
}

func numWorker(tree int) int {
	procs := runtime.GOMAXPROCS(0) // current setting
	if tree < procs {
		return tree
	} else {
		return procs
	}
}
