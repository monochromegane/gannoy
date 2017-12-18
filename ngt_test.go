package gannoy

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	ngt "github.com/monochromegane/go-ngt"
)

func TestCreateGraphAndTree(t *testing.T) {
	property, _ := ngt.NewNGTProperty(1)
	defer property.Free()

	index, err := CreateGraphAndTree(tempDatabaseDir("db"), property)
	if err != nil {
		t.Errorf("CreateGraphAndTree should not return error, but return %v", err)
	}
	defer index.Drop()
}

func TestNewNGTIndex(t *testing.T) {
	index := testCreateGraphAndTree("db", 1)
	defer index.Drop()
	index.Save()

	_, err := NewNGTIndex(index.database, 1, 1)
	if err != nil {
		t.Errorf("NewNGTIndex should not return error, but return %v", err)
	}
}

func testCreateGraphAndTree(database string, dim int) NGTIndex {
	property, _ := ngt.NewNGTProperty(dim)
	defer property.Free()
	index, _ := CreateGraphAndTree(tempDatabaseDir(database), property)
	return index
}

func tempDatabaseDir(database string) string {
	dir, _ := ioutil.TempDir("", "gannoy-test")
	return filepath.Join(dir, database)
}
