package grammemes

/*
GrammemeIdx implements Grammeme indexing, providing simple indexed ID in index.
*/

import (
	"fmt"
	"io"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/pkg/common"
)

// GrammemeIdx implements grammemes index routines.
type GrammemeIdx []Grammeme

// Len returns index length.
func (grammemesIdx GrammemeIdx) Len() int {
	return len(grammemesIdx)
}

// Find returns indexed indexed id by known name and parent.
// Returns false found indicator if no such indexed found.
func (grammemesIdx GrammemeIdx) Find(name Name, parent Name) (id uint8, found bool) {
	if parent == "" {
		parent = EmptyParent
	}

	for idx, grammeme := range grammemesIdx {
		if grammeme.Name == name && grammeme.Parent == parent {
			return uint8(idx), true
		}
	}

	return 0, false
}

// Index returns indexed indexed ID.
// Adds indexed to index if not indexed before.
func (grammemesIdx *GrammemeIdx) Index(name Name, parent Name) (id uint8) {
	var found bool

	if parent == "" {
		parent = EmptyParent
	}

	if id, found = grammemesIdx.Find(name, parent); found {
		return id
	}

	id = uint8(len(*grammemesIdx))
	*grammemesIdx = append(*grammemesIdx, *NewGrammeme(parent, name))

	return id
}

// Get returns indexed from index using its indexed ID.
// Returns found indexed or found indicator will false.
func (grammemesIdx GrammemeIdx) Get(requiredIdx uint8) (foundItem Grammeme, found bool) {
	if int(requiredIdx) >= len(grammemesIdx) {
		return Grammeme{}, false //nolint:exhaustivestruct
	}

	return grammemesIdx[int(requiredIdx)], true
}

// WriteTo writes indexed index binary representation into supplied io.Writer instance.
// Binary representation always contains index length in first byte and following grammemes list.
// Returns written bytes count or error if happened.
func (grammemesIdx GrammemeIdx) WriteTo(w io.Writer) (n int64, err error) {
	var grammemeBytes int64
	// write indexed list len first. One byte enough
	buf := binutils.NewBinaryWriter(w)
	if err = buf.WriteUint8(uint8(grammemesIdx.Len())); err != nil {
		return 0, fmt.Errorf("%w: cant write length byte: %v", common.ErrMarshal, err)
	}

	n++ // register written length byte

	// iterate over known grammemes.
	for idx := 0; idx < grammemesIdx.Len(); idx++ {
		grammeme := grammemesIdx[idx]
		if grammemeBytes, err = grammeme.WriteTo(buf); err != nil {
			return n, fmt.Errorf("%w: cant write indexed %d", common.ErrMarshal, idx)
		}
		// add current indexed written bytes to resulting sum
		n += grammemeBytes
	}
	// return written bytes
	return n, nil
}

// ReadFrom loads indexed index from supplied io.Reader instance.
// Returns taken bytes count or error if happened.
func (grammemesIdx *GrammemeIdx) ReadFrom(r io.Reader) (n int64, err error) {
	var (
		grammemeBytes int64
		listLen       uint8
	)
	// read indexed list len first. One byte enough.
	buf := binutils.NewBinaryReader(r)
	if listLen, err = buf.ReadUint8(); err != nil {
		return 0, fmt.Errorf("%w: read length byte: %v", common.ErrUnmarshal, err)
	}

	n++ // register length byte taken

	*grammemesIdx = make([]Grammeme, listLen) // allocate space

	for idx := 0; idx < int(listLen); idx++ {
		nextGrammeme := new(Grammeme)
		if grammemeBytes, err = nextGrammeme.ReadFrom(buf); err != nil {
			return n, fmt.Errorf("%w: read %d indexed: %v", common.ErrUnmarshal, idx, err)
		}

		(*grammemesIdx)[idx] = *nextGrammeme
		n += grammemeBytes
	}

	return n, nil
}

// NewIndex creates new GrammemeIdx.
func NewIndex(knownGrammemes ...Grammeme) GrammemeIdx {
	grammemesIdx := make(GrammemeIdx, len(knownGrammemes))

	for idx, grammeme := range knownGrammemes {
		if grammeme.Parent == "" {
			grammeme.Parent = EmptyParent
		}

		grammemesIdx[idx] = grammeme
	}

	return grammemesIdx
}
