package dag

// Tag is a structured definition of grammatical category.
// It combines base Name name with relation vector from child onto parent.

import (
	"fmt"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/tag"
)

// Tag implements storage for structured Name's.
type Tag struct {
	Parent tag.Name // Parent Name name.
	Name   tag.Name // Tag name.
}

// BinaryWriteTo writes Tag data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (g Tag) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = g.Parent.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: name: %v", tag.Error, err)
	}

	if err = g.Name.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: parent: %v", tag.Error, err)
	}

	return nil
}

// BinaryReadFrom reads Tag data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (g *Tag) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	if err = g.Parent.BinaryReadFrom(reader); err != nil {
		return fmt.Errorf("%w: tag: read: %v", tag.Error, err)
	}

	if err = g.Name.BinaryReadFrom(reader); err != nil {
		return fmt.Errorf("%w: tag: read: %v", tag.Error, err)
	}

	return nil
}

// NewTag makes new tag with required parent, name, alias and description.
func NewTag(parent tag.Name, name tag.Name) *Tag {
	if parent == "" {
		parent = tag.EmptyTagName
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
