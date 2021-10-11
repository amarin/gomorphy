package grammemes

// Grammeme name represents name of grammatical category. It uses exactly 4 ASCII characters to name categories.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

const (
	grammemeNameLength = 4
)

var (
	ErrName   = errors.New("name")
	emptyName = []byte{0x20, 0x20, 0x20, 0x20}
)

// Name represents name of grammatical category. It always consists of 4 ASCII characters.
type Name string

// WriteTo writes name data which is always 4-bytes into specified writer.
// Returns written bytes count and any error if occurs.
// Implements io.WriterTo.
func (g *Name) WriteTo(w io.Writer) (n int64, err error) {
	var (
		nameBytes    []byte
		bytesWritten int
	)

	if nameBytes, err = g.MarshalBinary(); err != nil {
		return 0, err
	}

	bytesWritten, err = w.Write(nameBytes)

	switch {
	case err != nil:
		return int64(bytesWritten), fmt.Errorf("%v: write: %w", ErrName, err)
	case bytesWritten != grammemeNameLength:
		return int64(bytesWritten), fmt.Errorf(
			"%w: written %v bytes of %v", ErrName, bytesWritten, grammemeNameLength)
	}

	return int64(bytesWritten), nil
}

// ReadFrom loads name data from specified io.Reader instance.
// Returns taken bytes count and any error if occurs. It expected it takes exactly 4 bytes if no errors.
// Implements io.ReaderFrom.
func (g *Name) ReadFrom(r io.Reader) (n int64, err error) {
	bytesTaken := 0
	nameBytes := make([]byte, grammemeNameLength)
	isEmpty := false
	bytesTaken, err = r.Read(nameBytes)

	switch {
	case err != nil:
		return int64(bytesTaken), fmt.Errorf("%v: read next byte: %w", ErrName, err)
	case bytesTaken != grammemeNameLength:
		return int64(bytesTaken), fmt.Errorf("%v: expected 4 bytes: %w", ErrName, err)
	}

	for idx := 0; idx < len(emptyName); idx++ {
		isEmpty = nameBytes[idx] == emptyName[idx]
		if !isEmpty {
			break
		}
	}

	if isEmpty || string(nameBytes) == "    " {
		*g = Empty
	} else {
		*g = Name(nameBytes)
	}

	return int64(bytesTaken), nil
}

// String returns grammeme name as string type. Implements fmt.Stringer.
func (g Name) String() string {
	return string(g)
}

// MarshalBinary makes name bytes string representation.
// Always produces 4 bytes. Empty grammeme name represents as 4 spaces.
func (g Name) MarshalBinary() (data []byte, err error) {
	if len(g) == 0 || g == Empty {
		return []byte("    "), nil
	}

	res := []byte(g)

	if len(res) != grammemeNameLength {
		return []byte{}, fmt.Errorf("%w: %T.MarshalBinary(%v)=%v len=%d", ErrName, g, g, res, len(res))
	}

	return res, nil
}

// UnmarshalBinary loads grammeme name from bytes slice.
// Expects 4-bytes len byte string.
// If it takes 0x20202020 (4 spaces) it replaces spaces into empty grammeme name.
// Returns error if data slice len mismatch.
func (g *Name) UnmarshalBinary(data []byte) (err error) {
	if len(data) != grammemeNameLength {
		return fmt.Errorf("%w: expected 4 bytes, not %d", ErrName, len(data))
	}

	_, err = g.ReadFrom(bytes.NewReader(data))

	return err
}
