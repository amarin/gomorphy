package storage

import (
	"fmt"
	"io"
	"slices"
)

type staticBytes struct {
	data []byte
}

// StaticBytes создаёт реализацию сегмента для записи и чтения статической последовательности байт.
// При записи выводит заданную последовательность.
// При чтении получает количество байт, соответствующее заданному значению
// и проверяет соответствие полученных байт заданному значению.
func StaticBytes(data []byte) Segment {
	return &staticBytes{
		data: data,
	}
}

func (b staticBytes) WriteTo(w io.Writer) (int64, error) {
	ni, err := w.Write(b.data)
	return int64(ni), err
}

func (b staticBytes) ReadFrom(r io.Reader) (int64, error) {
	target := make([]byte, len(b.data))
	ni, err := r.Read(target)
	if err != nil {
		return int64(ni), err
	}

	if !slices.Equal(target, b.data) {
		return int64(ni), fmt.Errorf("data mismatch")
	}

	return int64(ni), nil
}
