package internal

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeebo/xxh3"
)

// The GMOR dictionary's unified on-disk format.
//
//	┌──────────────────────────────────┐
//	│  Header                          │
//	│  magic  "GMOR"  4 bytes          │
//	│  version    u32                  │
//	│  checksum   xxh3-64  8 bytes     │
//	├──────────────────────────────────┤
//	│  Section catalog                 │
//	│  count      u16                  │
//	│  entries:                        │
//	│    name     [16]byte             │
//	│    offset   u64                  │
//	│    size     u64                  │
//	│    flags    u8                   │
//	├──────────────────────────────────┤
//	│  Data sections                   │
//	└──────────────────────────────────┘
//
// The checksum covers everything after the checksum field itself (the
// catalog + sections). Section offsets are 8-byte aligned so the
// words.dawg section can be aliased from mmap without copying (offset+4
// is a multiple of 4).
//
// flags: the low 4 bits are the section's compression algorithm id (see
// Compression*, 0 = uncompressed), the high 4 bits are reserved for
// future flags independent of compression. The algorithm is recorded
// explicitly (not guessed from the section's data signature): a section
// is an arbitrary blob with no self-describing header, and an explicit
// id gives a precise, unambiguous diagnostic for a file from a future
// version with an algorithm this reader doesn't know ("unsupported
// compression algorithm 3, upgrade required") instead of trying to guess
// the format from the first bytes. Compression is chosen per section
// (not for the whole file): words.dawg stays uncompressed so it can be
// aliased from mmap without copying; the choice of algorithm for the
// other sections is up to the compression implementation itself (see
// docs/en/todo.md, "Stage 17").
const (
	magicHeader = "GMOR"
	headerSize  = 16 // magic(4) + version(4) + checksum(8)
	nameSize    = 16
	entrySize   = nameSize + 8 + 8 + 1 // 33 bytes per catalog entry
)

const (
	// Version is the GMOR format version.
	Version uint32 = 1
)

// Section compression algorithms (the low 4 bits of Entry.Flags/
// Section.Flags). Adding a new algorithm means adding a constant and a
// branch in Container.Section/validateSections, with no change to the
// catalog's byte layout and no versioning needed: compression has never
// been written to any released file yet, so this nibble is free for
// full control right now, before the 1.0 release.
const (
	CompressionNone uint8 = 0
	CompressionZstd uint8 = 1

	// compressionMask extracts the algorithm id from a catalog entry's flags.
	compressionMask uint8 = 0x0F
	// maxKnownCompression is the upper bound of known algorithm ids;
	// extend it as new Compression* constants are added.
	maxKnownCompression = CompressionZstd
)

// compression returns a section's compression algorithm id from its
// catalog entry flags.
func compression(flags uint8) uint8 { return flags & compressionMask }

// Format errors.
var (
	ErrBadMagic               = errors.New("format: bad magic")
	ErrUnsupportedVersion     = errors.New("format: unsupported version")
	ErrBadChecksum            = errors.New("format: checksum mismatch")
	ErrMalformedFile          = errors.New("format: malformed file")
	ErrUnknownSection         = errors.New("format: unknown section")
	ErrUnsupportedCompression = errors.New("format: unsupported compression algorithm")
)

// Section is a section to be written: its data and flags.
type Section struct {
	Name  string
	Data  []byte
	Flags uint8
}

// Entry is a section catalog entry.
type Entry struct {
	Name   string
	Offset int64
	Size   int64
	Flags  uint8
}

// Container is an opened GMOR file. Section data aliases the original
// data slice (zero-copy for an mmap region).
type Container struct {
	data    []byte
	entries []Entry
}

// Entries returns a copy of the section catalog.
func (c *Container) Entries() []Entry {
	out := make([]Entry, len(c.entries))
	copy(out, c.entries)
	return out
}

// Section returns a section's data by name (the slice aliases the file)
// and its flags.
func (c *Container) Section(name string) ([]byte, uint8, error) {
	for _, e := range c.entries {
		if e.Name == name {
			if algo := compression(e.Flags); algo != CompressionNone {
				return nil, e.Flags, wrap(ErrUnsupportedCompression, fmt.Sprintf("%s (algorithm %d)", name, algo))
			}
			if e.Size == 0 {
				return nil, e.Flags, nil
			}
			return c.data[int(e.Offset):int(e.Offset+e.Size)], e.Flags, nil
		}
	}
	return nil, 0, wrap(ErrUnknownSection, name)
}

