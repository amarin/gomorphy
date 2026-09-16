package pymorphy2_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecompileDense(t *testing.T) {
	dir := makeFixtureDir(t)

	raw, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)

	dense, err := pymorphy2.RecompileDense(dir)
	require.NoError(t, err)
	require.NotNil(t, dense)
	require.NotNil(t, dense.Alphabet, "RecompileDense must set Alphabet")
	assert.Equal(t, "dense-1", dense.Alphabet.Name())

	for _, word := range []string{"кот", "кота"} {
		rawItems := raw.Words[0].SimilarItems(word, raw.CharPolicy, nil)
		denseItems := dense.Words[0].SimilarItems(word, dense.CharPolicy, dense.Alphabet)
		require.Len(t, denseItems, len(rawItems), "word %q", word)
		for i := range rawItems {
			assert.Equal(t, rawItems[i].Key, denseItems[i].Key, "word %q item %d", word, i)
			assert.Equal(t, rawItems[i].Values, denseItems[i].Values, "word %q item %d", word, i)
		}
	}

	// Everything except Words[0] must be copied through unchanged.
	assert.Equal(t, raw.Suffixes, dense.Suffixes)
	assert.Equal(t, raw.Prefixes, dense.Prefixes)
	assert.Equal(t, raw.Paradigms, dense.Paradigms)
	assert.Equal(t, raw.TagSet, dense.TagSet)
}

func TestRecompileDense_MissingDir(t *testing.T) {
	_, err := pymorphy2.RecompileDense(t.TempDir() + "/does-not-exist")
	require.Error(t, err)
}
