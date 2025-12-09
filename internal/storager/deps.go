package storager

import "io"

// writerFunc задаёт интерфейс функции io.Writer
type writerFunc func(w io.Writer) (n int64, err error)

// readerFunc задаёт интерфейс функции io.Reader
type readerFunc func(r io.Reader) (n int64, err error)
