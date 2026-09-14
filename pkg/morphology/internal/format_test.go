package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeContainer(t *testing.T, sections []Section) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dict.dat")
	require.NoError(t, SaveContainer(path, sections))
	return path
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestSaveContainerRoundtrip(t *testing.T) {
	sections := []Section{
		{Name: "meta", Data: []byte("ru")},
		{Name: "suffixes", Data: []byte("a\x00b\x00")},
		{Name: "empty", Data: nil},
		{Name: "words.dawg", Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
	}
	path := writeContainer(t, sections)

	cont, err := OpenContainer(readFile(t, path))
	require.NoError(t, err)

	for _, want := range sections {
		data, flags, err := cont.Section(want.Name)
		require.NoError(t, err)
		assert.Equal(t, want.Flags, flags)
		assert.Equal(t, want.Data, data)
	}

	_, _, err = cont.Section("nope")
	assert.ErrorIs(t, err, ErrUnknownSection)
}

func TestSaveContainerAlignsSections(t *testing.T) {
	cont, err := OpenContainer(readFile(t, writeContainer(t, []Section{
		{Name: "a", Data: []byte("123")}, // длина 3
		{Name: "b", Data: []byte("xy")},  // длина 2
		{Name: "words.dawg", Data: []byte{9, 9, 9, 9, 9, 9, 9, 9}},
	})))
	require.NoError(t, err)

	for _, e := range cont.Entries() {
		assert.Zero(t, e.Offset%8, "section %s offset %d not 8-aligned", e.Name, e.Offset)
	}
	words, _, err := cont.Section("words.dawg")
	require.NoError(t, err)
	assert.Equal(t, []byte{9, 9, 9, 9, 9, 9, 9, 9}, words)
}

func TestOpenContainerBadMagic(t *testing.T) {
	path := writeContainer(t, []Section{{Name: "meta", Data: []byte{1}}})
	data := readFile(t, path)
	data[0] = 'X'

	_, err := OpenContainer(data)
	assert.ErrorIs(t, err, ErrBadMagic)
}

func TestOpenContainerUnsupportedVersion(t *testing.T) {
	path := writeContainer(t, []Section{{Name: "meta", Data: []byte{1}}})
	data := readFile(t, path)
	data[4] = 0x63
	data[5] = 0x00

	_, err := OpenContainer(data)
	assert.ErrorIs(t, err, ErrUnsupportedVersion)
}

func TestOpenContainerBadChecksum(t *testing.T) {
	path := writeContainer(t, []Section{{Name: "meta", Data: []byte("hello")}})
	data := readFile(t, path)
	data[len(data)-1] ^= 0xff

	_, err := OpenContainer(data)
	assert.ErrorIs(t, err, ErrBadChecksum)
}

func TestOpenContainerMalformed(t *testing.T) {
	for _, name := range []string{"short", "truncated-catalog", "truncated-section"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.dat")
			var data []byte
			switch name {
			case "short":
				data = []byte("GMOR")
			case "truncated-catalog":
				require.NoError(t, SaveContainer(path, []Section{{Name: "meta", Data: []byte("x")}}))
				data = readFile(t, path)[:20]
			case "truncated-section":
				require.NoError(t, SaveContainer(path, []Section{{Name: "meta", Data: []byte("hello")}}))
				data = readFile(t, path)[:len(readFile(t, path))-4]
			}
			_, err := OpenContainer(data)
			require.Error(t, err)
		})
	}
}

func TestOpenContainerCompressedUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.dat")
	require.NoError(t, SaveContainer(path, []Section{{Name: "tagset", Data: []byte{2}, Flags: CompressionZstd}}))

	cont, err := OpenContainer(readFile(t, path))
	require.NoError(t, err)
	_, _, err = cont.Section("tagset")
	assert.ErrorIs(t, err, ErrUnsupportedCompression)
}

