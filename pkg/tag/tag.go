package tag

// Tag is a structured definition of grammatical category.
// It combines base Name name with relation vector from child onto parent.

import (
	"io"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/storage"
)

// Tag implements storage for structured Name's.
type Tag struct {
	Parent Name // Parent Name name.
	Name   Name // Tag name.
}

func (g *Tag) WriteTo(w io.Writer) (int64, error) {
	bytesWritten := int64(0)
	for _, writeFun := range []storage.WriterToFunc{
		g.Parent.WriteTo,
		g.Name.WriteTo,
	} {
		c, err := writeFun(w)
		bytesWritten += c
		if err != nil {
			return bytesWritten, err
		}
	}

	return bytesWritten, nil
}

func (g *Tag) ReadFrom(r io.Reader) (int64, error) {
	bytesTaken := int64(0)
	for _, readFun := range []storage.ReaderFromFunc{
		g.Parent.ReadFrom,
		g.Name.ReadFrom,
	} {
		c, err := readFun(r)
		bytesTaken += c

		if err != nil {
			return bytesTaken, err
		}
	}

	return bytesTaken, nil
}

// BinaryWriteTo writes Tag data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (g *Tag) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if _, err = g.WriteTo(writer); err != nil {
		return err
	}
	return nil
}

// BinaryReadFrom reads Tag data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (g *Tag) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	if _, err = g.ReadFrom(reader); err != nil {
		return err
	}

	return nil
}

// NewTag makes new tag with required parent, name, alias and description.
func NewTag(parent Name, name Name) *Tag {
	if parent == "" {
		parent = EmptyTagName
	}

	return &Tag{
		Parent: parent,
		Name:   name,
	}
}

// String returns string representation of tag. Implements Stringer.
func (g *Tag) String() string {
	return g.Name.String()
}
