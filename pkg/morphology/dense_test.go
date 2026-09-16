package morphology_test

import (
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
