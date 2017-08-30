package gannoy

import (
	"fmt"

	ngt "github.com/monochromegane/go-ngt"
)

type NGTIndex struct {
	database  string
	index     ngt.NGTIndex
	buildChan chan buildArgs
}

func NewNGTIndex(database string) (NGTIndex, error) {
	index, err := ngt.OpenIndex(database)
	if err != nil {
		return NGTIndex{}, err
	}
	idx := NGTIndex{
		database:  database,
		index:     index,
		buildChan: make(chan buildArgs, 1),
	}
	go idx.builder()
	return idx, nil
}

func (idx NGTIndex) GetNnsById(id uint, n int, epsilon float32) ([]int, error) {
	v, err := idx.getItem(id)
	if err != nil {
		return []int{}, err
	}
	return idx.GetAllNns(v, n, epsilon)
}

func (idx NGTIndex) GetAllNns(v []float64, n int, epsilon float32) ([]int, error) {
	results, err := idx.index.Search(v, n, epsilon)
	ids := make([]int, len(results))
	for i, result := range results {
		ids[i] = int(result.Id)
	}
	return ids, err
}

func (idx NGTIndex) builder() {
	for args := range idx.buildChan {
		switch args.action {
		case ADD:
			_, err := idx.addItem(args.w)
			args.result <- err
		case DELETE:
			args.result <- idx.removeItem(uint(args.key))
		}
	}
}

func (idx NGTIndex) AddItem(key int, w []float64) error {
	args := buildArgs{action: ADD, key: key, w: w, result: make(chan error)}
	idx.buildChan <- args
	return <-args.result
}

func (idx NGTIndex) RemoveItem(key int) error {
	args := buildArgs{action: DELETE, key: key, result: make(chan error)}
	idx.buildChan <- args
	return <-args.result
}

func (idx NGTIndex) addItem(v []float64) (uint, error) {
	newId, err := idx.index.InsertIndex(v)
	fmt.Printf("newId: %d\n", newId)
	if err != nil {
		return 0, err
	}

	err = idx.index.CreateIndex(24)
	if err != nil {
		return 0, err
	}
	return newId, idx.index.SaveIndex(idx.database)
}

func (idx NGTIndex) removeItem(id uint) error {
	if !idx.existItem(id) {
		return fmt.Errorf("Not Found")
	}
	err := idx.index.RemoveIndex(id)
	if err != nil {
		return err
	}
	return idx.index.SaveIndex(idx.database)
}

func (idx NGTIndex) getItem(id uint) ([]float64, error) {
	o, err := idx.index.GetObjectSpace()
	if err != nil {
		return []float64{}, err
	}

	obj, err := o.GetObjectAsFloat(int(id))
	if err != nil {
		return []float64{}, err
	}
	v := make([]float64, len(obj))
	for i, o := range obj {
		v[i] = float64(o)
	}
	return v, nil
}

func (idx NGTIndex) existItem(id uint) bool {
	obj, err := idx.getItem(id)
	if err != nil || len(obj) == 0 {
		return false
	}
	return true
}
