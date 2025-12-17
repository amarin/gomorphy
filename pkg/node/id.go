package node

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/amarin/gomorphy/pkg/size"
)

var ErrUnexpectedIdSize = errors.New("unexpected ID size")

type ID[S size.Morphemes] struct {
	id S
}

func NewID[S size.Morphemes](id S) *ID[S] {
	return &ID[S]{id: id}
}

func (id *ID[S]) WriteTo(w io.Writer) (n int64, err error) {
	switch s := id.size(); s {
	case size.Uint8:
		if err = binary.Write(w, binary.LittleEndian, uint8(id.id)); err != nil {
			return 0, err
		}

		return s.BytesCount(), nil
	case size.Uint16:
		if err = binary.Write(w, binary.LittleEndian, uint16(id.id)); err != nil {
			return 0, err
		}

		return s.BytesCount(), nil
	case size.Uint32:
		if err = binary.Write(w, binary.LittleEndian, uint32(id.id)); err != nil {
			return 0, err
		}

		return s.BytesCount(), nil

	default:
		return 0, fmt.Errorf("%w: %d", ErrUnexpectedIdSize, s)
	}
}

func (id *ID[S]) ReadFrom(r io.Reader) (n int64, err error) {

	switch s := id.size(); s {
	case size.Uint8:
		target := uint8(0)

		if err = binary.Read(r, binary.LittleEndian, &target); err != nil {
			return 0, err
		}
		id.id = S(target)

		return s.BytesCount(), nil
	case size.Uint16:
		target := uint16(0)

		if err = binary.Read(r, binary.LittleEndian, &target); err != nil {
			return 0, err
		}
		id.id = S(target)

		return s.BytesCount(), nil
	case size.Uint32:
		target := uint32(0)
		if err = binary.Read(r, binary.LittleEndian, &target); err != nil {
			return 0, err
		}
		id.id = S(target)

		return s.BytesCount(), nil
	default:
		return 0, fmt.Errorf("%w: %d", ErrUnexpectedIdSize, s)
	}
}

func (id *ID[S]) size() size.Index {
	var idxSize any = S(0)
	switch idxSize.(type) {
	case uint8:
		return size.Uint8
	case uint16:
		return size.Uint16
	case uint32:
		return size.Uint32
	default:
		return size.Unknown
	}
}

func (id *ID[S]) Value() S {
	return id.id
}
