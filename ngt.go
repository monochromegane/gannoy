package gannoy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	ngt "github.com/monochromegane/go-ngt"
)

type NGTIndex struct {
	database  string
	index     ngt.NGTIndex
	buildChan chan buildArgs
	mu        *sync.RWMutex
	pair      Pair
	thread    int
	timeout   time.Duration
	bin       BinLog
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
		mu:        &sync.RWMutex{},
		pair:      pair,
	}
	go idx.builder()
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
		database:  database,
		index:     index,
		buildChan: make(chan buildArgs, 1),
		mu:        &sync.RWMutex{},
		pair:      pair,
		thread:    thread,
		timeout:   timeout,
		bin:       bin,
	}
	go idx.builder()
	return idx, nil
}

func (idx NGTIndex) String() string {
	return filepath.Base(idx.database)
}

type searchResult struct {
	ids []int
	err error
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

func (idx *NGTIndex) SearchItem(key uint, limit int, epsilon float32) ([]int, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
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

type buildArgs struct {
	action int
	key    int
	w      []float64
	result chan error
}

func (idx *NGTIndex) buildWithTimeout(errCh chan error) error {
	ctx, cancel := context.WithTimeout(context.Background(), idx.timeout)
	defer cancel()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return newTimeoutErrorFrom(ctx.Err())
	}
}

func (idx *NGTIndex) builder() {
	for args := range idx.buildChan {
		switch args.action {
		case ADD:
			func() {
				idx.mu.Lock()
				defer idx.mu.Unlock()
				errCh := make(chan error, 1)
				go func() {
					_, err := idx.addItem(args.key, args.w)
					errCh <- err
					close(errCh)
				}()
				args.result <- idx.buildWithTimeout(errCh)
			}()
		case DELETE:
			func() {
				idx.mu.Lock()
				defer idx.mu.Unlock()
				errCh := make(chan error, 1)
				go func() {
					errCh <- idx.removeItem(args.key)
					close(errCh)
				}()
				args.result <- idx.buildWithTimeout(errCh)
			}()
		case UPDATE:
			func() {
				idx.mu.Lock()
				defer idx.mu.Unlock()
				if _, ok := idx.pair.idFromKey(uint(args.key)); ok {
					errCh := make(chan error, 1)
					go func() {
						errCh <- idx.removeItem(args.key)
						close(errCh)
					}()
					err := idx.buildWithTimeout(errCh)
					if err != nil {
						args.result <- err
						return
					}
				}
				errCh := make(chan error, 1)
				go func() {
					_, err := idx.addItem(args.key, args.w)
					errCh <- err
					close(errCh)
				}()
				args.result <- idx.buildWithTimeout(errCh)
			}()
		case SAVE:
			args.result <- idx.save()
		case ASYNC_SAVE:
			idx.save()
		}
	}
}

func (idx *NGTIndex) AddItem(key int, w []float64) error {
	args := buildArgs{action: ADD, key: key, w: w, result: make(chan error)}
	defer close(args.result)
	idx.buildChan <- args
	return <-args.result
}

func (idx *NGTIndex) RemoveItem(key int) error {
	args := buildArgs{action: DELETE, key: key, result: make(chan error)}
	defer close(args.result)
	idx.buildChan <- args
	return <-args.result
}

func (idx *NGTIndex) UpdateItem(key int, w []float64) error {
	args := buildArgs{action: UPDATE, key: key, w: w, result: make(chan error)}
	defer close(args.result)
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
	return newId, idx.index.CreateIndex(idx.thread)
}

func (idx *NGTIndex) addItemWithoutCreateIndex(key int, v []float64) (uint, error) {
	newId, err := idx.index.InsertIndex(v)
	if err != nil {
		return 0, err
	}
	idx.pair.addPair(uint(key), newId)
	return newId, nil
}

func (idx *NGTIndex) removeItem(key int) error {
	if id, ok := idx.pair.idFromKey(uint(key)); ok {
		if idx.pair.isLast() {
			// If all is deleted, the next create index will stop responding.
			return fmt.Errorf("Skip removing")
		} else {
			err := idx.index.RemoveIndex(id.(uint))
			if err != nil {
				return err
			}
			idx.pair.removeByKey(uint(key))
			return nil
		}
	} else {
		return fmt.Errorf("Not found")
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
