package gannoy

import (
	"fmt"

	ngt "github.com/monochromegane/go-ngt"
)

type NGTIndex struct {
	database  string
	index     ngt.NGTIndex
	buildChan chan buildArgs
	pair      Pair
}

func CreateGraphAndTree(database string, property ngt.NGTProperty) (NGTIndex, error) {
	index, err := ngt.CreateGraphAndTree(database, property)
	if err != nil {
		return NGTIndex{}, err
	}
	pair, err := newPair(database + ".map")
	if err != nil {
		return NGTIndex{}, err
	}
	idx := NGTIndex{
		database:  database,
		index:     index,
		buildChan: make(chan buildArgs, 1),
		pair:      pair,
	}
	go idx.builder()
	return idx, nil
}

func NewNGTIndex(database string) (NGTIndex, error) {
	index, err := ngt.OpenIndex(database)
	if err != nil {
		return NGTIndex{}, err
	}
	pair, err := newPair(database + ".map")
	if err != nil {
		return NGTIndex{}, err
	}
	idx := NGTIndex{
		database:  database,
		index:     index,
		buildChan: make(chan buildArgs, 1),
		pair:      pair,
	}
	go idx.builder()
	return idx, nil
}

func (idx *NGTIndex) GetNnsById(id uint, n int, epsilon float32) ([]int, error) {
	v, err := idx.getItem(id)
	if err != nil {
		return []int{}, err
	}
	return idx.GetAllNns(v, n, epsilon)
}

func (idx *NGTIndex) GetAllNns(v []float64, n int, epsilon float32) ([]int, error) {
	results, err := idx.index.Search(v, n, epsilon)
	ids := make([]int, len(results))
	for i, result := range results {
		ids[i] = int(result.Id)
	}
	return ids, err
}

type buildArgs struct {
	action int
	key    int
	w      []float64
	result chan error
}

func (idx *NGTIndex) builder() {
	for args := range idx.buildChan {
		switch args.action {
		case ADD:
			_, err := idx.addItem(args.key, args.w)
			args.result <- err
		case DELETE:
			args.result <- idx.removeItem(args.key)
		case SAVE:
			args.result <- idx.save()
		case ASYNC_SAVE:
			idx.save()
		}
	}
}

func (idx *NGTIndex) AddItem(key int, w []float64) error {
	args := buildArgs{action: ADD, key: key, w: w, result: make(chan error)}
	idx.buildChan <- args
	return <-args.result
}

func (idx *NGTIndex) RemoveItem(key int) error {
	args := buildArgs{action: DELETE, key: key, result: make(chan error)}
	idx.buildChan <- args
	return <-args.result
}

func (idx *NGTIndex) AsyncSave() {
	idx.buildChan <- buildArgs{action: ASYNC_SAVE}
}

func (idx *NGTIndex) Save() error {
	args := buildArgs{action: SAVE, result: make(chan error)}
	idx.buildChan <- args
	return <-args.result
}

func (idx *NGTIndex) save() error {
	err := idx.pair.save()
	if err != nil {
		return err
	}
	return idx.index.SaveIndex(idx.database)
}

func (idx *NGTIndex) addItem(key int, v []float64) (uint, error) {
	newId, err := idx.index.InsertIndex(v)
	if err != nil {
		return 0, err
	}
	idx.pair.addPair(uint(key), newId)
	return newId, idx.index.CreateIndex(24)
}

func (idx *NGTIndex) removeItem(key int) error {
	if id, ok := idx.pair.idFromKey(uint(key)); ok {
		err := idx.index.RemoveIndex(id.(uint))
		if err != nil {
			return err
		}
		idx.pair.removeByKey(uint(key))
		return nil
	} else {
		return fmt.Errorf("Not Found")
	}
}

func (idx *NGTIndex) getItem(id uint) ([]float64, error) {
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

func (idx *NGTIndex) existItem(id uint) bool {
	obj, err := idx.getItem(id)
	if err != nil || len(obj) == 0 {
		return false
	}
	return true
}
