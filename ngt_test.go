package gannoy

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	ngt "github.com/yahoojapan/gongt"
)

func TestCreateGraphAndTree(t *testing.T) {
	database := tempDatabaseDir("db")
	index, err := CreateGraphAndTree(database, ngt.New(database))
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

func TestNGTIndexString(t *testing.T) {
	dbname := "db"
	index := testCreateGraphAndTree(dbname, 1)
	defer index.Drop()

	s := index.String()
	if s != dbname {
		t.Errorf("NGTIndex.String should return %s, but return %s", dbname, s)
	}
}

func TestNGTIndexAddAndDeleteItemByApplyBinlog(t *testing.T) {
	timeout := 30 * time.Second
	idx := testCreateGraphAndTree("db", 5)
	defer idx.Drop()
	idx.Save()

	index, err := NewNGTIndex(idx.database, 1, timeout)
	if err != nil {
		t.Errorf("NewNGTIndex should not return error, but return %v", err)
	}
	defer index.Drop()

	// Add
	key := 1
	features := []byte(`{"features":[0.0,0.0,0.0,0.0,0.0]}`)
	err = index.UpdateBinLog(key, UPDATE, features)
	if err != nil {
		t.Errorf("NGTIndex.UpdateBinLog should not return error, but return %v", err)
	}

	// Apply to file
	err = index.Apply()
	if err != nil {
		t.Errorf("NGTIndex.ApplyToDB should not return error, but return %v", err)
	}

	// Open as new index
	newIndex, err := NewNGTIndex(idx.database, 1, timeout)
	if err != nil {
		t.Errorf("NewNGTIndex should not return error, but return %v", err)
	}
	defer newIndex.Drop()

	// Check
	keys, err := newIndex.SearchItem(uint(key), 1, 0.1)
	if err != nil || len(keys) != 1 || keys[0] != key {
		t.Errorf("NGTIndex.AddItem should register object, but dose not register.")
	}

	// Remove
	err = newIndex.UpdateBinLog(1, DELETE, []byte{})
	if err != nil {
		t.Errorf("NGTIndex.UpdateBinLog should not return error, but return %v", err)
	}

	err = newIndex.Apply()
	if err != nil {
		t.Errorf("NGTIndex.applyFromBinLog should not return error, but return %v", err)
	}

	newIndex2, err := NewNGTIndex(idx.database, 1, timeout)
	if err != nil {
		t.Errorf("NewNGTIndex should not return error, but return %v", err)
	}
	defer newIndex2.Drop()

	keys, err = newIndex2.SearchItem(uint(key), 1, 0.1)
	if err == nil {
		t.Errorf("NGTIndex.RemoveItem should delete object, but dose not delete.")
	}
}

func testCreateGraphAndTree(database string, dim int) NGTIndex {
	path := tempDatabaseDir(database)
	index, _ := CreateGraphAndTree(path, ngt.New(path).SetDimension(dim))
	return index
}

func tempDatabaseDir(database string) string {
	dir, _ := ioutil.TempDir("", "gannoy-test")
	return filepath.Join(dir, database)
}
