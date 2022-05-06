package index

import (
	"fmt"
	"sort"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/storage"
)

// CollectionTableID wraps storage.ID16 to represent TableIDCollection ID in CollectionTable.
type CollectionTableID storage.ID16

// ID16 returns storage.ID16 of CollectionTableID value.
// It's a simple wrapper over storage.ID16(CollectionTableID).
func (id CollectionTableID) ID16() storage.ID16 {
	return storage.ID16(id)
}

// CollectionTable stores a list of TableID.
type CollectionTable []TableIDCollection

// BinaryWriteTo writes CollectionTable data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagSetTable CollectionTable) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint32(uint32(len(tagSetTable))); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tableIDCollection := range tagSetTable {
		if err = writer.WriteObject(tableIDCollection); err != nil {
			return fmt.Errorf("%w: %v", Error, err)
		}
	}

	return nil
}

// BinaryReadFrom reads CollectionTable data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (tagSetTable *CollectionTable) BinaryReadFrom(reader *binutils.BinaryReader) (n int64, err error) {
	var (
		tagSetIndexLen uint32
		currentBytes   int64
	)

	n = 0

	if tagSetIndexLen, err = reader.ReadUint32(); err != nil {
		return n, fmt.Errorf("%w: read: tagset: %v", Error, err)
	}
	n += binutils.Uint32size

	*tagSetTable = make(CollectionTable, tagSetIndexLen)
	for idx := 0; idx < int(tagSetIndexLen); idx++ {
		if currentBytes, err = (*tagSetTable)[idx].BinaryReadFrom(reader); err != nil {
			return n, fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
		n += currentBytes
	}

	return n, nil
}

// Find returns 0-based index of ids.Set16 in Table16 storage.
// If no such set found returns 0 and false found indicator.
// It required sets.Set16 argument to be sorted before.
func (tagSetTable CollectionTable) Find(item TableIDCollection) (storageIdx CollectionTableID, found bool) {
	for id, existedSet := range tagSetTable {
		if existedSet.EqualTo(item) {
			return CollectionTableID(id), true
		}
	}

	return 0, false
}

// Index returns 0-based TagSetID index of ids.Set16 in Table16 instance.
// Returns index of existed item if found or of appended item.
// Panics if specified set empty.
func (tagSetTable *CollectionTable) Index(item TableIDCollection) (storageIdx CollectionTableID) {
	var found bool

	if len(item) == 0 {
		return 0 // zero index means no data
	}

	sort.Sort(item) // sort item before find or adding to index.

	if storageIdx, found = tagSetTable.Find(item); found {
		return storageIdx
	}

	storageIdx = CollectionTableID(len(*tagSetTable))

	*tagSetTable = append(*tagSetTable, item)

	return storageIdx
}

// Get returns TableIDCollection by its CollectionTableID if present or found indicator will be false.
func (tagSetTable CollectionTable) Get(storageIdx CollectionTableID) TableIDCollection {
	if storageIdx == 0 {
		return make(TableIDCollection, 0)
	}
	return tagSetTable[storageIdx]
}
