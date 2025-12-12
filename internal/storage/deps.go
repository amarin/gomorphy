package storage

import "io"

// WriterToFunc задаёт интерфейс функции io.Writer
type WriterToFunc func(w io.Writer) (n int64, err error)

// ReaderFromFunc задаёт интерфейс функции io.Reader
type ReaderFromFunc func(r io.Reader) (n int64, err error)

// Segment задаёт интерфейс объектов, реализующих запись и чтение данных.
type Segment interface {
	io.ReaderFrom
	io.WriterTo
}

type storageConfigProvider interface {
	// StorageConfig возвращает конфигурацию чтения-записи.
	StorageConfig() Config
}
