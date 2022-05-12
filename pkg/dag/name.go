package dag

// TagName represents name of grammatical category.

import (
	"errors"
	"fmt"

	"github.com/amarin/binutils"
)

const (
	// EmptyTagName defines constant tag name to replace empty strings and distinct unfilled and empty cases.
	EmptyTagName TagName = "----"

	tagNameLen = 4
)

var (
	// Error indicates grammar-related errors.
	Error = errors.New("grammar")

	emptyTagNameBytes = []byte{0x20, 0x20, 0x20, 0x20}
)

// TagName represents name of grammatical category. It uses exactly 4 ASCII characters to name categories.
// If TagName is shorter than 4 character is filled upto 4 with spaces.
type TagName string

func (g *TagName) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var tagNameBytes []byte
	if tagNameBytes, err = reader.ReadBytesCount(tagNameLen); err != nil {
		return fmt.Errorf("%w: tag name: read: %v", Error, err)
	}

	if err = g.UnmarshalBinary(tagNameBytes); err != nil {
		return err
	}

	return nil
}

// BinaryWriteTo writes TagName data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (g TagName) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	var nameBytes []byte

	if nameBytes, err = g.MarshalBinary(); err != nil {
		return err
	}

	if err = writer.WriteBytes(nameBytes); err != nil {
		return fmt.Errorf("%w: tag name: write: %v", Error, err)
	}

	return nil
}

// String returns TagName name as string type. Implements fmt.Stringer.
func (g TagName) String() string {
	return string(g)
}

// MarshalBinary makes name bytes string representation.
// Always produces 4 bytes. EmptyTagName TagName name represents as 4 spaces.
func (g TagName) MarshalBinary() (data []byte, err error) {
	if len(g) == 0 || g == EmptyTagName {
		return []byte("    "), nil
	}

	res := []byte(g)

	if len(res) != tagNameLen {
		return []byte{}, fmt.Errorf("%w: %T.MarshalBinary(%v)=%v len=%d", Error, g, g, res, len(res))
	}

	return res, nil
}

func (g *TagName) UnmarshalBinary(tagNameBytes []byte) error {
	if len(tagNameBytes) != tagNameLen {
		return fmt.Errorf("%w: tag name: read: expect %d bytes, not %d", Error, tagNameLen, len(tagNameBytes))
	}

	isEmpty := false
	for idx := 0; idx < len(emptyTagNameBytes); idx++ {
		isEmpty = tagNameBytes[idx] == emptyTagNameBytes[idx]
		if !isEmpty {
			break
		}
	}

	if isEmpty || string(tagNameBytes) == "    " {
		*g = EmptyTagName
	} else {
		*g = TagName(tagNameBytes)
	}

	return nil
}
