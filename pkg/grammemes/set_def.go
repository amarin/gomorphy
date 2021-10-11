package grammemes

import (
	"fmt"
	"io"
	"sort"

	"github.com/amarin/binutils"
)

// ErrSet identifies grammemes Set related errors.
var ErrSet = fmt.Errorf("%w: set", Error)

// Set stores grammemes unique sorted IDs sets. It used as grammeme ses to define word grammemes.
type Set []uint8

// Len returns length of grammemes GrammemesSet. Implements sort.Interface.
func (grammemesSet Set) Len() int {
	return len(grammemesSet)
}

// Less reports whether the element with index i must sort before the element with index j.
// Implements sort.Interface.
func (grammemesSet Set) Less(i, j int) bool {
	return grammemesSet[i] < grammemesSet[j]
}

// Swap swaps the elements with indexes i and j.
// Implements sort.Interface.
func (grammemesSet Set) Swap(i, j int) {
	grammemesSet[i], grammemesSet[j] = grammemesSet[j], grammemesSet[i]
}

// Sort is a convenience method: x.Sort() calls sort.Sort(x).
func (grammemesSet Set) Sort() { sort.Sort(grammemesSet) }

// EqualTo compares grammemes GrammemesSet with another one.
// Returns true if both sets are contains the same grammemes or false otherwise.
func (grammemesSet Set) EqualTo(another Set) bool {
	if grammemesSet.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < grammemesSet.Len(); idx++ {
		if grammemesSet[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}

// WriteTo writes GrammemesSet data into supplied io.Writer instance.
// Returns written bytes count and error if occurs.
func (grammemesSet Set) WriteTo(w io.Writer) (n int64, err error) {
	var written int

	if written, err = w.Write(binutils.Int8bytes(int8(grammemesSet.Len()))); err != nil || written != 1 {
		return 0, fmt.Errorf("%w: writeTo: len: %v", ErrSet, err)
	}

	if written, err = w.Write(grammemesSet); err != nil {
		return 1 + int64(written), fmt.Errorf("%v: writeTo: data: %w", ErrSet, err)
	}

	return 1 + int64(written), nil
}

// ReadFrom loads GrammemesSet data from provided io.Reader until all data loaded or any error including EOF.
// Returns taken bytes count and error if occurs.
func (grammemesSet *Set) ReadFrom(r io.Reader) (totalBytesTaken int64, err error) {
	var bytesTaken, n int

	data := make([]byte, 256)

	n, err = r.Read(data[0:1])

	switch {
	case n != 1:
		return int64(n), fmt.Errorf("%w: taken %d bytes for uint8 list len", Error, n)
	case err != nil:
		return int64(n), fmt.Errorf("%w: len read: %v", Error, err)
	}

	bytesTaken = n
	expectedLen := data[0]

	n, err = r.Read(data[0:expectedLen])

	switch {
	case n != int(expectedLen):
		return int64(n), fmt.Errorf("%w: got %d bytes for %d elements list", Error, n, expectedLen)
	case err != nil:
		return int64(n), fmt.Errorf("%w: data read: %v", Error, err)
	}

	bytesTaken += n
	*grammemesSet = make(Set, expectedLen)
	copy(*grammemesSet, data[0:expectedLen])

	return int64(bytesTaken), err
}
