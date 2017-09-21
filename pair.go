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
	keyToId *syncmap.Map
	idToKey *syncmap.Map
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

func (p *Pair) isEmpty() bool {
	count := 0
	p.keyToId.Range(func(key, value interface{}) bool {
		count += 1
		return false
	})
	return count <= 0
}

func (p *Pair) isLast() bool {
	count := 0
	p.keyToId.Range(func(key, value interface{}) bool {
		count += 1
		return count < 2
	})
	return count == 1
}

func newPair(file string) (Pair, error) {
	f, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return Pair{}, err
	}
	pair, err := newPairFromReader(f)
	if err != nil {
		return pair, err
	}
	pair.file = file
	return pair, nil
}

func newPairFromReader(r io.Reader) (Pair, error) {
	keyToId := &syncmap.Map{}
	idToKey := &syncmap.Map{}

	reader := csv.NewReader(r)
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
	}, nil
}

func (p *Pair) save() error {
	f, err := os.OpenFile(p.file, os.O_WRONLY|os.O_TRUNC, 0644)
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
			fmt.Printf("%v\n", err)
			return false
		}
		return true
	})
	writer.Flush()
	return nil
}

func (p *Pair) drop() error {
	p.keyToId = &syncmap.Map{}
	p.idToKey = &syncmap.Map{}
	return os.Remove(p.file)
}
