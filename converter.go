package gannoy

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"syscall"
)

func NewConverter(dim int, order binary.ByteOrder) converter {
	return converter{
		dim: dim,

		order: order,
	}
}

type converter struct {
	dim   int
	order binary.ByteOrder
}

func (c converter) Convert(from, to string, tree int) error {
	ann, err := os.Open(from)
	if err != nil {
		return err
	}

	err = CreateMeta(".", to, tree, c.dim, 50)
	if err != nil {
		return err
	}

	gannoy, err := NewGannoyIndex(to+".meta", Angular{}, RandRandom{})
	if err != nil {
		return err
	}

	stat, _ := ann.Stat()
	count := int(stat.Size() / c.nodeSize())

	for i := 0; i < count; i++ {
		b := make([]byte, c.nodeSize())
		_, err = syscall.Pread(int(ann.Fd()), b, c.offset(i))
		if err != nil {
			return err
		}

		buf := bytes.NewReader(b)

		var nDescendants int32
		binary.Read(buf, c.order, &nDescendants)
		if int(nDescendants) != 1 {
			break
		}

		buf.Seek(int64(4*2), io.SeekCurrent) // skip children

		vec := make([]float64, c.dim)
		binary.Read(buf, c.order, &vec)

		err = gannoy.AddItem(i, vec)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c converter) offset(index int) int64 {
	return c.nodeSize() * int64(index)
}

func (c converter) nodeSize() int64 {
	return int64(4 + // n_descendants
		4*2 + // children[2]
		8*c.dim) // v[1]
}
