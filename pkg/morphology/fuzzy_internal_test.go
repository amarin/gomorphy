package morphology

import (
	"sort"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildDenseFuzzyDAWG builds a real DAWG whose keys are words encoded
// through a width-wide DenseAlphabet built from words itself — the same
// shape RecompileDense produces, just with an arbitrary width instead of
// always 1. Used to exercise fuzzyWalkShard/decodeTail/decodeWord's
// width>1 path, which nothing in production builds today (RecompileDense
// only ever uses width 1) but which the code must still handle correctly.
func buildDenseFuzzyDAWG(t *testing.T, width int, words []string) (*internal.DAWG, *internal.DenseAlphabet) {
	t.Helper()
	alphabet, err := internal.NewDenseAlphabet(width, words)
	require.NoError(t, err)

	keys := make([]string, len(words))
	values := make([]uint32, len(words))
	for i, w := range words {
		enc, err := alphabet.Encode(w)
		require.NoError(t, err)
		keys[i] = string(enc)
		values[i] = uint32(i)
	}
	dawg, err := internal.BuildDAWGWithValues(keys, values)
	require.NoError(t, err)
	return dawg, alphabet
}

func TestFuzzyWalkShardDenseAlphabetWidth2(t *testing.T) {
	words := []string{"кот", "код", "дом", "яснее"}
	dawg, alphabet := buildDenseFuzzyDAWG(t, 2, words)

	got := fuzzyWalkShard(dawg, alphabet, nil, "код", 1)
	sort.Slice(got, func(i, j int) bool { return got[i].Word < got[j].Word })

	want := []FuzzyMatch{
		{Word: "код", Distance: 0},
		{Word: "кот", Distance: 1},
	}
	assert.Equal(t, want, got)
}

func TestMaxWordRunesInShardDenseAlphabetWidth2(t *testing.T) {
	words := []string{"кот", "яснее"}
	dawg, _ := buildDenseFuzzyDAWG(t, 2, words)

	assert.Equal(t, 5, maxWordRunesInShard(dawg, 2), "яснее is 5 runes, must not be reported as 10 (raw byte count) or halved incorrectly")
}
