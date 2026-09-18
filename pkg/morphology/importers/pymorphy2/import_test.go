package pymorphy2_test

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// payloadSeparator is the protocol separator between a word and its
// payload in words.dawg (0x01).
const payloadSeparator = "\x01"

func b64(p []byte) string {
	return base64.StdEncoding.EncodeToString(p)
}

func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeParadigms(t *testing.T, dir string, paradigms [][]uint16) {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint16(len(paradigms))))
	for _, p := range paradigms {
		require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint16(len(p))))
		for _, v := range p {
			require.NoError(t, binary.Write(&buf, binary.LittleEndian, v))
		}
	}
	writeFile(t, dir, "paradigms.array", buf.Bytes())
}

// makeFixtureDir assembles a minimal pymorphy2 directory in t.TempDir().
func makeFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeParadigms(t, dir, [][]uint16{
		{10, 20, 0, 1, 0, 0}, // 2 forms: suffixes[10,20], tags[0,1], prefixes[0,0]
	})

	writeFile(t, dir, "suffixes.json", []byte(`["","кот","кота","x"]`))
	writeFile(t, dir, "paradigm-prefixes.json", []byte(`["","по","наи"]`))
	writeFile(t, dir, "gramtab-opencorpora-int.json", []byte(`["NOUN,anim,masc,sing,nomn","NOUN,anim,masc,sing,gent"]`))

	words, guide := testdawg.Build(map[string]uint32{
		"кот" + payloadSeparator + b64([]byte{0, 0, 0, 0}):  0,
		"кота" + payloadSeparator + b64([]byte{0, 0, 0, 1}): 0,
	})
	writeFile(t, dir, "words.dawg", testdawg.Marshal(words, guide))

	return dir
}

func TestImportFromDir(t *testing.T) {
	dir := makeFixtureDir(t)

	d, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)
	require.NotNil(t, d)

	assert.Equal(t, "ru", d.Language)
	require.NotNil(t, d.CharPolicy)

	require.NotNil(t, d.TagSet)
	id, ok := d.TagSet.ID("NOUN,anim,masc,sing,gent")
	require.True(t, ok)
	assert.Equal(t, uint16(1), id)

	require.Len(t, d.Suffixes, 1)
	assert.Equal(t, []string{"", "кот", "кота", "x"}, d.Suffixes[0])
	assert.Equal(t, []string{"", "по", "наи"}, d.Prefixes)

	require.Len(t, d.Paradigms, 1)
	require.Len(t, d.Paradigms[0], 1)
	p := d.Paradigms[0][0]
	assert.Equal(t, 2, p.Len())
	assert.Equal(t, uint16(10), p.Suffix(0))
	assert.Equal(t, uint16(1), p.Tag(1))
	assert.Equal(t, uint16(0), p.Prefix(1))

	require.Len(t, d.Words, 1)
	require.NotNil(t, d.Words[0])
	items := d.Words[0].SimilarItems("кот", d.CharPolicy, nil)
	require.Len(t, items, 1)
	assert.Equal(t, "кот", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{0, 0, 0, 0}, items[0].Values[0])

	assert.Empty(t, d.Prediction)
	assert.Nil(t, d.Probability)

	require.NotNil(t, d.Info)
	assert.Equal(t, "pymorphy2", d.Info.Source)
	assert.Empty(t, d.Info.SourceVersion, "fixture has no meta.json")
}

func TestImportFromDirSourceVersion(t *testing.T) {
	dir := makeFixtureDir(t)
	writeFile(t, dir, "meta.json", []byte(`[
		["language_code", "ru"],
		["format_version", "2.4"],
		["source", "opencorpora.org"],
		["source_version", "0.92"],
		["source_revision", "417127"],
		["source_lexemes_count", 391764]
	]`))

	d, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)
	require.NotNil(t, d.Info)
	assert.Equal(t, "0.92/417127", d.Info.SourceVersion)
}

func TestImportFromDirDefaultPrefixes(t *testing.T) {
	dir := makeFixtureDir(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "paradigm-prefixes.json")))

	d, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"", "по", "наи"}, d.Prefixes)
}

func TestImportFromDirOptionalFiles(t *testing.T) {
	dir := makeFixtureDir(t)

	for i := 0; i < 3; i++ {
		words, guide := testdawg.Build(map[string]uint32{fmt.Sprintf("все:NOUN,sing,nomn,%d", i): uint32(100 + i)})
		writeFile(t, dir, fmt.Sprintf("prediction-suffixes-%d.dawg", i), testdawg.Marshal(words, guide))
	}
	pwords, pguide := testdawg.Build(map[string]uint32{"все:NOUN,plur,nomn": 555})
	writeFile(t, dir, "p_t_given_w.intdawg", testdawg.Marshal(pwords, pguide))

	d, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)

	require.Len(t, d.Prediction, 3)
	require.NotNil(t, d.Probability)
	assert.Equal(t, uint32(555), d.Probability.Find("все:NOUN,plur,nomn"))
}

func TestImportFromDirMissingMandatory(t *testing.T) {
	dir := makeFixtureDir(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "words.dawg")))

	_, err := pymorphy2.ImportFromDir(dir)
	require.Error(t, err)
}

func TestImportFromDirBadParadigm(t *testing.T) {
	dir := makeFixtureDir(t)
	writeParadigms(t, dir, [][]uint16{{1, 2, 3, 4}})

	_, err := pymorphy2.ImportFromDir(dir)
	require.Error(t, err, "paradigm length must be divisible by 3")
}
