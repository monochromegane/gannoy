package gannoy

import (
	"os"
	"testing"
)

func TestCreateMetaAlreadyExist(t *testing.T) {
	meta := "test_create_meta_already_exist"
	os.Create(meta + ".meta")
	defer os.Remove(meta + ".meta")

	err := CreateMeta(".", meta, 1, 1, 1)
	if err == nil {
		t.Errorf("CreateMeta when already exist should return error.")
	}
}

func TestLoadMeta(t *testing.T) {
	file := "test_load_meta"

	tree := 2
	dim := 3
	K := 4

	CreateMeta(".", file, tree, dim, K)
	defer os.Remove(file + ".meta")

	meta, err := loadMeta(file + ".meta")
	if err != nil {
		t.Errorf("LoadMeta should not return error.")
	}

	if meta.tree != tree {
		t.Errorf("tree should be %d, but %d.", tree, meta.tree)
	}
	if meta.dim != dim {
		t.Errorf("dim should be %d, but %d.", dim, meta.dim)
	}
	if meta.K != K {
		t.Errorf("K should be %d, but %d.", K, meta.K)
	}

	roots := meta.roots()
	if len(roots) != tree {
		t.Errorf("roots size should be %d, but %d.", tree, len(roots))
	}
	for _, root := range roots {
		if root != -1 {
			t.Errorf("initialized roots value should be -1, but %d", root)
			break
		}
	}
}

func TestUpdateRoot(t *testing.T) {
	file := "test_update_root"

	tree := 2
	dim := 3
	K := 4

	CreateMeta(".", file, tree, dim, K)
	defer os.Remove(file + ".meta")

	meta, _ := loadMeta(file + ".meta")
	meta.updateRoot(0, 10)

	expects := []int{10, -1}
	for i, root := range meta.roots() {
		if root != expects[i] {
			t.Errorf("Updated root should be %d, but %d.", expects[i], root)
			break
		}
	}
}
