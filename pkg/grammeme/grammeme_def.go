package grammeme

// Grammeme is a structured definition of grammatical category.
// It combines base grammeme name with relation from child to parent.
// Also adds alias for national lang abbreviation and description string.

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/amarin/gomorphy/internal/text"
)

var (
	// ErrGrammeme indicates grammeme-related errors.
	ErrGrammeme = errors.New("grammeme")
	// ErrGrammemeWrite indicates grammeme write error.
	ErrGrammemeWrite = fmt.Errorf("%w: write", ErrGrammeme)
	// ErrGrammemeRead indicates grammeme read error.
	ErrGrammemeRead = fmt.Errorf("%w: read", ErrGrammeme)
)

// Grammeme implements storage for russian grammatical category structure data.
type Grammeme struct {
	ParentAttr  Name             // Parent grammeme name.
	Name        Name             // Grammeme name.
	Alias       text.RussianText // Localized grammeme name.
	Description text.RussianText // Grammeme description.
}

// NewGrammeme makes new grammeme with required parent, name, alias and description.
func NewGrammeme(parent Name, name Name, alias text.RussianText, desc text.RussianText) *Grammeme {
	return &Grammeme{ParentAttr: parent, Name: name, Alias: alias, Description: desc}
}

// String returns string representation of grammeme. Implements Stringer.
func (g Grammeme) String() string {
	return "Grammeme{" + g.ParentAttr.String() + "," + g.Name.String() + "," +
		g.Alias.String() + "," + g.Description.String() + "}"
}

// ReadFrom loads grammeme data from specified reader.
func (g *Grammeme) ReadFrom(r io.Reader) (n int64, err error) {
	var int64bytes int64

	if int64bytes, err = g.Name.ReadFrom(r); err != nil {
		return int64bytes, fmt.Errorf("%v: name: %w", ErrGrammemeRead, err)
	}

	n += int64bytes

	if int64bytes, err = g.ParentAttr.ReadFrom(r); err != nil {
		return n + int64bytes, fmt.Errorf("%v: parent: %w", ErrGrammemeRead, err)
	}

	n += int64bytes

	if int64bytes, err = g.Alias.ReadFrom(r); err != nil {
		return n + int64bytes, fmt.Errorf("%v: alias: %w", ErrGrammemeRead, err)
	}

	n += int64bytes

	if int64bytes, err = g.Description.ReadFrom(r); err != nil {
		return n + int64bytes, fmt.Errorf("%v: description: %w", ErrGrammemeRead, err)
	}

	return n + int64bytes, nil
}

// WriteTo writes grammeme definition into specified io.Writer instance.
// Returns written bytes count and any underlying errors if occurs.
// Implements io.WriterTo.
func (g Grammeme) WriteTo(w io.Writer) (n int64, err error) {
	var (
		written   int
		written64 int64
	)

	if written, err = w.Write([]byte(g.Name)); err != nil {
		return n, fmt.Errorf("%w: name: %v", ErrGrammemeWrite, err)
	}

	n += int64(written)

	if written, err = w.Write([]byte(g.ParentAttr)); err != nil {
		return n, fmt.Errorf("%w: parent: %v", ErrGrammemeWrite, err)
	}

	n += int64(written)

	if written64, err = g.Alias.WriteTo(w); err != nil {
		return n, fmt.Errorf("%w: alias: %v", ErrGrammemeWrite, err)
	}

	n += written64

	if written64, err = g.Description.WriteTo(w); err != nil {
		return n, fmt.Errorf("%w: description: %v", ErrGrammemeWrite, err)
	}

	n += written64

	return n, nil
}

// MarshalBinary makes binary grammeme data.
func (g Grammeme) MarshalBinary() (res []byte, err error) {
	buffer := bytes.NewBuffer(make([]byte, 0))
	if _, err = g.WriteTo(buffer); err != nil {
		return buffer.Bytes(), fmt.Errorf("%v: write: %w", ErrGrammeme, err)
	}

	return buffer.Bytes(), nil
}

// UnmarshalBinary loads grammeme data from specified byte string.
// Returns error if unmarshal failed.
func (g *Grammeme) UnmarshalBinary(data []byte) error {
	_, err := g.ReadFrom(bytes.NewReader(data))

	return fmt.Errorf("%w: %v", ErrGrammemeRead, err)
}