// SaveContainer writes a GMOR file: header + catalog + sections. Section
// offsets are aligned to 8 bytes.
func SaveContainer(path string, sections []Section) error {
	if err := validateSections(sections); err != nil {
		return err
	}

	catalogLen, placed := layoutSections(sections)

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("format: mkdir %s: %w", dir, err)
		}
	}

	// Write to a temp file and rename into place on success, so a failure
	// partway through (disk full, process killed) never leaves a truncated
	// file at path — Open() would reject it by checksum, but a broken file
	// sitting where a working one is expected is still a bad experience.
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("format: create temp file: %w", err)
	}
	tmpPath := f.Name()
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmpPath) // no-op once renamed to path
	}()

	if err := writeSections(f, sections, placed, catalogLen); err != nil {
		return err
	}
	if err := finalizeChecksum(f); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("format: close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("format: rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

// layoutSections computes each section's byte offset (8-aligned, right
// after the header+catalog) and the total catalog length.
func layoutSections(sections []Section) (catalogLen int, placed []struct{ offset int64 }) {
	catalogLen = 2 + len(sections)*entrySize
	placed = make([]struct{ offset int64 }, len(sections))
	pos := int64(headerSize + catalogLen)
	for i, s := range sections {
		pos = align8(pos)
		placed[i].offset = pos
		pos += int64(len(s.Data))
	}
	return catalogLen, placed
}

// writeSections writes the header+catalog followed by each section's
// padding and data, per the layout computed by layoutSections.
func writeSections(f *os.File, sections []Section, placed []struct{ offset int64 }, catalogLen int) error {
	if err := writeContainerHead(f, sections, placed); err != nil {
		return err
	}

	cur := int64(headerSize + catalogLen)
	for i, s := range sections {
		if pad := placed[i].offset - cur; pad > 0 {
			if _, err := f.Write(make([]byte, pad)); err != nil {
				return fmt.Errorf("format: write padding: %w", err)
			}
			cur += pad
		}
		if _, err := f.Write(s.Data); err != nil {
			return fmt.Errorf("format: write section %s: %w", s.Name, err)
		}
		cur += int64(len(s.Data))
	}
	return nil
}

// finalizeChecksum computes the checksum over the data just written,
// patches it into the header, and syncs the file to disk.
func finalizeChecksum(f *os.File) error {
	sum, err := checksumFile(f, headerSize)
	if err != nil {
		return fmt.Errorf("format: checksum: %w", err)
	}
	if _, err := f.Seek(8, io.SeekStart); err != nil {
		return err
	}
	var chk [8]byte
	binary.LittleEndian.PutUint64(chk[:], sum)
	if _, err := f.Write(chk[:]); err != nil {
		return fmt.Errorf("format: write checksum: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("format: sync: %w", err)
	}
	return nil
}

// OpenContainer validates a GMOR file from its bytes (magic, version,
// checksum, catalog) and returns a container giving access to its
// sections.
func OpenContainer(data []byte) (*Container, error) {
	if len(data) < headerSize {
		return nil, ErrMalformedFile
	}
	if string(data[:4]) != magicHeader {
		return nil, wrap(ErrBadMagic, string(data[:4]))
	}
	if binary.LittleEndian.Uint32(data[4:8]) != Version {
		return nil, wrap(ErrUnsupportedVersion, fmt.Sprint(binary.LittleEndian.Uint32(data[4:8])))
	}

	want := binary.LittleEndian.Uint64(data[8:16])
	got := xxh3.Hash(data[16:])
	if got != want {
		return nil, wrap(ErrBadChecksum, fmt.Sprintf("got %x, want %x", got, want))
	}

	count := int(binary.LittleEndian.Uint16(data[16:18]))
	p := 18
	entries := make([]Entry, 0, count)
	for i := 0; i < count; i++ {
		if p+entrySize > len(data) {
			return nil, ErrMalformedFile
		}
		name := strings.TrimRight(string(data[p:p+nameSize]), "\x00")
		p += nameSize
		offset := int64(binary.LittleEndian.Uint64(data[p:]))
		p += 8
		size := int64(binary.LittleEndian.Uint64(data[p:]))
		p += 8
		flags := data[p]
		p++

		if offset < 0 || size < 0 || offset+size > int64(len(data)) {
			return nil, wrap(ErrMalformedFile, name)
		}
		entries = append(entries, Entry{Name: name, Offset: offset, Size: size, Flags: flags})
	}
	return &Container{data: data, entries: entries}, nil
}

func validateSections(sections []Section) error {
	if len(sections) > math.MaxUint16 {
		return errors.New("format: too many sections")
	}
	seen := make(map[string]bool, len(sections))
	for _, s := range sections {
		if s.Name == "" || len(s.Name) > nameSize {
			return fmt.Errorf("format: section name %q longer than %d bytes", s.Name, nameSize)
		}
		if s.Flags&^compressionMask != 0 {
			return fmt.Errorf("format: section %s has reserved flag bits set: %#x", s.Name, s.Flags)
		}
		if compression(s.Flags) > maxKnownCompression {
			return fmt.Errorf("format: section %s has unknown compression algorithm %d", s.Name, compression(s.Flags))
		}
		if seen[s.Name] {
			return fmt.Errorf("format: duplicate section %q", s.Name)
		}
		seen[s.Name] = true
	}
	return nil
}

func writeContainerHead(f *os.File, sections []Section, placed []struct{ offset int64 }) error {
	var buf bytes.Buffer
	buf.WriteString(magicHeader)
	var v [4]byte
	binary.LittleEndian.PutUint32(v[:], Version)
	buf.Write(v[:])
	buf.Write(make([]byte, 8)) // checksum placeholder
	var count [2]byte
	binary.LittleEndian.PutUint16(count[:], uint16(len(sections)))
	buf.Write(count[:])
	for i, s := range sections {
		var name [nameSize]byte
		copy(name[:], s.Name)
		buf.Write(name[:])
		var off [8]byte
		binary.LittleEndian.PutUint64(off[:], uint64(placed[i].offset))
		buf.Write(off[:])
		var sz [8]byte
		binary.LittleEndian.PutUint64(sz[:], uint64(len(s.Data)))
		buf.Write(sz[:])
		buf.WriteByte(s.Flags)
	}
	_, err := f.Write(buf.Bytes())
	return err
}

func checksumFile(f *os.File, from int64) (uint64, error) {
	if _, err := f.Seek(from, io.SeekStart); err != nil {
		return 0, err
	}
	h := xxh3.New()
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

func align8(n int64) int64 {
	if n%8 == 0 {
		return n
	}
	return n + 8 - n%8
}

func wrap(err error, msg string) error {
	return fmt.Errorf("%w: %s", err, msg)
}

// EncodeMeta serializes the language and substitution policy (CharPolicy).
func EncodeMeta(language string, policy *CharPolicy) []byte {
	var buf bytes.Buffer
	writeU16String(&buf, language)
	if policy == nil || len(policy.Substitutions) == 0 {
		buf.WriteByte(0)
		return buf.Bytes()
	}
	if len(policy.Substitutions) > 255 {
		panic("internal: too many char substitutions")
	}
	buf.WriteByte(byte(len(policy.Substitutions)))
	for _, s := range policy.Substitutions {
		var e [8]byte
		binary.LittleEndian.PutUint32(e[:4], uint32(s.From))
		binary.LittleEndian.PutUint32(e[4:], uint32(s.To))
		buf.Write(e[:])
	}
	return buf.Bytes()
}

// DecodeMeta reads {language, CharPolicy} written by EncodeMeta.
func DecodeMeta(data []byte) (string, *CharPolicy, error) {
	lang, p, err := readU16String(data)
	if err != nil {
		return "", nil, err
	}
	if p >= len(data) {
		return "", nil, ErrMalformedFile
	}
	n := int(data[p])
	p++
	if p+n*8 > len(data) {
		return "", nil, ErrMalformedFile
	}
	subs := make([]Substitution, 0, n)
	for i := 0; i < n; i++ {
		subs = append(subs, Substitution{
			From: rune(binary.LittleEndian.Uint32(data[p:])),
			To:   rune(binary.LittleEndian.Uint32(data[p+4:])),
		})
		p += 8
	}
	return lang, NewCharPolicy(subs...), nil
}

// EncodeStrings serializes a []string as uvarint-length-prefixed strings.
func EncodeStrings(ar []string) []byte {
	var buf bytes.Buffer
	var tmp [binary.MaxVarintLen64]byte
	for _, s := range ar {
		n := binary.PutUvarint(tmp[:], uint64(len(s)))
		buf.Write(tmp[:n])
		buf.WriteString(s)
	}
	return buf.Bytes()
}

// DecodeStrings reads strings written by EncodeStrings.
func DecodeStrings(data []byte) ([]string, error) {
	var out []string
	p := 0
	for p < len(data) {
		n, nr := binary.Uvarint(data[p:])
		if nr <= 0 {
			return nil, ErrMalformedFile
		}
		if p+nr > len(data) {
			return nil, ErrMalformedFile
		}
		p += nr
		if n > uint64(len(data)-p) {
			return nil, ErrMalformedFile
		}
		out = append(out, string(data[p:p+int(n)]))
		p += int(n)
	}
	return out, nil
}

// EncodeTagSet serializes a TagSet as JSON {name, tags}.
func EncodeTagSet(ts *TagSet) []byte {
	v := struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{Name: ts.Name, Tags: ts.Tags}
	data, _ := json.Marshal(v)
	return data
}

// DecodeTagSet reconstructs a TagSet from EncodeTagSet's JSON.
func DecodeTagSet(data []byte) (*TagSet, error) {
	var v struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, wrap(ErrMalformedFile, "tagset")
	}
	ts := NewTagSet(v.Name)
	for _, name := range v.Tags {
		if _, err := ts.Add(name); err != nil {
			return nil, wrap(ErrMalformedFile, "tagset: "+err.Error())
		}
	}
	return ts, nil
}

// EncodeParadigms serializes paradigms: u32 count; per paradigm, u32 len
// + len×u16 of data.
func EncodeParadigms(ps []Paradigm) []byte {
	var buf bytes.Buffer
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(len(ps)))
	buf.Write(b[:])
	for _, par := range ps {
		data := par.Data()
		binary.LittleEndian.PutUint32(b[:], uint32(len(data)))
		buf.Write(b[:])
		for _, v := range data {
			binary.LittleEndian.PutUint16(b[:2], v)
			buf.Write(b[:2])
		}
	}
	return buf.Bytes()
}

// DecodeParadigms reads paradigms written by EncodeParadigms.
func DecodeParadigms(data []byte) ([]Paradigm, error) {
	if len(data) < 4 {
		return nil, ErrMalformedFile
	}
	count := int(binary.LittleEndian.Uint32(data[:4]))
	if count > 1<<20 {
		return nil, ErrMalformedFile
	}
	p := 4
	out := make([]Paradigm, 0, count)
	for i := 0; i < count; i++ {
		if p+4 > len(data) {
			return nil, ErrMalformedFile
		}
		n := int(binary.LittleEndian.Uint32(data[p:]))
		p += 4
		if n < 0 || p+n*2 > len(data) {
			return nil, ErrMalformedFile
		}
		raw := make([]uint16, n)
		j := p
		for k := range raw {
			raw[k] = binary.LittleEndian.Uint16(data[j:])
			j += 2
		}
		p += n * 2
		par, err := NewParadigmFromData(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, par)
	}
	return out, nil
}

// denseAlphabetKind is EncodeAlphabet's kind byte for *DenseAlphabet — the
// only Alphabet implementation with an on-disk representation today.
const denseAlphabetKind = 0

// EncodeAlphabet serializes a into the "alphabet" section's bytes: 1 byte
// kind (denseAlphabetKind), 1 byte width, then every rune in code order
// (code 2 first) written as a UTF-8 string. Returns an error for any
// Alphabet implementation other than *DenseAlphabet (IdentityAlphabet
// needs no section at all — callers write "alphabet" only when
// Dictionary.Alphabet is non-nil, mirroring the "probability" section's
// conditional pattern in save.go).
func EncodeAlphabet(a Alphabet) ([]byte, error) {
	da, ok := a.(*DenseAlphabet)
	if !ok {
		return nil, fmt.Errorf("internal: EncodeAlphabet: unsupported Alphabet type %T", a)
	}
	var buf bytes.Buffer
	buf.WriteByte(denseAlphabetKind)
	buf.WriteByte(byte(da.width))
	buf.WriteString(string(da.runeOf))
	return buf.Bytes(), nil
}

// DecodeAlphabet reconstructs the Alphabet written by EncodeAlphabet.
func DecodeAlphabet(data []byte) (Alphabet, error) {
	if len(data) < 2 {
		return nil, wrap(ErrMalformedFile, "alphabet")
	}
	kind := data[0]
	if kind != denseAlphabetKind {
		return nil, wrap(ErrMalformedFile, fmt.Sprintf("alphabet: unknown kind %d", kind))
	}
	width := int(data[1])
	runes := []rune(string(data[2:]))
	a, err := newDenseAlphabetFromRunes(width, runes)
	if err != nil {
		return nil, wrap(ErrMalformedFile, "alphabet: "+err.Error())
	}
	return a, nil
}

func writeU16String(buf *bytes.Buffer, s string) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(len(s)))
	buf.Write(b[:])
	buf.WriteString(s)
}

func readU16String(data []byte) (string, int, error) {
	if len(data) < 2 {
		return "", 0, ErrMalformedFile
	}
	n := int(binary.LittleEndian.Uint16(data[:2]))
	if n < 0 || 2+n > len(data) {
		return "", 0, ErrMalformedFile
	}
	return string(data[2 : 2+n]), 2 + n, nil
}
