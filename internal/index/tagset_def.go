package index

import (
	"fmt"
	"io"
	"sort"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/dag"
)

// TagSet stores unique sorted ID8's ids sets.
// Requires external management to provide item-to-ID8 and vise-versa transitions.
type TagSet []dag.TagID

// BinaryReadFrom reads TagSet data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (tagSet *TagSet) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var (
		tagSetLen    uint8
		currentTagID uint8
	)

	if tagSetLen, err = reader.ReadUint8(); err != nil {
		return fmt.Errorf("%w: read: tagset: %v", Error, err)
	}

	*tagSet = make(TagSet, tagSetLen)
	for idx := 0; idx < int(tagSetLen); idx++ {
		if currentTagID, err = reader.ReadUint8(); err != nil {
			return fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
		(*tagSet)[idx] = dag.TagID(currentTagID)
	}

	return nil
}

// BinaryWriteTo writes TagSet data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagSet TagSet) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint8(uint8(len(tagSet))); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tagID := range tagSet {
		if err = writer.WriteUint8(uint8(tagID)); err != nil {
			return fmt.Errorf("%w: %v", Error, err)
		}
	}

	return nil
}

// Len returns length of TagSet. Implements sort.Interface.
func (tagSet TagSet) Len() int {
	return len(tagSet)
}

// Less reports whether the element with index i must be placed before the element with index j.
// Implements sort.Interface.
func (tagSet TagSet) Less(i, j int) bool {
	return tagSet[i] < tagSet[j]
}

// Swap swaps the elements with ids i and j.
// Implements sort.Interface.
func (tagSet TagSet) Swap(i, j int) {
	tagSet[i], tagSet[j] = tagSet[j], tagSet[i]
}

// Sort is a convenience method: x.Sort() calls sort.Sort(x). Does inplace sorting of TagSet items.
func (tagSet TagSet) Sort() { sort.Sort(tagSet) }

// EqualTo compares TagSet with another one.
// Returns true if both sets are contains the same ID8 elements or false otherwise.
// Note: both sets should be sorted before compare.
func (tagSet TagSet) EqualTo(another TagSet) bool {
	if tagSet.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < tagSet.Len(); idx++ {
		if tagSet[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}

// WriteTo writes TagSet data into supplied io.Writer instance.
// Returns written bytes count and error if occurs.
// Implements io.WriterTo.
func (tagSet TagSet) WriteTo(w io.Writer) (n int64, err error) {
	writer := binutils.NewBinaryWriter(w)
	err = tagSet.BinaryWriteTo(writer)
	return int64(writer.BytesWritten()), err
}

// ReadFrom loads TagSet data from provided io.Reader until all data loaded or any error including EOF.
// Returns taken bytes count and error if occurs.
// Implements io.ReaderFrom.
func (tagSet *TagSet) ReadFrom(r io.Reader) (totalBytesTaken int64, err error) {
	var (
		bytesTaken  int
		expectedLen uint8
		nextUint8   uint8
	)

	reader := binutils.NewBinaryReader(r)
	if expectedLen, err = reader.ReadUint8(); err != nil {
		return int64(bytesTaken), fmt.Errorf("%v: readFrom: len: %w", Error, err)
	}

	bytesTaken += binutils.Uint8size
	*tagSet = make(TagSet, expectedLen)

	for i := 0; uint8(i) < expectedLen; i++ {
		if nextUint8, err = reader.ReadUint8(); err != nil {
			return int64(bytesTaken), fmt.Errorf("%v: readFrom: data[%v]: %w", Error, i, err)
		}

		bytesTaken += binutils.Uint8size
		(*tagSet)[i] = dag.TagID(nextUint8)
	}

	return int64(bytesTaken), err
}
