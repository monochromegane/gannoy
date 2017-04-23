package gannoy

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

func CreateMeta(path, file string, tree, dim int) error {
	database := filepath.Join(path, file+".meta")
	_, err := os.Stat(database)
	if err == nil {
		return fmt.Errorf("Already exist database: %s.", database)
	}

	f, err := os.Create(database)
	if err != nil {
		return err
	}
	defer f.Close()

	binary.Write(f, binary.BigEndian, int32(tree))
	binary.Write(f, binary.BigEndian, int32(dim))
	roots := make([]int32, tree)
	for i, _ := range roots {
		roots[i] = int32(-1)
	}
	binary.Write(f, binary.BigEndian, roots)

	return nil
}
