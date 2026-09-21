package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// importTSVFixture is a minimal 3-column (lemma-first) TSV corpus with two
// roots and one shared lemma over two wordforms.
const importTSVFixture = `кот	кот	NOUN,anim,masc,sing,nomn
кот	кота	NOUN,anim,masc,sing,gent
яблоко	яблоко	NOUN,inan,neut,sing,nomn
`

// writeFixtureTSV writes text to a fresh .tsv file in t.TempDir().
func writeFixtureTSV(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.tsv")
	require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
	return path
}

// openReadings opens a .dat and returns all readings of word.
func openReadings(t *testing.T, path, word string) []morphology.Reading {
	t.Helper()
	d, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, d.Close()) }()
	return d.Parse(word)
}

func TestImportTsvCommand_OK(t *testing.T) {
	tsvPath := writeFixtureTSV(t, importTSVFixture)
	out := filepath.Join(t.TempDir(), "out.dat")

	root := newTestRootCmd(newImportCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"import", "tsv", tsvPath, "-o", out})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "saved "+out)

	readings := openReadings(t, out, "кот")
	require.Len(t, readings, 1, "кот must have exactly one reading")
	assert.Equal(t, "кот", readings[0].Word)
	assert.Equal(t, "кот", readings[0].Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", readings[0].Tag)
	assert.NotEmpty(t, openReadings(t, out, "кота"), "second wordform of the same lemma must parse")
	assert.NotEmpty(t, openReadings(t, out, "яблоко"), "second root must parse")
}

func TestImportTsvCommand_SourceFlag(t *testing.T) {
	tsvPath := writeFixtureTSV(t, importTSVFixture)
	out := filepath.Join(t.TempDir(), "out.dat")

	root := newTestRootCmd(newImportCommand())
	root.SetArgs([]string{"import", "tsv", tsvPath, "-o", out, "--source", "custom"})
	require.NoError(t, root.Execute())

	d, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, d.Close()) }()
	require.NotNil(t, d.Info(), "imported dictionaries carry BuildInfo")
	assert.Equal(t, "custom", d.Info().Source, "--source must land in BuildInfo.Source")
}

func TestImportTsvCommand_DefaultSourceIsTsv(t *testing.T) {
	tsvPath := writeFixtureTSV(t, importTSVFixture)
	out := filepath.Join(t.TempDir(), "out.dat")

	root := newTestRootCmd(newImportCommand())
	root.SetArgs([]string{"import", "tsv", tsvPath, "-o", out})
	require.NoError(t, root.Execute())

	d, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, d.Close()) }()
	require.NotNil(t, d.Info())
	assert.Equal(t, "tsv", d.Info().Source, "Source defaults to tsv when --source is absent")
}

func TestImportTsvCommand_MissingOutput(t *testing.T) {
	tsvPath := writeFixtureTSV(t, importTSVFixture)

	root := newTestRootCmd(newImportCommand())
	root.SetArgs([]string{"import", "tsv", tsvPath})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "-o", "error must name the missing -o flag")
}

func TestImportTsvCommand_BadInputPath(t *testing.T) {
	root := newTestRootCmd(newImportCommand())
	root.SetArgs([]string{"import", "tsv", filepath.Join(t.TempDir(), "missing.tsv"), "-o", filepath.Join(t.TempDir(), "out.dat")})
	err := root.Execute()
	require.Error(t, err)
}

func TestImportTsvCommand_MalformedRowNamesLine(t *testing.T) {
	tsvPath := writeFixtureTSV(t, "кот\tкот\tNOUN\nодно\n")

	root := newTestRootCmd(newImportCommand())
	root.SetArgs([]string{"import", "tsv", tsvPath, "-o", filepath.Join(t.TempDir(), "out.dat")})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2", "error must name the offending line number")
}
