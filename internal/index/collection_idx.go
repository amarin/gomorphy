package index

import (
	"fmt"
	"sort"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/storage"
)

// CollectionID represents TagSet collection ID in TableIDCollectionIndex.
type CollectionID storage.ID32

// TableIDCollectionIndex stores index of all possible TableIDCollection.
type TableIDCollectionIndex []CollectionTable

// BinaryWriteTo writes CollectionTable data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagSetIndex TableIDCollectionIndex) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint8(uint8(len(tagSetIndex))); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tableIDCollection := range tagSetIndex {
		if err = writer.WriteObject(tableIDCollection); err != nil {
			return fmt.Errorf("%w: %v", Error, err)
		}
	}

	return nil
}

// BinaryReadFrom reads CollectionTable data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (tagSetIndex *TableIDCollectionIndex) BinaryReadFrom(reader *binutils.BinaryReader) (n int64, err error) {
	var (
		tagSetIndexLen uint8
		currentBytes   int64
	)

	n = 0

	if tagSetIndexLen, err = reader.ReadUint8(); err != nil {
		return n, fmt.Errorf("%w: read: tagset: %v", Error, err)
	}
	n += binutils.Uint8size

	*tagSetIndex = make(TableIDCollectionIndex, tagSetIndexLen)
	for idx := 0; idx < int(tagSetIndexLen); idx++ {
		if currentBytes, err = (*tagSetIndex)[idx].BinaryReadFrom(reader); err != nil {
			return n, fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
		n += currentBytes
	}

	return n, nil
}

// Find returns 0-based index of ids.Set16 item in TagSetIndex array.
// If no such set found returns 0 and false found flag.
func (tagSetIndex TableIDCollectionIndex) Find(item TableIDCollection) (storageIdx CollectionID, found bool) {
	var itemIdx CollectionTableID

	columnIdx := len(item) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if columnIdx >= len(tagSetIndex) {
		return 0, false // always return not found for columnIdx over collection length.
	}

	if itemIdx, found = tagSetIndex[columnIdx].Find(item); !found {
		return 0, false
	}

	return CollectionID(storage.Combine16(storage.ID16(columnIdx), itemIdx.ID16())), true
}

// Index returns 0-based index of set in Collection8x8 array.
func (tagSetIndex *TableIDCollectionIndex) Index(item TableIDCollection) (storageIdx CollectionID) {
	if len(item) == 0 {
		return 0 // empty set means no data attached
	}

	sort.Sort(item)

	columnIdx := len(item) - 1
	if columnIdx >= len(*tagSetIndex) { // no enough columns extend inplace
		currentLen := len(*tagSetIndex)

		for i := currentLen; i <= columnIdx; i++ {
			*tagSetIndex = append(*tagSetIndex, make(CollectionTable, 0))
		}
	}

	itemIdx := (*tagSetIndex)[columnIdx].Index(item)

	return CollectionID(storage.Combine16(storage.ID16(columnIdx), itemIdx.ID16()))
}

// Get returns set by index.
func (tagSetIndex TableIDCollectionIndex) Get(storageIdx CollectionID) TableIDCollection {
	if storageIdx == 0 {
		return make(TableIDCollection, 0)
	}
	return tagSetIndex[storage.ID32(storageIdx).Upper16()].Get(CollectionTableID(storage.ID32(storageIdx).Lower16()))
}
