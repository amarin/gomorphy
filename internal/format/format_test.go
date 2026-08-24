package format_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"

	"github.com/amarin/gomorphy/internal/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memFile struct {
	buf []byte
}

func (m *memFile) WriteAt(p []byte, off int64) (int, error) {
	end := int(off) + len(p)
	if end > len(m.buf) {
		m.buf = append(m.buf[:off], make([]byte, end-len(m.buf))...)
	}

	copy(m.buf[off:], p)

	return len(p), nil
}

func (m *memFile) ReadAt(p []byte, off int64) (int, error) {
	if off > int64(len(m.buf)) {
		return 0, io.EOF
	}

	n := copy(p, m.buf[off:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}

func newMemFile() *memFile { return &memFile{} }

func buildFile(t *testing.T, version uint32, sections map[string][]byte) *memFile {
	t.Helper()

	f := newMemFile()
	w := format.NewWriter(f)
	require.NoError(t, w.WriteHeader(version))

	for name, data := range sections {
		require.NoError(t, w.WriteSection(name, data))
	}

	require.NoError(t, w.Finalize())

	return f
}

func TestAppendDeltaParseDelta_Roundtrip(t *testing.T) {
	maxU64 := uint64(math.MaxUint64)

	cases := [][]uint64{
		{},
		{0},
		{1},
		{0, 0, 0, 0},
		{42, 42, 42},
		{0, 1, 2, 3, 4, 5},
		{maxU64},
		{0, maxU64},
		{maxU64, 0},
		{7, 3, 9, 1, math.MaxInt64, 0, math.MaxUint32 + 1},
		{1000000, 999999, 1000001, 5, math.MaxUint64 - 5, math.MaxUint64},
	}

	for i, vals := range cases {
		encoded := format.AppendDelta(nil, vals)
		decoded, err := format.ParseDelta(encoded)

		require.NoError(t, err, "case %d", i)
		assert.Equal(t, vals, decoded, "case %d", i)
	}
}

func TestAppendDelta32ParseDelta32_Roundtrip(t *testing.T) {
	cases := [][]uint32{
		{},
		{0},
		{math.MaxUint32},
		{10, 20, 30},
		{math.MaxUint32, 0, 1, 1, 2, 3},
		{500000000, 499999999, 0},
	}

	for i, vals := range cases {
		encoded := format.AppendDelta32(nil, vals)
		decoded, err := format.ParseDelta32(encoded)

		require.NoError(t, err, "case %d", i)
		assert.Equal(t, vals, decoded, "case %d", i)
	}
}

func TestAppendDelta_EncodesCompact(t *testing.T) {
	vals := make([]uint64, 1000)
	for i := range vals {
		vals[i] = uint64(i) * 8
	}

	encoded := format.AppendDelta(nil, vals)
	assert.Less(t, len(encoded), 2200, "ascending deltas should take ~1-2 bytes each")
}

func TestParseDelta_Malformed(t *testing.T) {
	_, err := format.ParseDelta([]byte{0xff, 0xff, 0xff})
	assert.ErrorIs(t, err, format.ErrMalformedDelta)

	valid := format.AppendDelta32(nil, []uint32{1, 2})
	_, err = format.ParseDelta(valid[:len(valid)-1])
	assert.ErrorIs(t, err, format.ErrMalformedDelta)
}

func TestParseDelta32_Overflow(t *testing.T) {
	big := format.AppendDelta(nil, []uint64{1 << 33})

	_, err := format.ParseDelta32(big)
	assert.ErrorIs(t, err, format.ErrMalformedDelta)
}

func TestOpen_SectionsRoundtrip(t *testing.T) {
	f := buildFile(t, format.Version, map[string][]byte{
		"texts":    bytes.Repeat([]byte("abracadabra"), 100),
		"offsets":  format.AppendDelta32(nil, []uint32{0, 10, 20, 100500}),
		"empty":    {},
		"unicode":  []byte("ёжик в тумане"),
	})

	r, err := format.Open(f, int64(len(f.buf)))
	require.NoError(t, err)
	assert.Equal(t, format.Version, r.Version())

	got, err := r.SectionBytes("texts")
	require.NoError(t, err)
	assert.Len(t, got, 1100)

	offsets, err := format.ParseDelta32(got2(t, r, "offsets"))
	require.NoError(t, err)
	assert.Equal(t, []uint32{0, 10, 20, 100500}, offsets)

	empty, err := r.SectionBytes("empty")
	require.NoError(t, err)
	assert.Empty(t, empty)

	uni, err := r.SectionBytes("unicode")
	require.NoError(t, err)
	assert.Equal(t, []byte("ёжик в тумане"), uni)

	names := map[string]bool{}
	for _, e := range r.Entries() {
		names[e.Name] = true

		if e.Name == "empty" {
			assert.Zero(t, e.Size)
		} else {
			assert.Positive(t, e.Size)
		}
	}

	assert.True(t, names["texts"])
	assert.True(t, names["offsets"])
}

func got2(t *testing.T, r *format.Reader, name string) []byte {
	t.Helper()

	data, err := r.SectionBytes(name)
	require.NoError(t, err)

	return data
}

func TestOpen_UnknownSection(t *testing.T) {
	f := buildFile(t, format.Version, map[string][]byte{"a": {1}})

	r, err := format.Open(f, int64(len(f.buf)))
	require.NoError(t, err)

	_, err = r.SectionBytes("missing")
	assert.ErrorIs(t, err, format.ErrUnknownSection)
}

func TestOpen_BadMagic(t *testing.T) {
	f := buildFile(t, format.Version, map[string][]byte{"a": {1}})
	f.buf[0] = 'X'

	_, err := format.Open(f, int64(len(f.buf)))
	assert.ErrorIs(t, err, format.ErrBadMagic)
}

func TestOpen_UnsupportedVersion(t *testing.T) {
	f := buildFile(t, format.Version+1, map[string][]byte{"a": {1}})

	_, err := format.Open(f, int64(len(f.buf)))
	assert.ErrorIs(t, err, format.ErrUnsupportedVer)
}

func TestOpen_BadChecksum(t *testing.T) {
	f := buildFile(t, format.Version, map[string][]byte{"a": bytes.Repeat([]byte{7}, 128)})
	f.buf[40] ^= 0xff

	_, err := format.Open(f, int64(len(f.buf)))
	assert.ErrorIs(t, err, format.ErrBadChecksum)
}

func TestOpen_TruncatedTrailer(t *testing.T) {
	f := buildFile(t, format.Version, map[string][]byte{"a": {1, 2, 3}})

	_, err := format.Open(f, int64(len(f.buf))-1)
	assert.Error(t, err)
}

func TestWriter_Errors(t *testing.T) {
	f := newMemFile()
	w := format.NewWriter(f)
	require.NoError(t, w.WriteHeader(format.Version))

	sw, err := w.Create("first")
	require.NoError(t, err)

	_, err = w.Create("second")
	assert.ErrorIs(t, err, format.ErrSectionNotClosed)

	_, err = sw.Write([]byte("data"))
	require.NoError(t, err)
	require.NoError(t, sw.Close())

	_, err = sw.Write([]byte("more"))
	assert.ErrorIs(t, err, format.ErrSectionClosed)

	_, err = w.Create("first")
	assert.ErrorIs(t, err, format.ErrDuplicateSection)

	_, err = w.Create("")
	assert.ErrorIs(t, err, format.ErrNameTooLong)

	long := string(bytes.Repeat([]byte("x"), 256))
	_, err = w.Create(long)
	assert.ErrorIs(t, err, format.ErrNameTooLong)

	require.NoError(t, w.Finalize())
	assert.ErrorIs(t, w.Finalize(), format.ErrAlreadyFinalized)

	_, err = w.Create("after")
	assert.ErrorIs(t, err, format.ErrAlreadyFinalized)
}

func TestWriter_StreamingSection(t *testing.T) {
	f := newMemFile()
	w := format.NewWriter(f)
	require.NoError(t, w.WriteHeader(format.Version))

	sw, err := w.Create("stream")
	require.NoError(t, err)

	var want []byte
	for i := range 10 {
		part := bytes.Repeat([]byte{byte('a' + i)}, 1000+i)
		want = append(want, part...)

		_, err := sw.Write(part)
		require.NoError(t, err)
	}

	require.NoError(t, sw.Close())
	require.NoError(t, w.Finalize())

	r, err := format.Open(f, int64(len(f.buf)))
	require.NoError(t, err)

	got, err := r.SectionBytes("stream")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestWriter_EmptySections(t *testing.T) {
	f := buildFile(t, format.Version, map[string][]byte{"e1": {}, "e2": {}})

	r, err := format.Open(f, int64(len(f.buf)))
	require.NoError(t, err)

	for _, name := range []string{"e1", "e2"} {
		data, err := r.SectionBytes(name)
		require.NoError(t, err)
		assert.Empty(t, data)
	}
}

func TestHeaderLayout(t *testing.T) {
	f := buildFile(t, format.Version, map[string][]byte{"a": {9}})
	assert.Equal(t, "GMRF", string(f.buf[:4]))
	assert.Equal(t, format.Version, binary.LittleEndian.Uint32(f.buf[4:8]))
}
