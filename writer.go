package gannoy

import (
	"io"
	"os"
)

type ReopenableWriter struct {
	path string
	w    io.WriteCloser
}

func NewReopenableWriter(path string) (*ReopenableWriter, error) {
	writer := &ReopenableWriter{path: path}
	err := writer.open()
	return writer, err
}

func (w *ReopenableWriter) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}

func (w *ReopenableWriter) ReOpen() error {
	current := w.w
	err := w.open()
	if err != nil {
		return err
	}
	defer current.Close()
	return nil
}

func (w *ReopenableWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	w.w = f
	return nil
}
