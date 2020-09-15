package grammemes

// Grammeme name represents name of grammatical category. It uses exactly 4 ASCII characters to name categories.

import (
	"github.com/amarin/binutils"
)

const (
	grammemeNameLength = 4
)

// GrammemeName represents name of grammatical category. Always 4 ASCII characters.
type GrammemeName string

// String returns grammeme name as string type. Implements stringer.
func (g GrammemeName) String() string {
	return string(g)
}

// MarshalBinary makes binary representation bytes slice.
// Always 4 bytes. Empty grammeme name represents as 4 spaces.
func (g GrammemeName) MarshalBinary() (data []byte, err error) {
	if len(g) == 0 {
		return []byte("    "), nil
	}

	res := []byte(g)

	if len(res) != grammemeNameLength {
		return []byte{}, NewErrorf(
			"%T.MarshalBinary(%v)=%v len=%d", g, g, res, len(res))
	}

	return res, nil
}

// UnmarshalFromBuffer loads grammeme name from buffer. If takes 0x20202020 replaces spaces into empty grammeme name.
// Implements buffer.BufferUnmarshaler.
func (g *GrammemeName) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	var dataBytes []byte
	if err = buffer.ReadBytes(&dataBytes, grammemeNameLength); err != nil {
		return err
	}

	*g = GrammemeName(dataBytes)
	// if empty name reduce to empty string
	if *g == "    " {
		*g = ""
	}

	return err
}

// UnmarshalBinary loads grammeme name from bytes slice.
// Requires slice len of grammemeNameLength bytes.
// If takes 0x20202020 replaces spaces into empty grammeme name.
// Returns error if data slice len mismatch.
func (g *GrammemeName) UnmarshalBinary(data []byte) error {
	if len(data) != grammemeNameLength {
		return NewErrorf("expected 4 bytes, not %d", len(data))
	}

	return g.UnmarshalFromBuffer(binutils.NewBuffer(data))
}
