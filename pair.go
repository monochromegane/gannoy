package gannoy

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/sync/syncmap"
)

type Pair struct {
	keyToId syncmap.Map
	idToKey syncmap.Map
	file    string
}

func (p *Pair) addPair(key, id interface{}) {
	p.keyToId.Store(key, id)
	p.idToKey.Store(id, key)
}

func (p *Pair) keyFromId(id interface{}) (interface{}, bool) {
	return p.idToKey.Load(id)
}

func (p *Pair) idFromKey(key interface{}) (interface{}, bool) {
	return p.keyToId.Load(key)
}

func (p *Pair) removeByKey(key interface{}) {
	if id, ok := p.idFromKey(key); ok {
		p.keyToId.Delete(key)
		p.idToKey.Delete(id)
	} else {
		p.keyToId.Delete(key)
	}
}

func newPair(file string) (Pair, error) {
	f, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return Pair{}, err
	}
	keyToId := syncmap.Map{}
	idToKey := syncmap.Map{}

	reader := csv.NewReader(f)
	reader.Comma = ','
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return Pair{}, err
		}
		key, err := strconv.ParseUint(record[0], 10, 0)
		if err != nil {
			return Pair{}, err
		}
		id, err := strconv.ParseUint(record[1], 10, 0)
		if err != nil {
			return Pair{}, err
		}
		uintKey := uint(key)
		uintId := uint(id)
		keyToId.Store(uintKey, uintId)
		idToKey.Store(uintId, uintKey)
	}

	return Pair{
		keyToId: keyToId,
		idToKey: idToKey,
		file:    file,
	}, nil
}

func (p *Pair) save() error {
	f, err := os.OpenFile(p.file, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(f)
	writer.Comma = ','
	p.keyToId.Range(func(key, value interface{}) bool {
		record := make([]string, 2)
		record[0] = fmt.Sprint(key)
		record[1] = fmt.Sprint(value)
		err := writer.Write(record)
		if err != nil {
			return false
		}
		return true
	})
	writer.Flush()
	return nil
}

func (p *Pair) drop() error {
	p.keyToId = syncmap.Map{}
	p.idToKey = syncmap.Map{}
	return os.Remove(p.file)
}
