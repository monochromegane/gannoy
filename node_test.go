package gannoy

import (
	"os"
	"testing"
)

func TestNewNodeAtFirst(t *testing.T) {
	name := "test_new_node_at_first.tree"
	defer os.Remove(name)
	nodes := newNodes(name, 2, 3, 4)

	if len(nodes.free.free) != 0 {
		t.Errorf("Initialized nodes.free size should be 0, but %d", len(nodes.free.free))
	}
	if len(nodes.maps.keyToId) != 0 {
		t.Errorf("Initialized nodes.maps size should be 0, but %d", len(nodes.maps.keyToId))
	}
}

func TestNewNodeMpas(t *testing.T) {
	name := "test_new_node_maps.tree"
	defer os.Remove(name)
	nodes := newNodes(name, 2, 3, 4)

	// Create
	node := nodes.newNode()
	node.key = 10
	node.parents = []int{2, 3}
	node.v = []float64{1.1, 1.2, 1.3}
	node.save()

	nodes = newNodes(name, 2, 3, 4)
	id, err := nodes.maps.getId(10)
	if err != nil {
		t.Errorf("nodes.maps should not return error.")
	}
	if node.id != id {
		t.Errorf("nodes.maps should contain map for id: %d, but %d", id, node.id)
	}
}

func TestNewNodeFree(t *testing.T) {
	name := "test_new_node_free.tree"
	defer os.Remove(name)
	nodes := newNodes(name, 2, 3, 4)

	// Create
	node := nodes.newNode()
	node.key = 10
	node.parents = []int{2, 3}
	node.v = []float64{1.1, 1.2, 1.3}
	node.save()
	// Found and remove
	node, _ = nodes.getNode(node.id)
	node.destroy()

	nodes = newNodes(name, 2, 3, 4)
	newNode := nodes.newNode() // from free node list.
	if node.id != newNode.id {
		t.Errorf("nodes.free should contain free node: %d, but %d", newNode.id, node.id)
	}
}

func TestNodeSaveNew(t *testing.T) {
	name := "test_node_save_new.tree"
	defer os.Remove(name)
	nodes := newNodes(name, 2, 3, 4)

	// Create
	node := nodes.newNode()
	node.key = 10
	node.parents = []int{2, 3}
	node.v = []float64{1.1, 1.2, 1.3}
	err := node.save()
	if err != nil {
		t.Errorf("node save should not return error.")
	}

	if node.id == -1 {
		t.Errorf("node save should set id.")
	}
	if node.isNewRecord {
		t.Errorf("node save should set isNewRecord to false.")
	}
}

func TestNodeSaveUpdate(t *testing.T) {
	name := "test_node_save_update.tree"
	defer os.Remove(name)
	nodes := newNodes(name, 2, 3, 4)

	// Create
	node := nodes.newNode()
	node.key = 10
	node.parents = []int{2, 3}
	node.v = []float64{1.1, 1.2, 1.3}
	node.save()
	// Update
	found, _ := nodes.Find(node.id)
	found.v = []float64{2.1, 2.2, 2.3}
	err := found.save()
	if err != nil {
		t.Errorf("node save should not return error.")
	}
}

func TestNodeDestroy(t *testing.T) {
	name := "test_node_destroy.tree"
	defer os.Remove(name)
	nodes := newNodes(name, 2, 3, 4)

	// Create
	node := nodes.newNode()
	node.key = 10
	node.parents = []int{2, 3}
	node.v = []float64{1.1, 1.2, 1.3}
	node.save()

	// Destroy
	found, _ := nodes.Find(node.id)
	err := found.destroy()
	if err != nil {
		t.Errorf("node destroy should not return error.")
	}
	if !found.free {
		t.Errorf("node destroy should set free to true.")
	}
}

func TestNodeIsLeaf(t *testing.T) {
	// Leaf node
	node := Node{
		key:          10,
		nDescendants: 1,
		parents:      []int{2, 3},
		v:            []float64{1.1, 1.2, 1.3},
	}
	if !node.isLeaf() {
		t.Errorf("node should be leaf node.")
	}
}

func TestNodeIsBucket(t *testing.T) {
	// Bucket node
	node := Node{
		key:          20,
		nDescendants: 3,
		parents:      []int{2, 3},
		children:     []int{5, 6, 7},
	}
	if !node.isBucket() {
		t.Errorf("node should be bucket node.")
	}
}

func TestNodeIsRoot(t *testing.T) {
	// Root node
	parents := []int{-1, -1}
	node := Node{
		key:          10,
		nDescendants: 1,
		parents:      parents,
		v:            []float64{1.1, 1.2, 1.3},
	}
	for i, _ := range parents {
		if !node.isRoot(i) {
			t.Errorf("node should be root node.")
		}
	}
}
