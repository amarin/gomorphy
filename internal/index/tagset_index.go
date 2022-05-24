package index

import (
	"fmt"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/storage"
)

const binaryTagSetPrefix = "TI"

// TagSetIndex stores unique TagSet items organized in TagSetTable storages where each storage
// keeps sets of same sizes.
// Each TagSetIndex item is addressable by unique TagSetID value
// which consists of TagSetTableNumber internal table index and TagSetSubID element index in TagSetTable item.
type TagSetIndex []TagSetTable

// // String returns string representation of TagSetIndex.
// // Implements fmt.Stringer.
// func (tagSetIndex TagSetIndex) String() string {
// 	tableStrings := make([]string, len(tagSetIndex))
// 	for idx, table := range tagSetIndex {
// 		tableStrings[idx] = strconv.Itoa(idx) + ": " + table.String()
// 	}
//
// 	return binaryTagSetPrefix + "(" + strings.Join(tableStrings, ",") + ")"
// }

// Size returns indexed elements count in TagSetIndex.
func (tagSetIndex TagSetIndex) Size() (res int) {
	res = 0
	for _, table := range tagSetIndex {
		res += table.Len()
	}
	return res
}

// BinaryWriteTo writes TagSetIndex data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagSetIndex TagSetIndex) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if writer == nil {
		return fmt.Errorf("%w: TagSetIndex", ErrNilWriter)
	}

	if err = writer.WriteUint32(uint32(len(tagSetIndex))); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tagSetTable := range tagSetIndex {
		if err = tagSetTable.BinaryWriteTo(writer); err != nil {
			return err
		}
	}

	return nil
}

// BinaryReadFrom reads TagSetIndex data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (tagSetIndex *TagSetIndex) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	if reader == nil {
		return fmt.Errorf("%w: TagSetIndex", ErrNilReader)
	}

	var tagSetIndexLen uint32

	if tagSetIndexLen, err = reader.ReadUint32(); err != nil {
		return fmt.Errorf("%w: read: tagset: %v", Error, err)
	}

	*tagSetIndex = make(TagSetIndex, tagSetIndexLen)
	for idx := 0; idx < int(tagSetIndexLen); idx++ {
		if err = (*tagSetIndex)[idx].BinaryReadFrom(reader); err != nil {
			return fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
	}

	return nil
}

// Find returns ID of specified TAgSet in index.
// If no such set found returns TagSetID=0 and false found indicator.
func (tagSetIndex *TagSetIndex) Find(item TagSet) (storageIdx TagSetID, found bool) {
	var lower TagSetSubID

	if len(item) == 0 {
		return 0, false // always return not found for empty sets
	}

	zeroBasedTableIdx := TagSetTableNumber(len(item) - 1)

	if table, ok := tagSetIndex.getTable(zeroBasedTableIdx); ok {
		if lower, ok = table.Find(item); ok {
			return zeroBasedTableIdx.TagSetID(lower), ok
		}
	}

	return 0, false
}

// Index returns specified TagSet ID in index.
// If no such TagSet registered before, it will be indexed first.
// To check if TagSet present in index without registering use Find instead.
func (tagSetIndex *TagSetIndex) Index(item TagSet) (storageIdx TagSetID) {
	if len(item) == 0 {
		return 0 // empty set means no data
	}

	columnIdx := len(item) - 1
	if columnIdx >= len(*tagSetIndex) { // no enough columns extend inplace
		currentLen := len(*tagSetIndex)

		for i := currentLen; i <= columnIdx; i++ {
			*tagSetIndex = append(*tagSetIndex, make(TagSetTable, 0))
		}
	}

	itemIdx := (*tagSetIndex)[columnIdx].Index(item)

	return TagSetID(storage.Combine16(storage.ID16(columnIdx), itemIdx.ID16()))
}

// TableIDs returns all known TagSet's identification list.
func (tagSetIndex TagSetIndex) TableIDs() (res []TagSetID) {
	res = make([]TagSetID, 0)
	for tableIDx, table := range tagSetIndex {
		for idxInTable := range table {
			tableID := TagSetTableNumber(tableIDx).Add(1).TagSetID(TagSetSubID(idxInTable))
			res = append(res, tableID)
		}
	}

	return res
}

// getTable returns TagSetTable by TagSetTableNumber value or false found indicator.
func (tagSetIndex TagSetIndex) getTable(zeroBasedTableNum TagSetTableNumber) (table TagSetTable, ok bool) {
	if zeroBasedTableNum.Int() >= len(tagSetIndex) {
		return nil, false
	}

	return tagSetIndex[zeroBasedTableNum], true
}

// getTagSet returns TagSet by TagSetTableNumber and TagSetSubID values or false found indicator.
func (tagSetIndex *TagSetIndex) getTagSet(zeroBasedTableNum TagSetTableNumber, lower TagSetSubID) (TagSet, bool) {
	if table, ok := tagSetIndex.getTable(zeroBasedTableNum); ok {
		return table.Get(lower)
	}

	return nil, false
}

// Get returns TagSet by its TagSetID value.
// If not found returns empty TagSet and false found indicator.
func (tagSetIndex TagSetIndex) Get(storageIdx TagSetID) (TagSet, bool) {
	if storageIdx.TagSetTableNumber().Int() >= len(tagSetIndex) {
		return nil, false
	}

	return tagSetIndex.getTagSet(storageIdx.TagSetTableNumber(), storageIdx.TagSetSubID())
}
