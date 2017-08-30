package gannoy

import (
	"encoding/binary"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"

	ngt "github.com/monochromegane/go-ngt"
)

func NewConverter(from string, dim, tree, K int, order binary.ByteOrder) Converter {
	if filepath.Ext(from) == ".csv" {
		return csvConverter{
			dim: dim,
		}
	} else {
		return converter{
			dim:   dim,
			tree:  tree,
			K:     K,
			order: order,
		}
	}
}

type Converter interface {
	Convert(string, string, string, string) error
}

type converter struct {
	dim   int
	tree  int
	K     int
	order binary.ByteOrder
}

func (c converter) Convert(from, path, to, mapPath string) error {
	return nil
}

type csvConverter struct {
	dim int
}

func (c csvConverter) Convert(from, path, to, mapPath string) error {
	file, err := os.Open(from)
	if err != nil {
		return err
	}
	defer file.Close()

	property, _ := ngt.NewNGTProperty(c.dim)
	index, err := CreateGraphAndTree(filepath.Join(path, to), property)
	if err != nil {
		return err
	}

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		key, err := strconv.Atoi(record[0])
		if err != nil {
			return err
		}

		vec := make([]float64, c.dim)
		for i, f := range record[1:] {
			if feature, err := strconv.ParseFloat(f, 64); err != nil {
				return err
			} else {
				vec[i] = feature
			}
		}
		err = index.AddItem(key, vec)
		if err != nil {
			return err
		}
	}
	return index.Save()
}
