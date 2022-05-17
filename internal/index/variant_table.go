package index

import (
	"fmt"
	"sort"

	"github.com/amarin/binutils"
)

// VariantsTable stores a list of TagSetID.
type VariantsTable []TagSetIDCollection

// BinaryWriteTo writes VariantsTable data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (variantsTable VariantsTable) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint32(uint32(len(variantsTable))); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tableIDCollection := range variantsTable {
		if err = writer.WriteObject(tableIDCollection); err != nil {
			return fmt.Errorf("%w: %v", Error, err)
		}
	}

	return nil
}

// BinaryReadFrom reads VariantsTable data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (variantsTable *VariantsTable) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var tagSetIndexLen uint32

	if tagSetIndexLen, err = reader.ReadUint32(); err != nil {
		return fmt.Errorf("%w: read: tagset: %v", Error, err)
	}

	*variantsTable = make(VariantsTable, tagSetIndexLen)
	for idx := 0; idx < int(tagSetIndexLen); idx++ {
		if err = (*variantsTable)[idx].BinaryReadFrom(reader); err != nil {
			return fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
	}

	return nil
}

// Find returns specified TagSetIDCollection instance ID in index.
// If no such variants set found returns VariantSubID=0 and false found indicator.
func (variantsTable VariantsTable) Find(item TagSetIDCollection) (storageIdx VariantSubID, found bool) {
	for id, existedSet := range variantsTable {
		if existedSet.EqualTo(item) {
			return VariantSubID(id), true
		}
	}

	return 0, false
}

// Index returns specified TagSetIDCollection instance ID in index.
// If no such variants set found it will be registered before.
// Always return VariantSubID=0 if empty or nil TagSetIDCollection specified.
func (variantsTable *VariantsTable) Index(item TagSetIDCollection) (storageIdx VariantSubID) {
	var found bool

	if len(item) == 0 {
		return 0 // zero index means no data
	}

	sort.Sort(item) // sort item before find or adding to index to compare and search easy.
	if storageIdx, found = variantsTable.Find(item); found {
		return storageIdx
	}

	storageIdx = VariantSubID(len(*variantsTable))

	*variantsTable = append(*variantsTable, item)

	return storageIdx
}

// Get returns TagSetIDCollection by its VariantSubID.
func (variantsTable VariantsTable) Get(storageIdx VariantSubID) TagSetIDCollection {
	if storageIdx == 0 {
		return make(TagSetIDCollection, 0)
	}

	return variantsTable[storageIdx]
}

// Following returns a list of VariantSubID having their VariantSubID above of specified.
func (variantsTable VariantsTable) Following(storageIdx VariantSubID) (res []VariantSubID) {
	res = make([]VariantSubID, 0)
	for idx := int(storageIdx) + 1; idx < len(variantsTable); idx++ {
		res = append(res, VariantSubID(idx))
	}

	return res
}
