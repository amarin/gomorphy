package dag

/*
Idx implements Tag ids, providing simple indexed ID in index.
*/

import (
	"fmt"

	"github.com/amarin/gomorphy/pkg/storage"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/common"
)

// TagID represents Tag id in storage array.
type TagID storage.ID8

// Uint8 returns uint8 value of TagID.
func (t TagID) Uint8() uint8 {
	return uint8(t)
}

// Idx implements Tag index routines.
type Idx []Tag

// NewIndex creates new Idx.
func NewIndex(knownTags ...Tag) Idx {
	tagsIndex := make(Idx, len(knownTags))

	for idx, tag := range knownTags {
		if tag.Parent == "" {
			tag.Parent = EmptyTagName
		}

		tagsIndex[idx] = tag
	}

	return tagsIndex
}

// BinaryReadFrom reads Idx data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (tagsIndex *Idx) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var listLen uint8

	if listLen, err = reader.ReadUint8(); err != nil {
		return fmt.Errorf("%w: read length byte: %v", common.ErrUnmarshal, err)
	}

	*tagsIndex = make(Idx, listLen) // allocate space

	for idx := 0; idx < int(listLen); idx++ {
		if err = (*tagsIndex)[idx].BinaryReadFrom(reader); err != nil {
			return fmt.Errorf("%w: read %d indexed: %v", common.ErrUnmarshal, idx, err)
		}
	}

	return nil
}

// BinaryWriteTo writes Idx data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (tagsIndex Idx) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint8(uint8(tagsIndex.Len())); err != nil {
		return fmt.Errorf("%w: cant write length byte: %v", common.ErrMarshal, err)
	}

	// iterate over known Tag's.
	for idx := 0; idx < tagsIndex.Len(); idx++ {
		if err = tagsIndex[idx].BinaryWriteTo(writer); err != nil {
			return fmt.Errorf("%w: cant write indexed %d", common.ErrMarshal, idx)
		}
	}
	return nil
}

// Len returns index length.
func (tagsIndex Idx) Len() int {
	return len(tagsIndex)
}

// Find returns indexed ID by known name and parent.
// Returns false found indicator if no such indexed found.
func (tagsIndex Idx) Find(name TagName) (id TagID, found bool) {
	for idx, tag := range tagsIndex {
		if string(tag.Name) == string(name) {
			return TagID(idx), true
		}
	}

	return 0, false
}

// Index returns indexed ID.
// Adds indexed to index if not indexed before.
func (tagsIndex *Idx) Index(name TagName, parent TagName) (id TagID) {
	var found bool

	if parent == "" {
		parent = EmptyTagName
	}

	if id, found = tagsIndex.Find(name); found {
		return id
	}

	id = TagID(len(*tagsIndex))
	*tagsIndex = append(*tagsIndex, *NewTag(parent, name))

	return id
}

// Get returns indexed from index using its indexed ID.
// Returns found indexed or found indicator will false.
func (tagsIndex Idx) Get(requiredIdx TagID) (foundItem Tag, found bool) {
	if int(requiredIdx) >= len(tagsIndex) {
		return Tag{}, false //nolint:exhaustivestruct
	}

	return tagsIndex[int(requiredIdx)], true
}
