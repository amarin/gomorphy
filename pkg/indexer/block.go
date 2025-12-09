package indexer

import (
	"encoding/binary"
	"errors"
	"io"
)

var errUnknownSize = errors.New("unknown block size")

type IndexSize byte

func (l *IndexSize) ReadFrom(r io.Reader) (n int64, err error) {
	var target byte
	if err = binary.Read(r, binary.LittleEndian, &target); err != nil {
		return 0, err
	}

	*l = IndexSize(target)
	return 1, nil
}

func (l *IndexSize) WriteTo(w io.Writer) (n int64, err error) {
	switch *l {
	case Uint8:
		if err = binary.Write(w, binary.LittleEndian, Uint8); err != nil {
			return 0, err
		}
		return 1, nil
	default:
		return 0, errUnknownSize
	}
}
