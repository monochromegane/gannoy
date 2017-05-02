package gannoy

import (
	"os"
	"testing"
)

func TestGannoyIndexNotFound(t *testing.T) {
	name := "test_gannoy_index_not_found.tree"
	defer os.Remove(name)
	_, err := NewGannoyIndex("not_found.meta", Angular{}, RandRandom{})
	if err == nil {
		t.Errorf("NewGannoyIndex with not exist meta file should return error.")
	}

}

func TestGannoyIndexAttribute(t *testing.T) {
	tree := 2
	dim := 3
	K := 4
	meta := "test_update_root"
	CreateMeta(".", meta, tree, dim, K)
	defer os.Remove(meta + ".meta")

	name := "test_gannoy_index.tree"
	defer os.Remove(name)
	gannoy, _ := NewGannoyIndex(meta+".meta", Angular{}, RandRandom{})

	if gannoy.tree != tree {
		t.Errorf("NewGannoyIndex should contain tree %d, but %d", tree, gannoy.tree)
	}
	if gannoy.dim != dim {
		t.Errorf("NewGannoyIndex should contain dim %d, but %d", dim, gannoy.dim)
	}
	if gannoy.K != K {
		t.Errorf("NewGannoyIndex should contain K %d, but %d", K, gannoy.K)
	}
}

func TestGannoyIndexAddItem(t *testing.T) {
	// first item (be root)
	// second item (change root and make tree)
	// add item in bucket
	// add item in full bucket
	// add item in leaf node
}

func TestGannoyIndexRemoveItem(t *testing.T) {
	// remove item from bucket
	// remove item from single bucket
	// remove item from single bucket and root
}

func TestGannoyIndexUpdateItem(t *testing.T) {
	// remove and add item
}

func TestGannoyIndexGetNnsByKey(t *testing.T) {
	// search nns (from builded tree file)
}