func TestSaveContainerRejectsReservedFlagBits(t *testing.T) {
	err := SaveContainer(filepath.Join(t.TempDir(), "c.dat"), []Section{
		{Name: "tagset", Data: []byte{2}, Flags: 1 << 4},
	})
	require.Error(t, err)
}

func TestSaveContainerRejectsUnknownCompressionAlgorithm(t *testing.T) {
	err := SaveContainer(filepath.Join(t.TempDir(), "c.dat"), []Section{
		{Name: "tagset", Data: []byte{2}, Flags: maxKnownCompression + 1},
	})
	require.Error(t, err)
}

func TestSaveContainerBadName(t *testing.T) {
	require.Error(t, SaveContainer(filepath.Join(t.TempDir(), "x.dat"), []Section{
		{Name: "this-name-is-way-too-long-16", Data: []byte{1}},
	}))
}

func TestSaveContainerTooManySections(t *testing.T) {
	sections := make([]Section, 65536)
	for i := range sections {
		sections[i] = Section{Name: "meta",
			Data: []byte{1}}
	}
	require.Error(t, SaveContainer(filepath.Join(t.TempDir(), "x.dat"), sections))
}

func TestMetaRoundtrip(t *testing.T) {
	policy := RussianCharPolicy()
	data := EncodeMeta("ru", policy)

	lang, got, err := DecodeMeta(data)
	require.NoError(t, err)
	assert.Equal(t, "ru", lang)
	require.NotNil(t, got)
	assert.Equal(t, policy.Substitutions, got.Substitutions)
}

func TestMetaNilPolicy(t *testing.T) {
	lang, policy, err := DecodeMeta(EncodeMeta("en", nil))
	require.NoError(t, err)
	assert.Equal(t, "en", lang)
	assert.Empty(t, policy.Substitutions)
}

func TestMetaTruncated(t *testing.T) {
	_, _, err := DecodeMeta([]byte{0xff})
	require.Error(t, err)
}

func TestStringsRoundtrip(t *testing.T) {
	in := []string{"", "а", "по", "наи", "международный", "ёжик"}
	data := EncodeStrings(in)

	got, err := DecodeStrings(data)
	require.NoError(t, err)
	assert.Equal(t, in, got)
}

func TestStringsEmpty(t *testing.T) {
	got, err := DecodeStrings(nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestStringsTruncated(t *testing.T) {
	_, err := DecodeStrings([]byte{1})
	require.Error(t, err)
	_, err = DecodeStrings([]byte{3, 0, 0})
	require.Error(t, err)
}

func TestTagSetRoundtrip(t *testing.T) {
	ts := NewTagSet("opencorpora-int")
	ts.Add("NOUN,anim,masc,sing,nomn")
	ts.Add("VERB,impf,trans")
	id, _ := ts.ID("VERB,impf,trans")

	got, err := DecodeTagSet(EncodeTagSet(ts))
	require.NoError(t, err)
	assert.Equal(t, "opencorpora-int", got.Name)
	assert.Equal(t, ts.Tags, got.Tags)
	gid, ok := got.ID("VERB,impf,trans")
	require.True(t, ok)
	assert.Equal(t, id, gid)
}

func TestTagSetBadJSON(t *testing.T) {
	_, err := DecodeTagSet([]byte("nope"))
	require.Error(t, err)
}

func TestParadigmsRoundtrip(t *testing.T) {
	in := []Paradigm{NewParadigm([]uint16{0, 1}, []uint16{0, 1}, []uint16{0, 0})}
	data := EncodeParadigms(in)

	got, err := DecodeParadigms(data)
	require.NoError(t, err)
	require.Len(t, got, len(in))
	assert.Equal(t, in[0].data, got[0].data)
}

func TestParadigmsEmpty(t *testing.T) {
	got, err := DecodeParadigms(EncodeParadigms(nil))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDecodeParadigmsRejectsBadLength(t *testing.T) {
	bad := EncodeParadigms(nil)
	bad[0] = 1
	_, err := DecodeParadigms(bad)
	require.Error(t, err)
}
