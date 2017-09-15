package gannoy

import (
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"runtime"
	"sync"
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

func TestAddItemAndRemoveItem(t *testing.T) {
	index := testCreateGraphAndTree("db", 1)
	defer index.Drop()
	index.Save()

	index, _ = NewNGTIndex(index.database, 1, 10*time.Second)
	key := 100
	err := index.AddItem(key, []float64{1.0})
	if err != nil {
		t.Errorf("NGTIndex.AddItem should not return error, but return %v", err)
	}
	err = index.AddItem(key+1, []float64{3.0})
	if err != nil {
		t.Errorf("NGTIndex.AddItem should not return error, but return %v", err)
	}

	keys, err := index.SearchItem(uint(key), 2, 0.1)
	if keys[0] != key {
		t.Errorf("NGTIndex.AddItem should register object, but dose not register.")
	}

	err = index.RemoveItem(key)
	if err != nil {
		t.Errorf("NGTIndex.RemoveItem should not return error, but return %v", err)
	}

	keys, err = index.SearchItem(uint(key), 1, 0.1)
	if err == nil {
		t.Errorf("NGTIndex.RemoveItem should delete object, but dose not delete.")
	}
}

func TestUpdateItemWhenNotExist(t *testing.T) {
	index := testCreateGraphAndTree("db", 1)
	defer index.Drop()
	index.Save()

	index, _ = NewNGTIndex(index.database, 1, 10*time.Second)

	key := 100
	err := index.UpdateItem(key, []float64{1.0})
	if err != nil {
		t.Errorf("NGTIndex.UpdateItem should not return error, but return %v", err)
	}

	keys, err := index.SearchItem(uint(key), 1, 0.1)
	if keys[0] != key {
		t.Errorf("NGTIndex.AddItem should register object, but dose not register.")
	}
}

func TestUpdateItemWhenExist(t *testing.T) {
	index := testCreateGraphAndTree("db", 1)
	defer index.Drop()
	index.Save()

	index, _ = NewNGTIndex(index.database, 1, 10*time.Second)

	key := 100
	err := index.UpdateItem(key, []float64{1.0})
	if err != nil {
		t.Errorf("NGTIndex.UpdateItem should not return error, but return %v", err)
	}

	keys, err := index.SearchItem(uint(key), 1, 0.1)
	if keys[0] != key {
		t.Errorf("NGTIndex.AddItem should register object, but dose not register.")
	}

	err = index.AddItem(key+1, []float64{0.1}) // Avoid stopping the NGT when there are 0 objects.
	if err != nil {
		t.Errorf("NGTIndex.AddItem should not return error, but return %v", err)
	}
	err = index.UpdateItem(key, []float64{0.2})
	if err != nil {
		t.Errorf("NGTIndex.UpdateItem should not return error, but return %v", err)
	}

	keys, err = index.SearchItem(uint(key), 10, 0.1)
	if len(keys) > 3 {
		t.Errorf("NGTIndex.AddItem should update object, but inserted new one.")
	}
	if keys[0] != key {
		t.Errorf("NGTIndex.AddItem should register object, but dose not register.")
	}
}

func TestUpdateItemWithSearch(t *testing.T) {
	objectNum := 100
	dim := 2048
	index := testCreateGraphAndTree("db", dim)

	// Insert objects
	vecs := testRandomVectors(objectNum, dim)
	for i := 0; i < objectNum; i++ {
		_, err := index.addItemWithoutCreateIndex(i, vecs[i])
		if err != nil {
			t.Errorf("NGTIndex.UpdateItem should not return error, but return %v", err)
		}
	}
	err := index.index.CreateIndex(runtime.NumCPU())
	if err != nil {
		t.Errorf("NGTIndex.CreateItem should not return error, but return %v", err)
	}
	index.Save()
	file := index.database
	index, _ = NewNGTIndex(file, runtime.NumCPU(), 10*time.Second) // Avoid warining GraphAndTreeIndex::insert empty
	// defer index.Drop()

	go func() {
		for {
			index.SearchItem(uint(rand.Intn(objectNum)), objectNum, 0.1)
		}
	}()

	// UpdateItem concurrently
	vecs = testRandomVectors(objectNum, dim)
	var wg sync.WaitGroup
	wg.Add(objectNum)
	worker := func(inputChan chan int) {
		for key := range inputChan {
			index.UpdateItem(key, vecs[key])
			wg.Done()
		}
	}
	inputChan := make(chan int, objectNum)
	for i := 0; i < runtime.NumCPU(); i++ {
		go worker(inputChan)
	}

	for i := 0; i < objectNum; i++ {
		inputChan <- i
	}
	wg.Wait()
	close(inputChan)

	key := 0
	keys, err := index.SearchItem(uint(key), objectNum+10, 0.1)
	if len(keys) != objectNum {
		t.Errorf("NGTIndex.AddItem should update object, but inserted new one.")
	}
}

func TestUpdateItemAndRemoveItemConcurrently(t *testing.T) {
	objectNum := 50
	dim := 2048
	index := testCreateGraphAndTree("db", dim)

	// Insert objects
	vecs := testRandomVectors(objectNum, dim)
	for i := 0; i < objectNum; i++ {
		_, err := index.addItemWithoutCreateIndex(i, vecs[i])
		if err != nil {
			t.Errorf("NGTIndex.UpdateItem should not return error, but return %v", err)
		}
	}
	err := index.index.CreateIndex(runtime.NumCPU())
	if err != nil {
		t.Errorf("NGTIndex.CreateItem should not return error, but return %v", err)
	}
	index.Save()
	file := index.database
	index, _ = NewNGTIndex(file, runtime.NumCPU(), 10*time.Second) // Avoid warining GraphAndTreeIndex::insert empty
	defer index.Drop()

	// UpdateItem concurrently
	vecs = testRandomVectors(objectNum, dim)
	var wg sync.WaitGroup
	wg.Add(objectNum)
	worker := func(inputChan chan int) {
		for key := range inputChan {
			index.UpdateItem(key, vecs[key])
			wg.Done()
		}
	}
	inputChan := make(chan int, objectNum)
	for i := 0; i < runtime.NumCPU(); i++ {
		go worker(inputChan)
	}

	for i := 0; i < objectNum; i++ {
		inputChan <- i
	}
	wg.Wait()
	close(inputChan)

	key := 0
	keys, err := index.SearchItem(uint(key), objectNum+10, 0.1)
	if len(keys) != objectNum {
		t.Errorf("NGTIndex.AddItem should update object, but inserted new one.")
	}

	// UpdateItem(new Item) concurrently
	vecs = testRandomVectors(objectNum, dim)
	inputChan2 := make(chan int, objectNum)
	var wg2 sync.WaitGroup
	wg2.Add(objectNum)
	worker2 := func(inputChan2 chan int) {
		for key := range inputChan2 {
			index.UpdateItem(key, vecs[key-objectNum])
			wg2.Done()
		}
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		go worker2(inputChan2)
	}
	for i := 0; i < objectNum; i++ {
		inputChan2 <- i + objectNum
	}
	wg2.Wait()
	close(inputChan2)
	keys, err = index.SearchItem(uint(key), objectNum*2+1, 0.1)
	if len(keys) != objectNum*2 {
		t.Errorf("NGTIndex.AddItem should insert object, but updated new one.")
	}

	// RemoveItem concurrently
	inputChan3 := make(chan int, objectNum)
	var wg3 sync.WaitGroup
	wg3.Add(objectNum)
	worker3 := func(inputChan3 chan int) {
		for key := range inputChan3 {
			index.RemoveItem(key)
			wg3.Done()
		}
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		go worker3(inputChan3)
	}
	for i := 0; i < objectNum; i++ {
		inputChan3 <- i
	}
	wg3.Wait()
	close(inputChan3)

	keys, err = index.SearchItem(uint(key+objectNum), objectNum+1, 0.1)
	if len(keys) != objectNum {
		t.Errorf("NGTIndex.AddItem should update object, but inserted new one.")
	}
}

func testRandomVectors(objectNum, dim int) [][]float64 {
	vecs := make([][]float64, objectNum)
	for i := 0; i < objectNum; i++ {
		vecs[i] = testRandomVector(dim)
	}
	return vecs
}

func testRandomVector(dim int) []float64 {
	vec := make([]float64, dim)
	for j := 0; j < dim; j++ {
		vec[j] = rand.Float64()
	}
	return vec
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
