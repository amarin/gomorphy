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

// Единый дисковый формат словаря GMOR.
//
//	┌──────────────────────────────────┐
//	│  Заголовок                       │
//	│  magic  "GMOR"  4 байта          │
//	│  version    u32                  │
//	│  checksum   xxh3-64  8 байт      │
//	├──────────────────────────────────┤
//	│  Каталог секций                  │
//	│  count      u16                  │
//	│  entries:                        │
//	│    name     [16]byte             │
//	│    offset   u64                  │
//	│    size     u64                  │
//	│    flags    u8                   │
//	├──────────────────────────────────┤
//	│  Секции данных                   │
//	└──────────────────────────────────┘
//
// Checksum покрывает всё, что после поля checksum (каталог + секции).
// Смещения секций выровнены по 8 байтам, чтобы словарь words.dawg можно
// было алиасить из mmap без копирования (offset+4 кратен 4).
//
// flags: младшие 4 бита — id алгоритма сжатия секции (см. Compression*,
// 0 — без сжатия), старшие 4 бита зарезервированы под независимые от
// сжатия флаги будущих версий. Алгоритм записывается явно (а не
// угадывается по сигнатуре данных секции): секция — это произвольный
// blob без самоописывающегося заголовка, и явный id даёт точную,
// однозначную диагностику для файла из будущей версии с неизвестным
// читателю алгоритмом («unsupported compression algorithm 3, upgrade
// required») вместо попытки угадать формат по первым байтам. Сжатие
// выбирается на уровне секции (не файла целиком): words.dawg остаётся
// несжатой, чтобы её можно было алиасить из mmap без копирования; выбор
// алгоритма для остальных секций — за реализацией самого сжатия (см.
// docs/todo.md, "Этап 17").
const (
	magicHeader = "GMOR"
	headerSize  = 16 // magic(4) + version(4) + checksum(8)
	nameSize    = 16
	entrySize   = nameSize + 8 + 8 + 1 // 33 байта на запись каталога
)

const (
	// Version — версия формата GMOR.
	Version uint32 = 1
)

// Алгоритмы сжатия секции (младшие 4 бита Entry.Flags/Section.Flags).
// Добавление нового алгоритма — это добавление константы и ветки в
// Container.Section/validateSections, без изменения байтового layout
// каталога и без версионирования: сжатие ни разу не было записано ни в
// одном выпущенном файле, поэтому этот нибл свободен для полного контроля
// именно сейчас, до релиза 1.0.
const (
	CompressionNone uint8 = 0
	CompressionZstd uint8 = 1

	// compressionMask выделяет id алгоритма из флагов записи каталога.
	compressionMask uint8 = 0x0F
	// maxKnownCompression — верхняя граница известных id алгоритмов;
	// расширять по мере добавления новых констант Compression*.
	maxKnownCompression = CompressionZstd
)

// compression возвращает id алгоритма сжатия секции по флагам записи каталога.
func compression(flags uint8) uint8 { return flags & compressionMask }

// Ошибки формата.
var (
	ErrBadMagic               = errors.New("format: bad magic")
	ErrUnsupportedVersion     = errors.New("format: unsupported version")
	ErrBadChecksum            = errors.New("format: checksum mismatch")
	ErrMalformedFile          = errors.New("format: malformed file")
	ErrUnknownSection         = errors.New("format: unknown section")
	ErrUnsupportedCompression = errors.New("format: unsupported compression algorithm")
)

// Section — секция для записи: данные и флаги.
type Section struct {
	Name  string
	Data  []byte
	Flags uint8
}

// Entry — запись каталога секций.
type Entry struct {
	Name   string
	Offset int64
	Size   int64
	Flags  uint8
}

// Container — открытый формат ф-файла. Данные секций алиасят исходный
// срез data (для mmap-региона это zero-copy).
type Container struct {
	data    []byte
	entries []Entry
}

// Entries возвращает копию каталога секций.
func (c *Container) Entries() []Entry {
	out := make([]Entry, len(c.entries))
	copy(out, c.entries)
	return out
}

// Section возвращает данные секции по имени (срез алиасит файл) и её флаги.
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

// SaveContainer записывает файл GMOR: заголовок + каталог + секции.
// Смещения секций выравниваются по 8 байтам.
func SaveContainer(path string, sections []Section) error {
	if err := validateSections(sections); err != nil {
		return err
	}

	catalogLen := 2 + len(sections)*entrySize
	placed := make([]struct{ offset int64 }, len(sections))
	pos := int64(headerSize + catalogLen)
	for i, s := range sections {
		pos = align8(pos)
		placed[i].offset = pos
		pos += int64(len(s.Data))
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("format: mkdir %s: %w", dir, err)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("format: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

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
	return f.Sync()
}

// OpenContainer валидирует файл GMOR по байтам (magic, version, checksum,
// каталог) и возвращает контейнер с доступом к секциям.
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

// EncodeMeta сериализует язык и политику подстановок (CharPolicy).
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

// DecodeMeta читает {language, CharPolicy} из EncodeMeta.
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

// EncodeStrings сериализует []string как uvarint-length-prefixed строки.
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

// DecodeStrings читает строки, записанные EncodeStrings.
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

// EncodeTagSet сериализует TagSet в JSON {name, tags}.
func EncodeTagSet(ts *TagSet) []byte {
	v := struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{Name: ts.Name, Tags: ts.Tags}
	data, _ := json.Marshal(v)
	return data
}

// DecodeTagSet восстанавливает TagSet из JSON EncodeTagSet.
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
		ts.Add(name)
	}
	return ts, nil
}

// EncodeParadigms сериализует парадигмы: u32 count; на парадигму
// u32 len + len×u16 данных.
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

// DecodeParadigms читает парадигмы, записанные EncodeParadigms.
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
