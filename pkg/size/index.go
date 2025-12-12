package size

import (
	"encoding/binary"
	"errors"
	"io"
)

// Index задаёт размер индекса.
type Index byte

const (
	// Unknown задаёт константу для неизвестных размеров индекса
	Unknown Index = 0

	// Uint8 задаёт константу для определения размера индекса uint8
	Uint8 Index = 1

	// Uint16 задаёт константу для определения размера индекса uint16
	Uint16 Index = 2

	// Uint32 задаёт константу для определения размера индекса uint32
	Uint32 Index = 3
)

func (l *Index) String() string {
	switch *l {
	case Uint8:
		return "uint8"
	case Uint16:
		return "uint16"
	case Uint32:
		return "uint32"
	default:
		return "unknown"
	}
}

var errUnknownSize = errors.New("unknown block size")

func (l *Index) ReadFrom(r io.Reader) (n int64, err error) {
	var target byte
	if err = binary.Read(r, binary.LittleEndian, &target); err != nil {
		return 0, err
	}

	*l = Index(target)

	switch *l {
	case Uint8, Uint16, Uint32:
		return 1, nil
	default:
		return 0, errUnknownSize
	}
}

func (l *Index) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.LittleEndian, byte(*l)); err != nil {
		return 0, err
	}

	return 1, nil
}

func (l *Index) BytesCount() int64 {
	switch *l {
	case Uint8:
		return 1
	case Uint16:
		return 2
	case Uint32:
		return 4
	default:
		return 0
	}
}
