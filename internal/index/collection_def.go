package index

import (
	"fmt"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/storage"
)

// TableIDCollection stores a list of TableID identifiers of TagSet stored in TagSetIndex.
type TableIDCollection []TableID

// BinaryWriteTo writes TagSetIndex data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (t TableIDCollection) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint8(uint8(len(t))); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tagSetTable := range t {
		if err = writer.WriteUint32(storage.ID32(tagSetTable).Uint32()); err != nil {
			return fmt.Errorf("%w: %v", Error, err)
		}
	}

	return nil
}

// BinaryReadFrom reads TagSetIndex data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (t *TableIDCollection) BinaryReadFrom(reader *binutils.BinaryReader) (n int64, err error) {
	var (
		tagSetIndexLen uint8
		currentUint32  uint32
	)

	bytesTaken := int64(0)

	if tagSetIndexLen, err = reader.ReadUint8(); err != nil {
		return bytesTaken, fmt.Errorf("%w: read: tagset: %v", Error, err)
	}
	bytesTaken += binutils.Uint8size

	*t = make(TableIDCollection, tagSetIndexLen)
	for idx := 0; idx < int(tagSetIndexLen); idx++ {
		if currentUint32, err = reader.ReadUint32(); err != nil {
			return bytesTaken, fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
		bytesTaken += binutils.Uint32size
		(*t)[idx] = TableID(currentUint32)
	}

	return bytesTaken, nil
}

// Len returns length of TableIDCollection. Implements sort.Interface.
func (t TableIDCollection) Len() int {
	return len(t)
}

// Less returns true i-th element of TableIDCollection is fewer than j-th. Implements sort.Interface.
func (t TableIDCollection) Less(i, j int) bool {
	return t[i] < t[j]
}

// Swap swaps i-th and j-th elements of array in place. Implements sort.Interface.
func (t TableIDCollection) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// EqualTo compares TableIDCollection with another one.
// Returns true if both sets are contains the same ID8 elements or false otherwise.
// Note: both sets should be sorted before compare.
func (t TableIDCollection) EqualTo(another TableIDCollection) bool {
	if t.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < t.Len(); idx++ {
		if t[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}
