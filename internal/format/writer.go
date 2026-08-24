package format

import (
	"encoding/binary"
	"hash"
	"io"

	"github.com/zeebo/xxh3"
)

type Container interface {
	io.ReaderAt
	io.WriterAt
}

type Entry struct {
	Name   string
	Offset int64
	Size   int64
}

type Writer struct {
	w         Container
	hasher    hash.Hash64
	entries   []Entry
	pos       int64
	openName  string
	finalized bool
}

func NewWriter(w Container) *Writer {
	return &Writer{
		w:      w,
		hasher: xxh3.New(),
	}
}

func (w *Writer) WriteHeader(version uint32) error {
	var head [headerSize]byte

	copy(head[:4], Magic)
	binary.LittleEndian.PutUint32(head[4:8], version)

	if _, err := w.w.WriteAt(head[:], 0); err != nil {
		return err
	}

	w.pos = headerSize

	return nil
}

func (w *Writer) Create(name string) (io.WriteCloser, error) {
	switch {
	case w.finalized:
		return nil, ErrAlreadyFinalized
	case w.openName != "":
		return nil, wrap(ErrSectionNotClosed, w.openName)
	case len(name) == 0 || len(name) > maxNameLength:
		return nil, ErrNameTooLong
	case w.find(name) >= 0:
		return nil, wrap(ErrDuplicateSection, name)
	}

	offset := w.pos
	w.openName = name

	return &sectionWriter{writer: w, name: name, offset: offset}, nil
}

func (w *Writer) WriteSection(name string, data []byte) error {
	sw, err := w.Create(name)
	if err != nil {
		return err
	}

	if _, err := sw.Write(data); err != nil {
		return err
	}

	return sw.Close()
}

func (w *Writer) closeSection(name string, size int64) error {
	w.entries = append(w.entries, Entry{Name: name, Offset: w.pos, Size: size})
	w.openName = ""
	w.pos += size

	return nil
}

func (w *Writer) Finalize() error {
	switch {
	case w.finalized:
		return ErrAlreadyFinalized
	case w.openName != "":
		return wrap(ErrSectionNotClosed, w.openName)
	}

	indexOffset := w.pos

	if err := w.writeCatalog(indexOffset); err != nil {
		return err
	}

	var idx [8]byte
	binary.LittleEndian.PutUint64(idx[:], uint64(indexOffset))

	if _, err := w.w.WriteAt(idx[:], 8); err != nil {
		return err
	}

	end := indexOffset + int64(w.catalogSize())

	if _, err := w.hashRange(0, end); err != nil {
		return err
	}

	var trailer [trailerSize]byte
	binary.LittleEndian.PutUint64(trailer[:], w.hasher.Sum64())

	if _, err := w.w.WriteAt(trailer[:], end); err != nil {
		return err
	}

	w.pos = end + trailerSize
	w.finalized = true

	return nil
}

func (w *Writer) catalogSize() int {
	size := 4
	for _, e := range w.entries {
		size += 1 + len(e.Name) + 8 + 8
	}

	return size
}

func (w *Writer) writeCatalog(indexOffset int64) error {
	buf := make([]byte, w.catalogSize())
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(w.entries)))

	p := 4
	for _, e := range w.entries {
		buf[p] = byte(len(e.Name))
		p++

		p += copy(buf[p:], e.Name)
		binary.LittleEndian.PutUint64(buf[p:p+8], uint64(e.Offset))
		p += 8

		binary.LittleEndian.PutUint64(buf[p:p+8], uint64(e.Size))
		p += 8
	}

	if _, err := w.w.WriteAt(buf, indexOffset); err != nil {
		return err
	}

	return nil
}

func (w *Writer) hashRange(start, end int64) (int64, error) {
	w.hasher.Reset()

	const chunk = 1 << 20
	buf := make([]byte, chunk)

	var total int64

	for off := start; off < end; {
		n := min(end-off, chunk)

		if _, err := w.w.ReadAt(buf[:n], off); err != nil && err != io.EOF {
			return 0, err
		}

		_, _ = w.hasher.Write(buf[:n])
		total += n
		off += n
	}

	return total, nil
}

func (w *Writer) find(name string) int {
	for i, e := range w.entries {
		if e.Name == name {
			return i
		}
	}

	return -1
}

type sectionWriter struct {
	writer *Writer
	name   string
	offset int64
	size   int64
	closed bool
}

func (sw *sectionWriter) Write(p []byte) (int, error) {
	if sw.closed {
		return 0, ErrSectionClosed
	}

	n, err := sw.writer.w.WriteAt(p, sw.offset+sw.size)
	sw.size += int64(n)

	return n, err
}

func (sw *sectionWriter) Close() error {
	if sw.closed {
		return ErrAlreadyFinalized
	}

	sw.closed = true

	return sw.writer.closeSection(sw.name, sw.size)
}
