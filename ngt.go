package gannoy

import (
	"fmt"

	ngt "github.com/monochromegane/go-ngt"
)

type NGTIndex struct {
	database string
	index    ngt.NGTIndex
}

func NewNGTIndex(database string) (NGTIndex, error) {
	index, err := ngt.OpenIndex(database)
	if err != nil {
		return NGTIndex{}, err
	}
	return NGTIndex{database: database, index: index}, nil
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

func (idx NGTIndex) UpdateItem(id uint, v []float64) (uint, error) {
	err := idx.RemoveItem(id)
	if err != nil {
		return 0, err
	}
	return idx.AddItem(v)
}

func (idx NGTIndex) AddItem(v []float64) (uint, error) {
	newId, err := idx.addItem(v)
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

func (idx NGTIndex) RemoveItem(id uint) error {
	if !idx.existItem(id) {
		return fmt.Errorf("Not Found")
	}
	err := idx.removeItem(id)
	if err != nil {
		return err
	}
	return idx.index.SaveIndex(idx.database)
}

func (idx NGTIndex) addItem(v []float64) (uint, error) {
	return idx.index.InsertIndex(v)
}

func (idx NGTIndex) removeItem(id uint) error {
	return idx.index.RemoveIndex(id)
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
