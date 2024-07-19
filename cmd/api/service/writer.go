package service

import (
	"strings"
)

// ChunkedWriter is a custom writer that splits output into different sections based on a delimiter and split size.
type ChunkedWriter struct {
	builder   *strings.Builder
	buffer    *strings.Builder
	delimiter string
	splitSize int
	lastSplit int
}

func NewChunkedWriter(split int, delimiter string) ChunkedWriter {
	return ChunkedWriter{
		builder:   new(strings.Builder),
		buffer:    new(strings.Builder),
		delimiter: delimiter,
		splitSize: split,
	}
}

func (w *ChunkedWriter) WriteString(s string) {
	w.buffer.WriteString(s)
	if w.exceeds() && w.lastSplit > 0 {
		w.flush()
	}
}

// Split marks the current buffer position as a potential split point.
func (w *ChunkedWriter) Split() {
	w.lastSplit = w.buffer.Len()
	if w.buffer.Len() >= w.splitSize {
		w.flush()
	}
}

// String returns the entire content of the builder and buffer as a single string.
func (w *ChunkedWriter) String() string {
	if w.buffer.Len() > 0 {
		w.lastSplit = 0
		w.flush()
	}
	return w.builder.String()
}

func (w *ChunkedWriter) flush() {
	if w.builder.Len() > 0 {
		w.builder.WriteString(w.delimiter)
	}
	if w.lastSplit > 0 {
		w.builder.WriteString(w.buffer.String()[:w.lastSplit])
		remaining := w.buffer.String()[w.lastSplit:]
		w.buffer.Reset()
		w.buffer.WriteString(remaining)
	} else {
		w.builder.WriteString(w.buffer.String())
		w.buffer.Reset()
	}
	w.lastSplit = 0
}

func (w *ChunkedWriter) exceeds() bool {
	return w.buffer.Len() > w.splitSize
}
