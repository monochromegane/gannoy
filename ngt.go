package gannoy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
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

type Feature struct {
	W []float64 `json:"features"`
}

func (idx *NGTIndex) ApplyBinLog() error {
	tmp, err := ioutil.TempDir("", "gannoy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// Copy db and map
	tmpdb, err := idx.copyDatabase(tmp)
	if err != nil {
		return err
	}
	tmpmap, err := idx.copyMap(tmp)
	if err != nil {
		return err
	}

	// Open as new NGTIndex
	index, err := ngt.OpenIndex(tmpdb)
	if err != nil {
		return err
	}
	pair, err := newPair(tmpmap)
	if err != nil {
		return err
	}

	// Get current time
	current := time.Now().Format("2006-01-02 03:04:05")

	// Select from binlog where current time
	rows, err := idx.bin.Get(current)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Apply binlog
	for rows.Next() {
		var key int
		var action int
		var features []byte

		err := rows.Scan(&key, &action, &features)
		if err != nil {
			return err
		}
		switch action {
		case DELETE:
			if id, ok := pair.idFromKey(uint(key)); ok {
				err := index.RemoveIndex(id.(uint))
				if err != nil {
					return err
				}
				pair.removeByKey(uint(key))
			}
		case UPDATE:
			var f Feature
			err = json.Unmarshal(features, &f)
			if err != nil {
				return err
			}
			newId, err := index.InsertIndex(f.W)
			if err != nil {
				return err
			}
			pair.addPair(uint(key), newId)
		}
	}
	err = pair.save()
	if err != nil {
		return err
	}
	err = index.CreateIndex(idx.thread)
	if err != nil {
		return err
	}
	err = index.SaveIndex(tmpdb)
	if err != nil {
		return err
	}

	// Defer: delete old binlog (timestamp < current time)
	err = idx.bin.Clear(current)
	if err != nil {
		return err
	}

	// Overwrite
	err = os.Rename(tmpmap, idx.pair.file)
	if err != nil {
		return err
	}
	files, err := ioutil.ReadDir(tmpdb)
	if err != nil {
		return err
	}
	for _, f := range files {
		err = os.Rename(filepath.Join(tmpdb, f.Name()), filepath.Join(idx.database, f.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}

func (idx *NGTIndex) Save() error {
	err := idx.pair.save()
	if err != nil {
		return err
	}
	return idx.index.SaveIndex(idx.database)
}

func (idx NGTIndex) copyDatabase(tmp string) (string, error) {
	path := filepath.Join(tmp, path.Base(idx.database))
	err := os.Mkdir(path, 0755)
	if err != nil {
		return "", err
	}

	files, err := ioutil.ReadDir(idx.database)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		src, err := os.Open(filepath.Join(idx.database, f.Name()))
		if err != nil {
			return "", err
		}

		dst, err := os.Create(filepath.Join(path, f.Name()))
		if err != nil {
			return "", err
		}
		_, err = io.Copy(dst, src)
		if err != nil {
			return "", err
		}

		src.Close()
		dst.Close()
	}

	return path, nil
}

func (idx NGTIndex) copyMap(tmp string) (string, error) {
	src, err := os.Open(idx.pair.file)
	if err != nil {
		return "", err
	}
	defer src.Close()

	path := filepath.Join(tmp, path.Base(idx.pair.file))
	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return path, err
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
