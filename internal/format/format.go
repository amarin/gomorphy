// Package format implements the compiled dictionary container:
// varint/delta codecs, sectioned writer and verified reader.
//
// File layout (all integers little-endian):
//
//	offset 0:  magic "GMRF"
//	offset 4:  format version, uint32
//	offset 8:  indexOffset, uint64 - absolute offset of the section catalog
//	offset 20: sections data, sequential
//	indexOffset: catalog: count uint32, then entries:
//	    nameLen uint8, name bytes, offset uint64, size uint64
//	size-8: checksum uint64, xxh3-64 of all preceding bytes
package format

import (
	"errors"
	"fmt"
)

const (
	Magic   = "GMRF"
	Version = uint32(1)

	headerSize    = 4 + 4 + 8
	trailerSize   = 8
	maxNameLength = 255
)

var (
	ErrBadMagic         = errors.New("format: bad magic")
	ErrUnsupportedVer   = errors.New("format: unsupported version")
	ErrBadChecksum      = errors.New("format: bad checksum")
	ErrUnknownSection   = errors.New("format: unknown section")
	ErrDuplicateSection = errors.New("format: duplicate section")
	ErrSectionNotClosed = errors.New("format: previous section not closed")
	ErrSectionClosed    = errors.New("format: section already closed")
	ErrNotFinalized     = errors.New("format: not finalized")
	ErrAlreadyFinalized = errors.New("format: already finalized")
	ErrNameTooLong      = errors.New("format: section name too long")
	ErrMalformedDelta   = errors.New("format: malformed delta data")
	ErrMalformedCatalog = errors.New("format: malformed catalog")
)

func wrap(err error, msg string) error {
	return fmt.Errorf("%w: %s", err, msg)
}
