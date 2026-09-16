//go:build integration

// Compares Parse() between the raw pymorphy2 import and RecompileDense's
// dense-alphabet rebuild on a sample of real words. Needs a real pymorphy2
// dictionary directory (see pkg/pymorphy.Loader for how to get one — e.g.
// `.data/pymorphy/data` after a Loader.Sync(false)). Slow: run explicitly:
//
//	GOMORPHY_PYMORPHY2_DIR=.data/pymorphy/data \
//	  go test -tags=integration ./pkg/morphology/ -run TestOpenPyMorphyDense_RealCorpus -v -timeout 20m
package morphology_test

import (
	"os"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenPyMorphyDense_RealCorpus(t *testing.T) {
	dir := os.Getenv("GOMORPHY_PYMORPHY2_DIR")
	if dir == "" {
		t.Skip("GOMORPHY_PYMORPHY2_DIR not set")
	}

	raw, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	sample := []string{
		"все", "занудами", "кот", "кота", "стол", "ежик", "ёжик",
		"пояснее", "поясней", "яснее", "ясней", "дом", "дома", "мама",
		"стали", "шли", "красивее",
	}
	for _, word := range sample {
		rawReadings := raw.Parse(word)
		denseReadings := dense.Parse(word)
		require.Equal(t, len(rawReadings), len(denseReadings), "word %q: reading count differs", word)
		for i := range rawReadings {
			assert.Equal(t, rawReadings[i].Word, denseReadings[i].Word, "word %q reading %d: Word", word, i)
			assert.Equal(t, rawReadings[i].Normal, denseReadings[i].Normal, "word %q reading %d: Normal", word, i)
			assert.Equal(t, rawReadings[i].Tag, denseReadings[i].Tag, "word %q reading %d: Tag", word, i)
		}
	}
}
