package iorw

import (
	"io"
	"sync"
)

// PrefixedWriter - io.Writer с префексированным выводом
type PrefixedWriter struct {
	writer        io.Writer
	lock          *sync.Mutex
	prefix        []byte
	atStartOfLine bool
}

// NewPrefixedWriter - конструктор префексированного io.Writer
func NewPrefixedWriter(prefix string, writer io.Writer) *PrefixedWriter {
	return &PrefixedWriter{
		prefix:        []byte(prefix),
		writer:        writer,
		lock:          &sync.Mutex{},
		atStartOfLine: true,
	}
}

func (w *PrefixedWriter) Write(b []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	toWrite := []byte{}

	for _, c := range b {
		if w.atStartOfLine {
			toWrite = append(toWrite, w.prefix...)
		}

		toWrite = append(toWrite, c)

		w.atStartOfLine = c == '\n'
	}

	if _, err := w.writer.Write(toWrite); err != nil {
		return 0, err
	}

	return len(b), nil
}
