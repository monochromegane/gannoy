package gannoy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	ngt "github.com/yahoojapan/gongt"
)

type NGTIndex struct {
	database string
	index    *ngt.NGT
	pair     Pair
	thread   int
	timeout  time.Duration
	bin      BinLog
}

func CreateGraphAndTree(database string, index *ngt.NGT) (NGTIndex, error) {
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
		index:    index.Open(),
		pair:     pair,
		bin:      bin,
	}
	return idx, nil
}

func NewNGTIndex(database string, thread int, timeout time.Duration) (NGTIndex, error) {
	index := ngt.New(database).Open()
	ngtIndex, err := NewNGTIndexMeta(database, thread, timeout)
	if err != nil {
		return NGTIndex{}, err
	}
	ngtIndex.index = index
	return ngtIndex, nil
}

func NewNGTIndexMeta(database string, thread int, timeout time.Duration) (NGTIndex, error) {
	pair, err := newPair(database + ".map")
	if err != nil {
		return NGTIndex{}, err
	}
	bin := NewBinLog(database + ".bin")
	err = bin.Open()
	if err != nil {
		return NGTIndex{}, err
	}
	return NGTIndex{
		database: database,
		pair:     pair,
		thread:   thread,
		timeout:  timeout,
		bin:      bin,
	}, nil
}

func (idx NGTIndex) String() string {
	return filepath.Base(idx.database)
}

type searchResult struct {
	ids []int
	err error
}

func (idx *NGTIndex) SearchItem(key uint, limit int, epsilon float64) ([]int, error) {
	resultCh := make(chan searchResult, 1)
	go func() {
		ids, err := idx.GetNnsByKey(key, limit, epsilon)
		resultCh <- searchResult{ids: ids, err: err}
		close(resultCh)
	}()
	result := idx.searchWithTimeout(resultCh)
	return result.ids, result.err
}

func (idx *NGTIndex) GetNnsByKey(key uint, n int, epsilon float64) ([]int, error) {
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

func (idx *NGTIndex) GetAllNns(v []float64, n int, epsilon float64) ([]int, error) {
	results, err := idx.index.Search(v, n, epsilon)
	ids := make([]int, len(results))
	for i, result := range results {
		ids[i] = int(result.ID)
	}
	return ids, newNGTSearchErrorFrom(err)
}

func (idx *NGTIndex) UpdateBinLog(key, action int, features []byte) error {
	return idx.bin.Add(key, action, features)
}

type Feature struct {
	W []float64 `json:"features"`
}

func (idx *NGTIndex) Apply() error {
	tmp, err := ioutil.TempDir("", "gannoy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// Get current time
	current := time.Now().Format("2006-01-02 15:04:05")

	// Select from binlog where current time
	cnt, err := idx.bin.Count(current)
	if err != nil {
		return err
	} else if cnt == 0 {
		return TargetNotExistError{}
	}

	rows, err := idx.bin.Get(current)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Open as new NGTIndex
	index, err := NewNGTIndex(idx.database, idx.thread, idx.timeout)
	if err != nil {
		return err
	}
	defer index.Close()

	// Apply from binlog
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
			if id, ok := index.pair.idFromKey(uint(key)); ok {
				err := index.index.StrictRemove(id.(uint))
				if err != nil {
					return err
				}
				index.pair.removeByKey(uint(key))
			}
		case UPDATE:
			var f Feature
			err = json.Unmarshal(features, &f)
			if err != nil {
				return err
			}
			newId, err := index.index.StrictInsert(f.W)
			if err != nil {
				return err
			}
			index.pair.addPair(uint(key), newId)
		}
	}

	tmpmap := filepath.Join(tmp, path.Base(idx.pair.file))
	err = index.pair.saveAs(tmpmap)
	if err != nil {
		return err
	}
	err = index.index.CreateIndex(idx.thread)
	if err != nil {
		return err
	}
	tmpdb := filepath.Join(tmp, path.Base(idx.database))
	index.index.SetIndexPath(tmpdb)
	err = index.index.SaveIndex()
	if err != nil {
		return err
	}

	// Apply to DB
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

	return idx.bin.Clear(current)
}

func (idx *NGTIndex) Save() error {
	err := idx.pair.save()
	if err != nil {
		return err
	}
	return idx.index.SaveIndex()
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
	v, err := idx.index.GetStrictVector(id)
	if err != nil {
		return []float64{}, err
	}

	ret := make([]float64, len(v))
	for i, o := range v {
		ret[i] = float64(o)
	}
	return ret, nil
}

func (idx *NGTIndex) existItem(id uint) bool {
	obj, err := idx.getItem(id)
	if err != nil || len(obj) == 0 {
		return false
	}
	return true
}

func (idx *NGTIndex) Close() {
	if idx.index != nil {
		idx.index.Close()
	}
	idx.bin.Close()
}

func (idx *NGTIndex) Drop() error {
	err := idx.pair.drop()
	if err != nil {
		return err
	}
	err = idx.bin.Drop()
	if err != nil {
		return err
	}
	return os.RemoveAll(idx.database)
}
