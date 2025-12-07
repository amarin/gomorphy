package storage

import (
	"fmt"
	"io"

	"github.com/amarin/binutils"
)

const (
	runeSize     = binutils.Int32size
	runesLenSize = binutils.Uint8size
)

var ErrRunes = fmt.Errorf("%w: runes", Error)

type Runes []rune

// Len returns length of Set16. Implements sort.Interface.
func (runes *Runes) Len() int {
	return len(*runes)
}

// EqualTo compares Set8 with another one.
// Returns true if both sets are contains the same ID8 elements or false otherwise.
// Note: both sets should be sorted before compare.
func (runes *Runes) EqualTo(another Runes) bool {
	if runes.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < runes.Len(); idx++ {
		if (*runes)[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}

// WriteTo writes Set16 data into supplied io.Writer instance.
// Returns written bytes count and error if occurs.
// Implements io.WriterTo.
func (runes *Runes) WriteTo(w io.Writer) (n int64, err error) {
	var written int

	writer := binutils.NewBinaryWriter(w)
	if err = writer.WriteUint8(uint8(runes.Len())); err != nil {
		return 0, fmt.Errorf("%w: writeTo: len: %v", ErrRunes, err)
	}

	written += runesLenSize

	for idx, item := range *runes {
		if err = writer.WriteInt32(item); err != nil {
			return int64(written), fmt.Errorf("%v: writeTo: data[%d]: %w", ErrRunes, idx, err)
		}

		written += runeSize
	}

	return int64(written), nil
}

// ReadFrom loads Set16 data from provided io.Reader until all data loaded or any error including EOF.
// Returns taken bytes count and error if occurs.
// Implements io.ReaderFrom.
func (runes *Runes) ReadFrom(r io.Reader) (totalBytesTaken int64, err error) {
	var (
		bytesTaken   int
		expectedLen  uint8
		nextRuneCode int32
	)

	reader := binutils.NewBinaryReader(r)
	if expectedLen, err = reader.ReadUint8(); err != nil {
		return int64(bytesTaken), fmt.Errorf("%v: readFrom: len: %w", ErrRunes, err)
	}

	bytesTaken += runesLenSize
	*runes = make(Runes, expectedLen)

	for i := 0; uint8(i) < expectedLen; i++ {
		if nextRuneCode, err = reader.ReadInt32(); err != nil {
			return int64(bytesTaken), fmt.Errorf("%v: readFrom: data[%v]: %w", ErrRunes, i, err)
		}

		bytesTaken += runeSize
		(*runes)[i] = nextRuneCode
	}

	return int64(bytesTaken), err
}
