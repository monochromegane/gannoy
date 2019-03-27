package gannoy

import (
	"os"
	"testing"
)

func TestFileCreateAndFind(t *testing.T) {
	name := "test_file_create_and_find.tree"
	defer os.Remove(name)
	file := newFile(name, 2, 3, 6)

	nodes := []Node{
		// Leaf node
		Node{
			key:          10,
			nDescendants: 1,
			parents:      []int{2, 3},
			children:     []int{0, 0},
			v:            []float64{1.1, 1.2, 1.3},
		},
		// Bucket node
		Node{
			key:          20,
			nDescendants: 3,
			parents:      []int{2, 3},
			children:     []int{5, 6, 7},
		},
		// Branch node
		Node{
			key:          30,
			nDescendants: 2,
			parents:      []int{2, 3},
			children:     []int{5, 6},
			v:            []float64{1.1, 1.2, 1.3},
		},
	}

	// Create
	for i, node := range nodes {
		id, err := file.Create(node)
		if id != i {
			t.Errorf("File create should return id: %d, but %d", i, id)
		}
		if err != nil {
			t.Errorf("File create should not return error.")
		}
	}

	// Find
	for id, node := range nodes {
		found, err := file.Find(id)
		if err != nil {
			t.Errorf("File find should not return error.")
		}
		if found.id != id {
			t.Errorf("File find should return created node with id %d, but %d", found.id, id)
		}
		if found.key != node.key {
			t.Errorf("File find should return created node with key %d, but %d", found.key, node.key)
		}
		if found.nDescendants != node.nDescendants {
			t.Errorf("File find should return created node with nDescendants %d, but %d", found.nDescendants, node.nDescendants)
		}
		for i, parent := range found.parents {
			if parent != node.parents[i] {
				t.Errorf("File find should return created node with parents %v, but %v", found.parents, node.parents)
			}
		}
		for i, child := range found.children {
			if child != node.children[i] {
				t.Errorf("File find should return created node with children %v, but %v", found.children, node.children)
			}
		}
		for i, v := range found.v {
			if v != node.v[i] {
				t.Errorf("File find should return created node with v %v, but %v", found.v, node.v)
			}
		}
		//if found.free != node.free {
		//	t.Errorf("File find should return created node with free %d, but %d", found.free, node.free)
		//}
	}
}

func TestFileUpdate(t *testing.T) {
	name := "test_file_update.tree"
	defer os.Remove(name)
	file := newFile(name, 2, 3, 4)

	node := Node{
		key:          10,
		nDescendants: 1,
		parents:      []int{2, 3},
		v:            []float64{1.1, 1.2, 1.3},
	}

	// Create
	id, _ := file.Create(node)
	// Found
	found, _ := file.Find(id)
	// Update
	found.v = []float64{2.1, 2.2, 2.3}
	err := file.Update(found)
	if err != nil {
		t.Errorf("File update should not return error.")
	}

	updated, _ := file.Find(id)
	for i, v := range updated.v {
		if v != found.v[i] {
			t.Errorf("File update should return updated node with v %v, but %v", found.v, updated.v)
		}
	}
}

func TestUpdateParent(t *testing.T) {
	name := "test_file_update_parent.tree"
	defer os.Remove(name)
	file := newFile(name, 2, 3, 4)

	node := Node{
		key:          10,
		nDescendants: 1,
		parents:      []int{2, 3},
		v:            []float64{1.1, 1.2, 1.3},
	}

	// Create
	id, _ := file.Create(node)
	// Update parent
	newParents := []int{20, 30}
	for i, newParent := range newParents {
		err := file.UpdateParent(id, i, newParent)
		if err != nil {
			t.Errorf("File update parent should not return error.")
		}
	}
	// Found
	updated, _ := file.Find(id)
	for i, newParent := range newParents {
		if updated.parents[i] != newParent {
			t.Errorf("File update parent should not return error.")
		}
	}
}

func TestFileIterate(t *testing.T) {
	name := "test_file_iterate.tree"
	defer os.Remove(name)
	file := newFile(name, 2, 3, 4)

	nodes := []Node{
		// Leaf node
		Node{
			key:          10,
			nDescendants: 1,
			parents:      []int{2, 3},
			v:            []float64{1.1, 1.2, 1.3},
		},
		// Bucket node
		Node{
			key:          20,
			nDescendants: 3,
			parents:      []int{2, 3},
			children:     []int{5, 6, 7},
		},
		// Branch node
		Node{
			key:          30,
			nDescendants: 2,
			parents:      []int{2, 3},
			children:     []int{5, 6},
			v:            []float64{1.1, 1.2, 1.3},
		},
	}

	// Create
	for _, node := range nodes {
		file.Create(node)
	}
	// Iterate
	iterator := make(chan Node)
	go file.Iterate(iterator)

	i := 0
	for node := range iterator {
		if nodes[i].key != node.key {
			t.Errorf("File iterate should return node (key: %d), but %d", nodes[i].key, node.key)
		}
		i++
	}
}
