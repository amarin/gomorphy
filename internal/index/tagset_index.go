package index

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/storage"
)

const binaryTagSetPrefix = "TI"

// TableID represents TagSetTable ID in TagSetIndex.
type TableID storage.ID32

// TagSetIndex stores unique ids.Set16 items organized in Table16 storages where each storage
// keeps sets of same sizes.
// Each TagSetIndex item is addressable by unique ids.ID32
// which consists of Table16 ID16 item index in collection and ID16 element index in Table16 item.
type TagSetIndex []TagSetTable

// String returns string representation of TagSetIndex.
// Implements fmt.Stringer.
func (tagSetIndex TagSetIndex) String() string {
	tableStrings := make([]string, tagSetIndex.Len())
	for idx, table := range tagSetIndex {
		tableStrings[idx] = strconv.Itoa(idx) + ": " + table.String()
	}

	return binaryTagSetPrefix + "(" + strings.Join(tableStrings, ",") + ")"
}

// Len returns length of TagSetIndex instance.
func (tagSetIndex *TagSetIndex) Len() int {
	return len(*tagSetIndex)
}

// BinaryWriteTo writes TagSetIndex data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagSetIndex *TagSetIndex) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint32(uint32(tagSetIndex.Len())); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tagSetTable := range *tagSetIndex {
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

// Find returns 0-based index of ids.Set16 item in TagSetIndex array.
// If no such set found returns 0 and false found flag.
func (tagSetIndex *TagSetIndex) Find(item TagSet) (storageIdx TableID, found bool) {
	var lower TagSetID

	columnIdx := len(item) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if table, ok := tagSetIndex.getTable(storage.ID16(columnIdx)); ok {
		if lower, ok = table.Find(item); ok {
			return TableID(storage.Combine16(storage.ID16(columnIdx), lower.ID16())), ok
		}
	}

	return 0, false
}

// Index returns 0-based index of set in Collection8x8 array.
func (tagSetIndex *TagSetIndex) Index(item TagSet) (storageIdx TableID) {
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

	return TableID(storage.Combine16(storage.ID16(columnIdx), itemIdx.ID16()))
}

func (tagSetIndex *TagSetIndex) getTable(upper storage.ID16) (table TagSetTable, ok bool) {
	if ok = int(upper) < tagSetIndex.Len(); !ok {
		return nil, false
	}

	return (*tagSetIndex)[upper], ok
}

func (tagSetIndex *TagSetIndex) getUpperLower(upper storage.ID16, lower storage.ID16) (TagSet, bool) {
	if table, ok := tagSetIndex.getTable(upper); ok {
		return table.Get(TagSetID(lower))
	}

	return nil, false
}

// Get returns set by index.
func (tagSetIndex *TagSetIndex) Get(storageIdx TableID) (TagSet, bool) {
	if storageIdx == 0 {
		return make(TagSet, 0), true // zero-index TableID always exists and means no data
	}

	if int(storage.ID32(storageIdx).Upper16()) >= tagSetIndex.Len() {
		return nil, false
	}

	return tagSetIndex.getUpperLower(storage.ID32(storageIdx).Upper16(), storage.ID32(storageIdx).Lower16())
}
