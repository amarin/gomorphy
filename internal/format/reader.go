package format

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/zeebo/xxh3"
)

type Reader struct {
	r       Container
	size    int64
	version uint32
	entries []Entry
}

func Open(r Container, size int64) (*Reader, error) {
	var head [headerSize]byte

	if _, err := r.ReadAt(head[:], 0); err != nil {
		return nil, err
	}

	if string(head[:4]) != Magic {
		return nil, wrap(ErrBadMagic, string(head[:4]))
	}

	version := binary.LittleEndian.Uint32(head[4:8])
	if version != Version {
		return nil, wrap(ErrUnsupportedVer, fmt.Sprintf("file %d, supported %d", version, Version))
	}

	indexOffset := int64(binary.LittleEndian.Uint64(head[8:16]))
	if indexOffset < headerSize || indexOffset >= size-trailerSize {
		return nil, ErrMalformedCatalog
	}

	trailer := make([]byte, trailerSize)
	if _, err := r.ReadAt(trailer, size-trailerSize); err != nil {
		return nil, err
	}

	want := binary.LittleEndian.Uint64(trailer)
	got, err := checksumRange(r, 0, size-trailerSize)
	if err != nil {
		return nil, err
	}

	if got != want {
		return nil, wrap(ErrBadChecksum, fmt.Sprintf("got %x, want %x", got, want))
	}

	catalog := make([]byte, size-trailerSize-indexOffset)
	if _, err := r.ReadAt(catalog, indexOffset); err != nil {
		return nil, err
	}

	entries, err := parseCatalog(catalog)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.Offset < headerSize || e.Offset+e.Size > indexOffset {
			return nil, wrap(ErrMalformedCatalog, "section "+e.Name+" out of bounds")
		}
	}

	return &Reader{r: r, size: size, version: version, entries: entries}, nil
}

func (r *Reader) Version() uint32 {
	return r.version
}

func (r *Reader) Entries() []Entry {
	out := make([]Entry, len(r.entries))
	copy(out, r.entries)

	return out
}

func (r *Reader) Section(name string) (*io.SectionReader, error) {
	for _, e := range r.entries {
		if e.Name == name {
			return io.NewSectionReader(r.r, e.Offset, e.Size), nil
		}
	}

	return nil, wrap(ErrUnknownSection, name)
}

func (r *Reader) SectionBytes(name string) ([]byte, error) {
	sr, err := r.Section(name)
	if err != nil {
		return nil, err
	}

	data := make([]byte, sr.Size())

	if _, err := io.ReadFull(sr, data); err != nil {
		return nil, err
	}

	return data, nil
}

func parseCatalog(data []byte) ([]Entry, error) {
	if len(data) < 4 {
		return nil, ErrMalformedCatalog
	}

	count := binary.LittleEndian.Uint32(data[0:4])
	p := 4

	entries := make([]Entry, 0, min(uint64(count), 1<<16))

	for range count {
		if p >= len(data) {
			return nil, ErrMalformedCatalog
		}

		nameLen := int(data[p])
		p++

		if p+nameLen+16 > len(data) {
			return nil, ErrMalformedCatalog
		}

		name := string(data[p : p+nameLen])
		p += nameLen

		offset := int64(binary.LittleEndian.Uint64(data[p : p+8]))
		p += 8

		size := int64(binary.LittleEndian.Uint64(data[p : p+8]))
		p += 8

		entries = append(entries, Entry{Name: name, Offset: offset, Size: size})
	}

	return entries, nil
}

func checksumRange(r io.ReaderAt, start, end int64) (uint64, error) {
	const chunk = 1 << 20
	buf := make([]byte, chunk)
	hasher := xxh3.New()

	for off := start; off < end; {
		n := min(end-off, chunk)

		if _, err := r.ReadAt(buf[:n], off); err != nil && err != io.EOF {
			return 0, err
		}

		_, _ = hasher.Write(buf[:n])
		off += n
	}

	return hasher.Sum64(), nil
}
