package grammemes

// Grammeme is a structured definition of grammatical category.
// It combines base grammeme name with relation vector from child onto parent.

import (
	"errors"
	"fmt"
	"io"
)

// Error indicates grammeme-related errors.
var Error = errors.New("grammeme")

const EmptyParent Name = "----"

// Grammeme implements storage for russian grammatical category structure data.
type Grammeme struct {
	Parent Name // Parent grammeme name.
	Name   Name // Grammeme name.
	// Alias       text.RussianText // Localized grammeme name.
	// Description text.RussianText // Grammeme description.
}

// NewGrammeme makes new grammeme with required parent, name, alias and description.
func NewGrammeme(parent Name, name Name) *Grammeme {
	if parent == "" {
		parent = EmptyParent
	}

	return &Grammeme{
		Parent: parent,
		Name:   name,
		// Alias: alias,
		// Description: desc,
	}
}

// String returns string representation of grammeme. Implements Stringer.
func (g Grammeme) String() string {
	return "Grammeme{" + g.Parent.String() + "," + g.Name.String() + "}"
}

// ReadFrom loads grammeme data from specified reader.
func (g *Grammeme) ReadFrom(r io.Reader) (n int64, err error) {
	var int64bytes int64

	if int64bytes, err = g.Name.ReadFrom(r); err != nil {
		return int64bytes, fmt.Errorf("%v: read: name: %w", Error, err)
	}

	n += int64bytes

	if int64bytes, err = g.Parent.ReadFrom(r); err != nil {
		return n + int64bytes, fmt.Errorf("%v: read: parent: %w", Error, err)
	}

	// n += int64bytes
	//
	// if int64bytes, err = grammemes.Alias.ReadFrom(r); err != nil {
	// 	return n + int64bytes, fmt.Errorf("%v: alias: %w", ErrGrammemeRead, err)
	// }
	//
	// n += int64bytes
	//
	// if int64bytes, err = grammemes.Description.ReadFrom(r); err != nil {
	// 	return n + int64bytes, fmt.Errorf("%v: description: %w", ErrGrammemeRead, err)
	// }

	return n + int64bytes, nil
}

// WriteTo writes grammeme definition into specified io.Writer instance.
// Returns written bytes count and any underlying errors if occurs.
// Implements io.WriterTo.
func (g Grammeme) WriteTo(w io.Writer) (n int64, err error) {
	var written int

	if written, err = w.Write([]byte(g.Name)); err != nil {
		return n, fmt.Errorf("%w: write: name: %v", Error, err)
	}

	n += int64(written)

	if written, err = w.Write([]byte(g.Parent)); err != nil {
		return n, fmt.Errorf("%w: write: parent: %v", Error, err)
	}

	// n += int64(written)
	//
	// if written64, err = grammemes.Alias.WriteTo(w); err != nil {
	// 	return n, fmt.Errorf("%w: alias: %v", ErrGrammemeWrite, err)
	// }
	//
	// n += written64
	//
	// if written64, err = grammemes.Description.WriteTo(w); err != nil {
	// 	return n, fmt.Errorf("%w: description: %v", ErrGrammemeWrite, err)
	// }
	// n += written64

	return n + int64(written), nil
}
