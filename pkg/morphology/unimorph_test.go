package morphology_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const uniMorphTSV = "кот\tкот\tN;NOM;SG\n" +
	"кот\tкота\tN;ACC;SG\n" +
	"мышь\tмыши\tN;GEN;SG\n"

func TestCompileFromUniMorph(t *testing.T) {
	d, err := morphology.CompileFromUniMorph(strings.NewReader(uniMorphTSV), morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)

	readings := d.Parse("кота")
	require.NotEmpty(t, readings)
	assert.Equal(t, "кот", readings[0].Normal)
	assert.Equal(t, "N;ACC;SG", readings[0].Tag)

	assert.Equal(t, "ru", d.Language())
	assert.Equal(t, "unimorph", d.TagSetName())
}

func TestCompileFromUniMorphFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rus.tsv")
	require.NoError(t, os.WriteFile(path, []byte(uniMorphTSV), 0o644))

	d, err := morphology.CompileFromUniMorphFile(path, morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)
	require.NotEmpty(t, d.Parse("кота"))
}

func TestCompileFromUniMorph_UnsupportedLanguage(t *testing.T) {
	_, err := morphology.CompileFromUniMorph(strings.NewReader(uniMorphTSV), morphology.UniMorphOptions{Language: "en"})
	assert.Error(t, err)
}

// TestCompileFromUniMorphDense_MatchesRawParse mirrors
// TestCompileFromXMLDense_MatchesRawParse/TestOpenPyMorphyDense_MatchesRawParse:
// the dense variant of UniMorph import (gomorphy build unimorph's
// default, see cmd/gomorphy) must give identical readings to the raw
// compile.
func TestCompileFromUniMorphDense_MatchesRawParse(t *testing.T) {
	opts := morphology.UniMorphOptions{Language: "ru"}
	raw, err := morphology.CompileFromUniMorph(strings.NewReader(uniMorphTSV), opts)
	require.NoError(t, err)
	dense, err := morphology.CompileFromUniMorphDense(strings.NewReader(uniMorphTSV), opts)
	require.NoError(t, err)

	for _, word := range []string{"кот", "кота", "мыши"} {
		rawReadings := raw.Parse(word)
		denseReadings := dense.Parse(word)
		require.Equal(t, len(rawReadings), len(denseReadings), "word %q", word)
		for i := range rawReadings {
			assert.Equal(t, rawReadings[i].Word, denseReadings[i].Word, "word %q reading %d", word, i)
			assert.Equal(t, rawReadings[i].Normal, denseReadings[i].Normal, "word %q reading %d", word, i)
			assert.Equal(t, rawReadings[i].Tag, denseReadings[i].Tag, "word %q reading %d", word, i)
		}
	}
}

func TestCompileFromUniMorphDense_SaveOpenRoundtrip(t *testing.T) {
	dense, err := morphology.CompileFromUniMorphDense(strings.NewReader(uniMorphTSV), morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)

	out := filepath.Join(t.TempDir(), "unimorph-dense.dat")
	require.NoError(t, dense.SaveTo(out))

	got, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()

	assert.Equal(t, dense.Parse("кота"), got.Parse("кота"))
}

// mixedCaseUniMorphTSV mirrors uniMorphTSV but with the lemma/wordform
// carrying upper-case letters (as real UniMorph proper-noun rows do, e.g.
// "Аббас"). ImportFromTSV lower-cases lemma/wordform text on read so such
// rows stay reachable by Parse/IsKnown/Fuzzy, which all lower-case their
// query — see pkg/morphology/importers/unimorph/import.go.
const mixedCaseUniMorphTSV = "Аббас\tАббас\tN;NOM;SG\n" +
	"Аббас\tАббаса\tN;GEN;SG\n"

func TestCompileFromUniMorph_MixedCaseReachable(t *testing.T) {
	d, err := morphology.CompileFromUniMorph(strings.NewReader(mixedCaseUniMorphTSV), morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)

	readings := d.Parse("Аббас")
	require.NotEmpty(t, readings, "Parse must find the mixed-case UniMorph entry via its lower-cased query")
	assert.Equal(t, "аббас", readings[0].Normal)
	assert.False(t, readings[0].Predicted, "an exact dictionary match must not be Predicted")

	assert.True(t, d.IsKnown("Аббас"))

	matches := d.Fuzzy("Аббас", 0)
	require.NotEmpty(t, matches, "Fuzzy at distance 0 must find the lower-cased stored word")
	assert.Equal(t, "аббас", matches[0].Word)
}
