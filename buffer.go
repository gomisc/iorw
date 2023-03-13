package iorw

import (
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"gopkg.in/gomisc/errors.v1"
)

const (
	// ErrReadClosedBuffer - ошибка попытки чтения и з закрытого буфера
	ErrReadClosedBuffer = errors.Const("attempt to read from closed buffer")
	// ErrWriteClosedBuffer - ошибка попытки записи в закрытый буфер
	ErrWriteClosedBuffer = errors.Const("attempt to write to closed buffer")

	detectorTimeout = 10 * time.Millisecond
)

// Buffer буфер
type Buffer struct {
	lock         *sync.Mutex
	detectCloser chan interface{}

	contents   []byte
	readCursor uint64
	closed     bool
}

// NewBuffer - конструктор буфера
func NewBuffer() *Buffer {
	return &Buffer{
		lock: &sync.Mutex{},
	}
}

// Reader - враппер ридера
func Reader(reader io.Reader) *Buffer {
	b := &Buffer{
		lock: &sync.Mutex{},
	}

	go func() {
		_, _ = io.Copy(b, reader)
		_ = b.Close()
	}()

	return b
}

// Write - запись в буфер
func (b *Buffer) Write(p []byte) (n int, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.closed {
		return 0, ErrWriteClosedBuffer
	}

	b.contents = append(b.contents, p...)

	return len(p), nil
}

// Read - чтение из буфера
func (b *Buffer) Read(d []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.closed {
		return 0, ErrReadClosedBuffer
	}

	if uint64(len(b.contents)) <= b.readCursor {
		return 0, io.EOF
	}

	n := copy(d, b.contents[b.readCursor:])
	b.readCursor += uint64(n)

	return n, nil
}

// Close - закрывает буфер
func (b *Buffer) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.closed = true

	return nil
}

// Closed - возвращает признак того что буфер закрыт
func (b *Buffer) Closed() bool {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.closed
}

// Contents возвращает содержимое буфера полностью
func (b *Buffer) Contents() []byte {
	b.lock.Lock()
	defer b.lock.Unlock()

	contents := make([]byte, len(b.contents))
	copy(contents, b.contents)

	return contents
}

/*
Detect - поиск в буфере на совпадение искомого значения в потоке байт проходящих через буфер

select {
case <-buffer.Detect("You are not logged in"):

	//log in

case <-buffer.Detect("Success"):

	//carry on

case <-time.After(time.Second):

		//welp
	}

buffer.CancelDetects()
*/
func (b *Buffer) Detect(desired string, args ...any) chan bool {
	formattedRegexp := desired
	if len(args) > 0 {
		formattedRegexp = fmt.Sprintf(desired, args...)
	}

	re := regexp.MustCompile(formattedRegexp)

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.detectCloser == nil {
		b.detectCloser = make(chan interface{})
	}

	closer := b.detectCloser
	response := make(chan bool)

	go func() {
		ticker := time.NewTicker(detectorTimeout)
		defer ticker.Stop()
		defer close(response)

		for {
			select {
			case <-ticker.C:
				b.lock.Lock()
				data, cursor := b.contents[b.readCursor:], b.readCursor
				loc := re.FindIndex(data)
				b.lock.Unlock()

				if loc != nil {
					response <- true

					b.lock.Lock()
					newCursorPosition := cursor + uint64(loc[1])

					if newCursorPosition >= b.readCursor {
						b.readCursor = newCursorPosition
					}
					b.lock.Unlock()

					return
				}
			case <-closer:
				return
			}
		}
	}()

	return response
}

// CancelDetects - закрывает детектор поиска и его горутины
func (b *Buffer) CancelDetects() {
	b.lock.Lock()
	defer b.lock.Unlock()

	close(b.detectCloser)

	b.detectCloser = nil
}
