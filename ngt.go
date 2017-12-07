package gannoy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	ngt "github.com/monochromegane/go-ngt"
)

type NGTIndex struct {
	database string
	index    ngt.NGTIndex
	pair     Pair
	thread   int
	timeout  time.Duration
	bin      BinLog
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
		database: database,
		index:    index,
		pair:     pair,
	}
	return idx, nil
}

func NewNGTIndex(database string, thread int, timeout time.Duration) (NGTIndex, error) {
	index, err := ngt.OpenIndex(database)
	if err != nil {
		return NGTIndex{}, err
	}
	pair, err := newPair(database + ".map")
	if err != nil {
		return NGTIndex{}, err
	}
	bin := NewBinLog(database + ".bin")
	err = bin.Open()
	if err != nil {
		return NGTIndex{}, err
	}
	idx := NGTIndex{
		database: database,
		index:    index,
		pair:     pair,
		thread:   thread,
		timeout:  timeout,
		bin:      bin,
	}
	return idx, nil
}

func (idx NGTIndex) String() string {
	return filepath.Base(idx.database)
}

type searchResult struct {
	ids []int
	err error
}

func (idx *NGTIndex) SearchItem(key uint, limit int, epsilon float32) ([]int, error) {
	resultCh := make(chan searchResult, 1)
	go func() {
		ids, err := idx.GetNnsByKey(key, limit, epsilon)
		resultCh <- searchResult{ids: ids, err: err}
		close(resultCh)
	}()
	result := idx.searchWithTimeout(resultCh)
	return result.ids, result.err
}

func (idx *NGTIndex) GetNnsByKey(key uint, n int, epsilon float32) ([]int, error) {
	if id, ok := idx.pair.idFromKey(key); !ok {
		return nil, fmt.Errorf("Not found")
	} else {
		v, err := idx.getItem(id.(uint))
		if err != nil {
			return nil, err
		}
		ids, err := idx.GetAllNns(v, n, epsilon)
		if err != nil {
			return nil, err
		}
		keys := make([]int, len(ids))
		for i, id_ := range ids {
			if key, ok := idx.pair.keyFromId(uint(id_)); ok {
				keys[i] = int(key.(uint))
			}
		}
		return keys, nil
	}
}

func (idx *NGTIndex) GetAllNns(v []float64, n int, epsilon float32) ([]int, error) {
	results, err := idx.index.Search(v, n, epsilon)
	ids := make([]int, len(results))
	for i, result := range results {
		ids[i] = int(result.Id)
	}
	return ids, newNGTSearchErrorFrom(err)
}

func (idx *NGTIndex) UpdateBinLog(key, action int, features []byte) error {
	return idx.bin.Add(key, action, features)
}

func (idx *NGTIndex) searchWithTimeout(resultCh chan searchResult) searchResult {
	ctx, cancel := context.WithTimeout(context.Background(), idx.timeout)
	defer cancel()
	select {
	case result := <-resultCh:
		return result
	case <-ctx.Done():
		return searchResult{err: newTimeoutErrorFrom(ctx.Err())}
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

func (idx *NGTIndex) Close() {
	idx.index.Close()
}

func (idx *NGTIndex) Drop() error {
	idx.Close()
	err := idx.pair.drop()
	if err != nil {
		return err
	}
	return os.RemoveAll(idx.database)
}
