package storage //nolint:godupl

import (
	"fmt"
	"io"
	"sort"

	"github.com/amarin/binutils"
)

// ErrSet16 identifies Set16 related errors.
var ErrSet16 = fmt.Errorf("%w: set16", Error)

// Set16 stores unique sorted ids.ID16's ids sets.
// Requires external implementation to provide item -> ids.ID16 and vise-versa transitions.
type Set16 []ID16

// Len returns length of Set16. Implements sort.Interface.
func (set16 Set16) Len() int {
	return len(set16)
}

// Less reports whether the element with index i must be placed before the element with index j.
// Implements sort.Interface.
func (set16 Set16) Less(i, j int) bool {
	return set16[i] < set16[j]
}

// Swap swaps the elements with ids i and j.
// Implements sort.Interface.
func (set16 Set16) Swap(i, j int) {
	set16[i], set16[j] = set16[j], set16[i]
}

// Sort is a convenience method: x.Sort() calls sort.Sort(x). Does inplace sorting of Set8 items.
func (set16 Set16) Sort() { sort.Sort(set16) }

// EqualTo compares Set8 with another one.
// Returns true if both sets are contains the same ID8 elements or false otherwise.
// Note: both sets should be sorted before compare.
func (set16 Set16) EqualTo(another Set16) bool {
	if set16.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < set16.Len(); idx++ {
		if set16[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}

// WriteTo writes Set16 data into supplied io.Writer instance.
// Returns written bytes count and error if occurs.
// Implements io.WriterTo.
func (set16 Set16) WriteTo(w io.Writer) (n int64, err error) {
	var written int

	writer := binutils.NewBinaryWriter(w)
	if err = writer.WriteUint16(uint16(len(set16))); err != nil {
		return 0, fmt.Errorf("%w: writeTo: len: %v", ErrSet16, err)
	}

	written += binutils.Uint16size

	for idx, item := range set16 {
		if err = writer.WriteUint16(uint16(item)); err != nil {
			return int64(written), fmt.Errorf("%v: writeTo: data[%d]: %w", ErrSet16, idx, err)
		}

		written += binutils.Uint16size
	}

	return int64(written), nil
}

// ReadFrom loads Set16 data from provided io.Reader until all data loaded or any error including EOF.
// Returns taken bytes count and error if occurs.
// Implements io.ReaderFrom.
func (set16 *Set16) ReadFrom(r io.Reader) (totalBytesTaken int64, err error) {
	var (
		bytesTaken  int
		expectedLen uint16
		nextUint16  uint16
	)

	reader := binutils.NewBinaryReader(r)
	if expectedLen, err = reader.ReadUint16(); err != nil {
		return int64(bytesTaken), fmt.Errorf("%v: readFrom: len: %w", ErrSet16, err)
	}

	bytesTaken += binutils.Uint16size
	*set16 = make(Set16, expectedLen)

	for i := 0; uint16(i) < expectedLen; i++ {
		if nextUint16, err = reader.ReadUint16(); err != nil {
			return int64(bytesTaken), fmt.Errorf("%v: readFrom: data[%v]: %w", ErrSet16, i, err)
		}

		bytesTaken += binutils.Uint16size
		(*set16)[i] = ID16(nextUint16)
	}

	return int64(bytesTaken), err
}
