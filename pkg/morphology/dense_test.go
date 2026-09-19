package morphology_test

import (
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenPyMorphyDense_MatchesRawParse(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)

	raw, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	sample := []string{
		"кот", "кота", "мышь", "мыши", "ежик", "ёжик", "ёж",
		"код", "крот", "год", "дом", "дым", "стол", "стул", "лес",
	}
	for _, word := range sample {
		rawReadings := raw.Parse(word)
		denseReadings := dense.Parse(word)
		require.Equal(t, len(rawReadings), len(denseReadings), "word %q: reading count differs", word)
		for i := range rawReadings {
			assert.Equal(t, rawReadings[i].Word, denseReadings[i].Word, "word %q reading %d: Word", word, i)
			assert.Equal(t, rawReadings[i].Normal, denseReadings[i].Normal, "word %q reading %d: Normal", word, i)
			assert.Equal(t, rawReadings[i].Tag, denseReadings[i].Tag, "word %q reading %d: Tag", word, i)
			assert.Equal(t, rawReadings[i].Para, denseReadings[i].Para, "word %q reading %d: Para", word, i)
			assert.Equal(t, rawReadings[i].Form, denseReadings[i].Form, "word %q reading %d: Form", word, i)
		}
	}
}

func TestOpenPyMorphyDense_UnknownWordReturnsNil(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)

	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)
	assert.Nil(t, dense.Parse("несуществующееслово"))
}

// TestOpenPyMorphyDense_PredictionAndProbability confirms (item 3 of the
// dense-alphabet backlog, docs/en/todo.md) that the Prediction/Probability
// DAWGs behave identically on a dense dictionary. RecompileDense
// deliberately never touches Prediction/Probability (see recompile.go's
// doc comment) — they stay raw UTF-8 regardless of Words[0]'s alphabet —
// and predictForPrefix (parse.go) already always queries Prediction with
// a nil alphabet by design. This test is the "investigation": it exists
// to catch a regression if that design assumption is ever violated, not
// because a bug was found.
func TestOpenPyMorphyDense_PredictionAndProbability(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	prediction := map[string]uint32{
		"ёнок" + payloadSeparator + b64(predictionValue(3, 2, 1)): 0,
		"ёнка" + payloadSeparator + b64(predictionValue(1, 2, 1)): 0,
	}
	prob := map[string]uint32{
		"кот:NOUN,anim,masc,sing,nomn": 500,
	}
	dir := buildFixtureDir(t, words, prediction, prob)

	raw, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	// "котёнка" is out-of-dictionary: Parse must fall through to predict(),
	// matching against the "ёнка"/"ёнок" suffixes in the Prediction DAWG.
	rawPredicted := raw.Parse("котёнка")
	densePredicted := dense.Parse("котёнка")
	require.NotEmpty(t, rawPredicted, "fixture must actually predict something for this test to be meaningful")
	assert.Equal(t, rawPredicted, densePredicted)

	// "кот" is in-dictionary and carries a probability entry.
	rawKot := raw.Parse("кот")
	denseKot := dense.Parse("кот")
	assert.Equal(t, rawKot, denseKot)
	assert.NotZero(t, maxReadingProb(rawKot), "fixture must actually contain a probability for this test to be meaningful")

	// The same, once more after a full .dat SaveTo/Open round-trip (not
	// just the in-memory OpenPyMorphyDense above) — Prediction/Probability
	// are their own optional sections, written/read independently of the
	// new "alphabet" section.
	out := filepath.Join(t.TempDir(), "dense.dat")
	require.NoError(t, dense.SaveTo(out))
	reopened, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, reopened.Close()) }()

	assert.Equal(t, rawPredicted, reopened.Parse("котёнка"))
	assert.Equal(t, rawKot, reopened.Parse("кот"))
}
