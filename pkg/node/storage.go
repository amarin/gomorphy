package node

import (
	"io"
	"maps"
	"slices"

	"github.com/amarin/gomorphy/internal/storage"
)

// WriteTo writes node binary representation into target io.Writer.
func (node Node[A, M]) WriteTo(w io.Writer) (bytesWritten int64, err error) {
	var partBytes int64

	bytesWritten = 0

	// writing len of map
	lenOfMap := A(len(node))
	partBytes, err = storage.WriteUintTo(lenOfMap, w)
	bytesWritten += partBytes
	if err != nil {
		return bytesWritten, err
	}

	keys := slices.Collect(maps.Keys(node))
	slices.Sort(keys)

	// write map pairs
	for _, k := range keys {
		// writing child charIdx
		partBytes, err = storage.WriteUintTo(k, w)
		bytesWritten += partBytes
		if err != nil {
			return bytesWritten, err
		}
		// writing child wordIdx
		partBytes, err = storage.WriteUintTo((node)[k], w)
		bytesWritten += partBytes
		if err != nil {
			return bytesWritten, err
		}
	}

	return bytesWritten, nil
}

// ReadFrom reads node data from specified io.Reader.
func (node Node[A, M]) ReadFrom(r io.Reader) (bytesTaken int64, err error) {
	var (
		partBytes int64
		lenOfMap  A = 0
		currentK  A
		currentV  M
	)

	bytesTaken = 0

	// read map len
	partBytes, err = storage.ReadUintFrom(&lenOfMap, r)
	bytesTaken += partBytes
	if err != nil {
		return bytesTaken, err
	}

	// read <map len> pairs
	for i := A(0); i < lenOfMap; i++ {
		// reading child charIdx
		partBytes, err = storage.ReadUintFrom(&currentK, r)
		bytesTaken += partBytes
		if err != nil {
			return bytesTaken, err
		}
		// writing child wordIdx
		partBytes, err = storage.ReadUintFrom(&currentV, r)
		bytesTaken += partBytes
		if err != nil {
			return bytesTaken, err
		}

		(node)[currentK] = currentV
	}

	return bytesTaken, nil
}
