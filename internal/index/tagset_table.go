package index

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amarin/binutils"
)

const prefixTagSetTable = "TT"

// TagSetTable stores unique TagSet lists having equal length.
// It Uses its own ID16 index to address target sets.set16 stack in addition to address element in stack.
// Used in TagSetIndex to store different sized sets organized into stacks of equal-sized sets.
type TagSetTable []TagSet

// String returns string representation of TagSetTable.
// Implements fmt.Stringer.
func (tagSetTable TagSetTable) String() string {
	tagSetStrings := make([]string, tagSetTable.Len())
	for idx, ts := range tagSetTable {
		tagSetStrings[idx] = strconv.Itoa(idx) + ": " + ts.String()
	}

	return prefixTagSetTable + "(" + strings.Join(tagSetStrings, ",") + ")"
}

// BinaryReadFrom reads TagSetTable data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (tagSetTable *TagSetTable) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var tagSetLen uint16

	if reader == nil {
		return fmt.Errorf("%w: reader nil", Error)
	}

	if tagSetLen, err = reader.ReadUint16(); err != nil {
		return fmt.Errorf("%w: read: tagset: %v", Error, err)
	}

	*tagSetTable = make(TagSetTable, tagSetLen)
	for idx := 0; idx < int(tagSetLen); idx++ {
		if err = (*tagSetTable)[idx].BinaryReadFrom(reader); err != nil {
			return fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
	}

	return nil
}

// BinaryWriteTo writes TagSetTable data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagSetTable TagSetTable) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if writer == nil {
		return fmt.Errorf("%w: writer nil", Error)
	}
	if err = writer.WriteUint16(uint16(tagSetTable.Len())); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tagSet := range tagSetTable {
		if err = tagSet.BinaryWriteTo(writer); err != nil {
			return err
		}
	}

	return nil
}

// Find returns 0-based index of ids.Set16 in Table16 storage.
// If no such set found returns 0 and false found indicator.
// It required sets.Set16 argument to be sorted before.
func (tagSetTable TagSetTable) Find(item TagSet) (storageIdx TagSetSubID, found bool) {
	for id, existedSet := range tagSetTable {
		if existedSet.EqualTo(item) {
			return TagSetSubID(id), true
		}
	}

	return 0, false
}

// Index returns 0-based TagSetSubID index of ids.Set16 in Table16 instance.
// Returns index of existed item if found or of appended item.
// Panics if specified set empty.
func (tagSetTable *TagSetTable) Index(item TagSet) (storageIdx TagSetSubID) {
	var found bool

	if len(item) == 0 {
		panic("empty set")
	}

	item.Sort() // sort item before find or adding to index.

	if storageIdx, found = tagSetTable.Find(item); found {
		return storageIdx
	}

	storageIdx = TagSetSubID(len(*tagSetTable))

	*tagSetTable = append(*tagSetTable, item)

	return storageIdx
}

// Get returns ids.Set16 by index if present or found indicator will be false.
func (tagSetTable TagSetTable) Get(storageIdx TagSetSubID) (tagSet TagSet, found bool) {
	if storageIdx.Int() >= tagSetTable.Len() {
		return nil, false
	}

	return tagSetTable[storageIdx], true
}

// Len returns length of TagSetTable in TagSet items.
func (tagSetTable TagSetTable) Len() int {
	return len(tagSetTable)
}
