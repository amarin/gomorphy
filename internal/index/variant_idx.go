package index

import (
	"fmt"
	"sort"

	"github.com/amarin/binutils"
)

// VariantsIndex stores index of all possible TagSetIDCollection.
type VariantsIndex []VariantsTable

// BinaryWriteTo writes CollectionTable data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagSetIndex VariantsIndex) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
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
func (tagSetIndex *VariantsIndex) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var tagSetIndexLen uint8

	if tagSetIndexLen, err = reader.ReadUint8(); err != nil {
		return fmt.Errorf("%w: read: tagset: %v", Error, err)
	}

	*tagSetIndex = make(VariantsIndex, tagSetIndexLen)
	for idx := 0; idx < int(tagSetIndexLen); idx++ {
		if err = (*tagSetIndex)[idx].BinaryReadFrom(reader); err != nil {
			return fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
	}

	return nil
}

// Find returns 0-based index of ids.Set16 item in TagSetIndex array.
// If no such set found returns 0 and false found flag.
func (tagSetIndex VariantsIndex) Find(item TagSetIDCollection) (storageIdx VariantID, found bool) {
	var itemIdx VariantSubID

	if len(item) == 0 {
		return 0, true
	}

	internalTableID := CollectionTableNumber(len(item) - 1)

	if len(item) >= len(tagSetIndex) {
		return 0, false // always return not found for internalTableID over collection length.
	}

	if itemIdx, found = tagSetIndex[internalTableID].Find(item); !found {
		return 0, false
	}

	return internalTableID.Add(1).VariantID(itemIdx), true
}

// Index returns 0-based index of set in Collection8x8 array.
func (tagSetIndex *VariantsIndex) Index(item TagSetIDCollection) (storageIdx VariantID) {
	if len(item) == 0 {
		return 0
	}

	sort.Sort(item)

	tableNumber := CollectionTableNumber(len(item) - 1)
	if int(tableNumber) >= len(*tagSetIndex) { // no enough columns extend inplace
		currentLen := len(*tagSetIndex)

		for i := currentLen; i <= int(tableNumber); i++ {
			*tagSetIndex = append(*tagSetIndex, make(VariantsTable, 0))
		}
	}

	itemIdx := (*tagSetIndex)[tableNumber].Index(item)

	return tableNumber.Add(1).VariantID(itemIdx)
}

// Get returns set by index.
func (tagSetIndex VariantsIndex) Get(storageIdx VariantID) TagSetIDCollection {
	if storageIdx == 0 {
		return make(TagSetIDCollection, 0)
	}

	return tagSetIndex[storageIdx.TableNum().Add(-1)].Get(storageIdx.CollectionTableID())
}

// Table returns internal CollectionTable by its index.
func (tagSetIndex VariantsIndex) Table(tableID CollectionTableNumber) VariantsTable {
	return tagSetIndex[tableID.Add(-1)]
}

func (tagSetIndex VariantsIndex) KnownID() (res []VariantID) {
	res = make([]VariantID, 0)
	for tableIDx, table := range tagSetIndex {
		tableNumber := CollectionTableNumber(tableIDx).Add(1)
		for collectionTableIdx := range table {
			idxInTable := VariantSubID(collectionTableIdx)
			collectionID := tableNumber.VariantID(idxInTable)
			if collectionID != 0 {
				res = append(res, collectionID)
			}
		}
	}

	return res
}

// Tables returns internal variants tables.
func (tagSetIndex VariantsIndex) Tables() []VariantsTable {
	return tagSetIndex
}

type CollectionIDList []VariantID

func (c CollectionIDList) Len() int {
	return len(c)
}

func (c CollectionIDList) Less(i, j int) bool {
	return c[i] < c[j]
}

func (c CollectionIDList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c CollectionIDList) Has(collectionID VariantID) bool {
	for _, havingCollectionID := range c {
		if collectionID == havingCollectionID {
			return true
		}
	}

	return false
}
