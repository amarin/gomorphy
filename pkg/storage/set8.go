package storage

import (
	"fmt"
	"io"
	"sort"

	"github.com/amarin/binutils"
)

// ErrSet8 identifies Set8 related errors.
var ErrSet8 = fmt.Errorf("%w: set8", Error)

// Set8 stores unique sorted ID8's ids sets.
// Requires external management to provide item-to-ID8 and vise-versa transitions.
type Set8 []ID8

// Len returns length of Set8. Implements sort.Interface.
func (set8 Set8) Len() int {
	return len(set8)
}

// Less reports whether the element with index i must be placed before the element with index j.
// Implements sort.Interface.
func (set8 Set8) Less(i, j int) bool {
	return set8[i] < set8[j]
}

// Swap swaps the elements with ids i and j.
// Implements sort.Interface.
func (set8 Set8) Swap(i, j int) {
	set8[i], set8[j] = set8[j], set8[i]
}

// Sort is a convenience method: x.Sort() calls sort.Sort(x). Does inplace sorting of Set8 items.
func (set8 Set8) Sort() { sort.Sort(set8) }

// EqualTo compares Set8 with another one.
// Returns true if both sets are contains the same ID8 elements or false otherwise.
// Note: both sets should be sorted before compare.
func (set8 Set8) EqualTo(another Set8) bool {
	if set8.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < set8.Len(); idx++ {
		if set8[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}

// WriteTo writes Set8 data into supplied io.Writer instance.
// Returns written bytes count and error if occurs.
// Implements io.WriterTo.
func (set8 Set8) WriteTo(w io.Writer) (n int64, err error) {
	var written int

	writer := binutils.NewBinaryWriter(w)
	if err = writer.WriteUint8(uint8(len(set8))); err != nil {
		return 0, fmt.Errorf("%w: writeTo: len: %v", ErrSet8, err)
	}

	written += binutils.Uint8size

	for idx, item := range set8 {
		if err = writer.WriteUint8(uint8(item)); err != nil {
			return int64(written), fmt.Errorf("%v: writeTo: data[%d]: %w", ErrSet8, idx, err)
		}

		written += binutils.Uint8size
	}

	return int64(written), nil
}

// ReadFrom loads Set8 data from provided io.Reader until all data loaded or any error including EOF.
// Returns taken bytes count and error if occurs.
// Implements io.ReaderFrom.
func (set8 *Set8) ReadFrom(r io.Reader) (totalBytesTaken int64, err error) {
	var (
		bytesTaken  int
		expectedLen uint8
		nextUint8   uint8
	)

	reader := binutils.NewBinaryReader(r)
	if expectedLen, err = reader.ReadUint8(); err != nil {
		return int64(bytesTaken), fmt.Errorf("%v: readFrom: len: %w", ErrSet8, err)
	}

	bytesTaken += binutils.Uint8size
	*set8 = make(Set8, expectedLen)

	for i := 0; uint8(i) < expectedLen; i++ {
		if nextUint8, err = reader.ReadUint8(); err != nil {
			return int64(bytesTaken), fmt.Errorf("%v: readFrom: data[%v]: %w", ErrSet8, i, err)
		}

		bytesTaken += binutils.Uint8size
		(*set8)[i] = ID8(nextUint8)
	}

	return int64(bytesTaken), err
}
