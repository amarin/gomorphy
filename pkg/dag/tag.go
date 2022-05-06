package dag

// Tag is a structured definition of grammatical category.
// It combines base TagName name with relation vector from child onto parent.

import (
	"fmt"

	"github.com/amarin/binutils"
)

// Tag implements storage for structured TagName's.
type Tag struct {
	Parent TagName // Parent TagName name.
	Name   TagName // Tag name.
}

// BinaryWriteTo writes Tag data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (g Tag) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = g.Parent.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: name: %v", Error, err)
	}

	if err = g.Name.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: parent: %v", Error, err)
	}

	return nil
}

// BinaryReadFrom reads Tag data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (g *Tag) BinaryReadFrom(reader *binutils.BinaryReader) (n int64, err error) {
	var currentBytes int64

	n = 0

	if currentBytes, err = g.Parent.BinaryReadFrom(reader); err != nil {
		return 0, fmt.Errorf("%w: tag: read: %v", Error, err)
	}
	n += currentBytes

	if currentBytes, err = g.Name.BinaryReadFrom(reader); err != nil {
		return 0, fmt.Errorf("%w: tag: read: %v", Error, err)
	}
	n += currentBytes

	return n, nil
}

// NewTag makes new tag with required parent, name, alias and description.
func NewTag(parent TagName, name TagName) *Tag {
	if parent == "" {
		parent = EmptyTagName
	}

	return &Tag{
		Parent: parent,
		Name:   name,
	}
}

// String returns string representation of tag. Implements Stringer.
func (g Tag) String() string {
	return g.Name.String()
}
