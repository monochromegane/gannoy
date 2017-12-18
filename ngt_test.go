package gannoy

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

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

	// Apply in memory
	result := index.applyFromBinLog()
	if result.Err != nil {
		t.Errorf("NGTIndex.applyFromBinLog should not return error, but return %v", result.Err)
	}

	// Check
	exist := result.Index.existItem(uint(key))
	if !exist {
		t.Errorf("NGTIndex.existItem should return true, but return false")
	}

	// Apply to file
	err = index.ApplyToDB(result)
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

	result = newIndex.applyFromBinLog()
	if result.Err != nil {
		t.Errorf("NGTIndex.applyFromBinLog should not return error, but return %v", result.Err)
	}

	keys, err = result.Index.SearchItem(uint(key), 1, 0.1)
	if err == nil {
		t.Errorf("NGTIndex.RemoveItem should delete object, but dose not delete.")
	}
}

func TestNGTIndexWaitApplyFromBinLog(t *testing.T) {
	timeout := 30 * time.Second
	idx := testCreateGraphAndTree("db", 5)
	defer idx.Drop()
	idx.Save()

	index, err := NewNGTIndex(idx.database, 1, timeout)
	if err != nil {
		t.Errorf("NewNGTIndex should not return error, but return %v", err)
	}
	defer index.Drop()

	// Wait applyFromBinLog
	resultCh := make(chan ApplicationResult)
	defer close(resultCh)
	index.WaitApplyFromBinLog(1*time.Second, resultCh)

	// Add
	key := 1
	features := []byte(`{"features":[0.0,0.0,0.0,0.0,0.0]}`)
	err = index.UpdateBinLog(key, UPDATE, features)
	if err != nil {
		t.Errorf("NGTIndex.UpdateBinLog should not return error, but return %v", err)
	}

	// Check finish goroutine
	ch := make(chan struct{})
	defer close(ch)
	go func() {
		exit := <-index.exitCh
		ch <- exit
	}()

	// Auto apply
	for result := range resultCh {
		keys, err := result.Index.SearchItem(uint(key), 1, 0.1)
		if err != nil || len(keys) != 1 || keys[0] != key {
			t.Errorf("NGTIndex.WaitApplyFromBinLog should register object, but dose not register.")
		} else {
			break
		}
	}

	<-ch
}

func TestNGTIndexCancelWaitApplyFromBinLog(t *testing.T) {
	timeout := 30 * time.Second
	idx := testCreateGraphAndTree("db", 5)
	defer idx.Drop()
	idx.Save()

	index, err := NewNGTIndex(idx.database, 1, timeout)
	if err != nil {
		t.Errorf("NewNGTIndex should not return error, but return %v", err)
	}
	defer index.Drop()

	// Wait applyFromBinLog
	resultCh := make(chan ApplicationResult)
	defer close(resultCh)
	index.WaitApplyFromBinLog(10*time.Second, resultCh)

	// Add
	key := 1
	features := []byte(`{"features":[0.0,0.0,0.0,0.0,0.0]}`)
	err = index.UpdateBinLog(key, UPDATE, features)
	if err != nil {
		t.Errorf("NGTIndex.UpdateBinLog should not return error, but return %v", err)
	}

	// Check finish goroutine
	ch := make(chan struct{})
	defer close(ch)
	go func() {
		exit := <-index.exitCh
		ch <- exit
	}()
	index.cancel()
	<-ch
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
