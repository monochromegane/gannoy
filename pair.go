package gannoy

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

type Pair struct {
	keyToId map[interface{}]interface{}
	idToKey map[interface{}]interface{}
	file    string
}

func (p *Pair) addPair(key, id interface{}) {
	p.keyToId[key] = id
	p.idToKey[id] = key
}

func (p *Pair) keyFromId(id interface{}) (interface{}, bool) {
	key, ok := p.idToKey[id]
	return key, ok
}

func (p *Pair) idFromKey(key interface{}) (interface{}, bool) {
	id, ok := p.keyToId[key]
	return id, ok
}

func (p *Pair) removeByKey(key interface{}) {
	if id, ok := p.idFromKey(key); ok {
		delete(p.keyToId, key)
		delete(p.idToKey, id)
	} else {
		delete(p.keyToId, key)
	}
}

func newPair(file string) (Pair, error) {
	f, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return Pair{}, err
	}
	defer f.Close()

	pair, err := newPairFromReader(f)
	if err != nil {
		return pair, err
	}
	pair.file = file
	return pair, nil
}

func newPairFromReader(r io.Reader) (Pair, error) {
	keyToId := map[interface{}]interface{}{}
	idToKey := map[interface{}]interface{}{}

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
		keyToId[uintKey] = uintId
		idToKey[uintId] = uintKey
	}

	return Pair{
		keyToId: keyToId,
		idToKey: idToKey,
	}, nil
}

func (p *Pair) save() error {
	return p.saveAs(p.file)
}

func (p *Pair) saveAs(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = ','
	for key, value := range p.keyToId {
		record := make([]string, 2)
		record[0] = fmt.Sprint(key)
		record[1] = fmt.Sprint(value)
		err := writer.Write(record)
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return nil
}

func (p *Pair) drop() error {
	p.keyToId = map[interface{}]interface{}{}
	p.idToKey = map[interface{}]interface{}{}
	return os.Remove(p.file)
}
