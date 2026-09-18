package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
		{Name: "a", Data: []byte("123")}, // length 3
		{Name: "b", Data: []byte("xy")},  // length 2
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

// TestSaveContainerNoLeftoverTempFile guards the atomic-write path: a
// successful save must leave only the final file behind, no ".tmp-*"
// sibling from the temp-file-then-rename sequence.
func TestSaveContainerNoLeftoverTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.dat")
	require.NoError(t, SaveContainer(path, []Section{{Name: "meta", Data: []byte("ru")}}))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "expected only the final file in %s, got %v", dir, entries)
	assert.Equal(t, "dict.dat", entries[0].Name())
}

// TestSaveContainerAtomicOnFailure guards against a crashed/failed write
// clobbering whatever was already at path: SaveContainer writes to a temp
// file and renames it into place only on success, so a failure (here: path
// is a non-empty directory, which os.Rename cannot replace) must leave the
// existing path untouched and not strand a temp file next to it.
func TestSaveContainerAtomicOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.dat")

	require.NoError(t, os.Mkdir(path, 0o755))
	sentinel := filepath.Join(path, "sentinel")
	require.NoError(t, os.WriteFile(sentinel, []byte("keep me"), 0o644))

	err := SaveContainer(path, []Section{{Name: "meta", Data: []byte("ru")}})
	require.Error(t, err)

	info, statErr := os.Stat(path)
	require.NoError(t, statErr)
	assert.True(t, info.IsDir(), "path must still be the original directory")
	assert.FileExists(t, sentinel, "pre-existing content under path must survive a failed save")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no leftover temp file expected in %s, got %v", dir, entries)
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
	_, err := ts.Add("NOUN,anim,masc,sing,nomn")
	require.NoError(t, err)
	_, err = ts.Add("VERB,impf,trans")
	require.NoError(t, err)
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

func TestBuildInfoRoundtrip(t *testing.T) {
	want := &BuildInfo{
		BuiltAt:        time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC),
		LibraryVersion: "0.1.0",
		Source:         "opencorpora",
		SourceVersion:  "0.92 rev417257",
		Author:         "test",
		Description:    "unit test fixture",
		SourceURL:      "https://example.com/dict.dat",
	}

	got, err := DecodeBuildInfo(EncodeBuildInfo(want))
	require.NoError(t, err)
	assert.Equal(t, want.BuiltAt.Unix(), got.BuiltAt.Unix())
	assert.Equal(t, want.LibraryVersion, got.LibraryVersion)
	assert.Equal(t, want.Source, got.Source)
	assert.Equal(t, want.SourceVersion, got.SourceVersion)
	assert.Equal(t, want.Author, got.Author)
	assert.Equal(t, want.Description, got.Description)
	assert.Equal(t, want.SourceURL, got.SourceURL)
}

func TestBuildInfoEncodeNil(t *testing.T) {
	got, err := DecodeBuildInfo(EncodeBuildInfo(nil))
	require.NoError(t, err)
	assert.Zero(t, got.BuiltAt)
	assert.Empty(t, got.LibraryVersion)
}

func TestBuildInfoBadJSON(t *testing.T) {
	_, err := DecodeBuildInfo([]byte("nope"))
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
