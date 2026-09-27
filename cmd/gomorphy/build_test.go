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

func TestBuildCommand_OpenCorpora_WithInput(t *testing.T) {
	xmlPath := filepath.Join(t.TempDir(), "dict.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(fixtureXML("кот")), 0o644))
	outPath := filepath.Join(t.TempDir(), "out.dat")

	root := newTestRootCmd(newBuildCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"build", "opencorpora", "-i", xmlPath, "-o", outPath})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "saved")

	d, err := morphology.Open(outPath)
	require.NoError(t, err)
	defer func() { _ = d.Close() }()
	assert.NotEmpty(t, d.Parse("кот"))
}

func TestBuildCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "klingon"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}

func TestBuildCommand_MissingInputFile(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "opencorpora", "-i", "/no/such/file.xml", "-o", filepath.Join(t.TempDir(), "out.dat")})
	assert.Error(t, root.Execute())
}

// TestBuildCommand_PyMorphy_MissingDir is a smoke test for the "pymorphy"
// case's wiring (error propagation from morphology.OpenPyMorphyDense
// reaching the CLI). The dense-alphabet build itself — the actual
// behavior this case now defaults to — is covered exhaustively at the
// library level (pkg/morphology/dense_test.go, save_test.go); building a
// real pymorphy2-format fixture directory here would need
// pkg/morphology/internal, which cmd/gomorphy cannot import.
func TestBuildCommand_PyMorphy_MissingDir(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "pymorphy", "-i", "/no/such/dir", "-o", filepath.Join(t.TempDir(), "out.dat")})
	assert.Error(t, root.Execute())
}

// TestBuildCommand_UniMorph_WithInput exercises the "unimorph" case with
// a real fixture — unlike pymorphy2 (a binary format needing
// pkg/morphology/internal, which cmd/gomorphy cannot import), a UniMorph
// TSV is plain text, so a full positive-path test is possible here.
func TestBuildCommand_UniMorph_WithInput(t *testing.T) {
	tsvPath := filepath.Join(t.TempDir(), "rus.tsv")
	require.NoError(t, os.WriteFile(tsvPath, []byte("кот\tкот\tN;NOM;SG\nкот\tкота\tN;ACC;SG\n"), 0o644))
	outPath := filepath.Join(t.TempDir(), "out.dat")

	root := newTestRootCmd(newBuildCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"build", "unimorph", "-i", tsvPath, "-o", outPath, "--lang", "ru"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "saved")

	d, err := morphology.Open(outPath)
	require.NoError(t, err)
	defer func() { _ = d.Close() }()
	readings := d.Parse("кота")
	require.NotEmpty(t, readings)
	assert.Equal(t, "кот", readings[0].Normal)
	assert.Equal(t, "N;ACC;SG", readings[0].Tag)
}

func TestBuildCommand_UniMorph_UnsupportedLanguage(t *testing.T) {
	tsvPath := filepath.Join(t.TempDir(), "rus.tsv")
	require.NoError(t, os.WriteFile(tsvPath, []byte("кот\tкот\tN;NOM;SG\n"), 0o644))

	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "unimorph", "-i", tsvPath, "-o", filepath.Join(t.TempDir(), "out.dat"), "--lang", "en"})
	assert.Error(t, root.Execute())
}

// threeCatsXML is a minimal OpenCorpora dict.xml with three lemmas sharing
// one paradigm (like fixtureXML's, nomn form only) — pymorphy2-style
// pruning (≥3 lemmas per paradigm) needs that to predict an unknown word.
const threeCatsXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">сущ</grammeme>
  <grammeme id="nomn">им.</grammeme>
  <grammeme id="anim">од.</grammeme>
  <grammeme id="masc">м.</grammeme>
  <grammeme id="sing">ед.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот">
   <l t="кот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="кот"><g v="nomn"/></f>
  </lemma>
  <lemma id="2" text="лот">
   <l t="лот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="лот"><g v="nomn"/></f>
  </lemma>
  <lemma id="3" text="скот">
   <l t="скот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="скот"><g v="nomn"/></f>
  </lemma>
 </lemmata>
</dictionary>`

func TestBuildCommand_NoPrediction(t *testing.T) {
	xmlPath := filepath.Join(t.TempDir(), "dict.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(threeCatsXML), 0o644))

	for _, noPrediction := range []bool{false, true} {
		outPath := filepath.Join(t.TempDir(), "out.dat")
		args := []string{"build", "opencorpora", "-i", xmlPath, "-o", outPath}
		if noPrediction {
			args = append(args, "--no-prediction")
		}
		root := newTestRootCmd(newBuildCommand())
		root.SetOut(&bytes.Buffer{})
		root.SetArgs(args)
		require.NoError(t, root.Execute())

		d, err := morphology.Open(outPath)
		require.NoError(t, err)
		if noPrediction {
			assert.Nil(t, d.Parse("бот"), "--no-prediction")
		} else {
			assert.NotEmpty(t, d.Parse("бот"), "prediction by default (like кот)")
		}
		require.NoError(t, d.Close())
	}
}

func TestBuildCommand_NoPredictionRejectedForPymorphy(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "pymorphy", "--no-prediction", "-i", t.TempDir(), "-o", filepath.Join(t.TempDir(), "x.dat")})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--no-prediction")
}
