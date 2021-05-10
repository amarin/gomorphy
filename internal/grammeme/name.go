package grammeme

// Grammeme name represents name of grammatical category. It uses exactly 4 ASCII characters to name categories.

import (
	"fmt"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/pkg/common"
)

const (
	grammemeNameLength = 4
)

// Name represents name of grammatical category. Always 4 ASCII characters.
type Name string

// String returns grammeme name as string type. Implements stringer.
func (g Name) String() string {
	return string(g)
}

// MarshalBinary makes binary representation bytes slice.
// Always 4 bytes. Empty grammeme name represents as 4 spaces.
func (g Name) MarshalBinary() (data []byte, err error) {
	if len(g) == 0 {
		return []byte("    "), nil
	}

	res := []byte(g)

	if len(res) != grammemeNameLength {
		return []byte{}, fmt.Errorf("%w: %T.MarshalBinary(%v)=%v len=%d", common.ErrMarshal, g, g, res, len(res))
	}

	return res, nil
}

// UnmarshalFromBuffer loads grammeme name from buffer. If takes 0x20202020 replaces spaces into empty grammeme name.
// Implements buffer.BufferUnmarshaler.
func (g *Name) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	var dataBytes []byte
	if err = buffer.ReadBytes(&dataBytes, grammemeNameLength); err != nil {
		return err
	}

	*g = Name(dataBytes)
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
func (g *Name) UnmarshalBinary(data []byte) error {
	if len(data) != grammemeNameLength {
		return fmt.Errorf("%w: expected 4 bytes, not %d", common.ErrUnmarshal, len(data))
	}

	return g.UnmarshalFromBuffer(binutils.NewBuffer(data))
}
