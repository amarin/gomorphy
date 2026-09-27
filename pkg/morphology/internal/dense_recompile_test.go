package internal

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestDictionary(t *testing.T, shardWords [][]string, shardValues [][]uint32) *Dictionary {
	t.Helper()
	require.Len(t, shardValues, len(shardWords))

	suffixes := make([][]string, len(shardWords))
	paradigms := make([][]Paradigm, len(shardWords))
	words := make([]*DAWG, len(shardWords))
	for i := range shardWords {
		suffixes[i] = []string{}
		paradigms[i] = []Paradigm{}
		dawg, err := BuildDAWGWithValues(shardWords[i], shardValues[i])
		require.NoError(t, err)
		words[i] = dawg
	}
	return NewDictionary("ru", NewTagSet("test"), suffixes, nil, paradigms, words, nil)
}

func TestRecompileDenseSingleShard(t *testing.T) {
	d := buildTestDictionary(t,
		[][]string{{"кот", "кота", "мышь"}},
		[][]uint32{{1, 2, 3}},
	)

	require.NoError(t, RecompileDense(d))

	require.NotNil(t, d.Alphabet)
	assert.Equal(t, "dense-1", d.Alphabet.Name())
	require.Len(t, d.Words, 1)

	for word, want := range map[string]uint32{"кот": 1, "кота": 2, "мышь": 3} {
		items := d.Words[0].SimilarItems(word, d.CharPolicy, d.Alphabet)
		require.Len(t, items, 1, "word %q", word)
		require.Len(t, items[0].Values, 1, "word %q", word)
		assert.Equal(t, want, binary.BigEndian.Uint32(items[0].Values[0]), "word %q", word)
	}
}

// TestRecompileDenseMultiShard is the case pymorphy2's single-shard-only
// recompile never had to handle: OpenCorpora dictionaries can have
// several shards. Confirms every shard's words survive the rebuild under
// one Alphabet shared across all shards (see "Agreed design decisions",
// item 1, docs/en/implementation/pymorphy2-dense-alphabet.md) — not a
// separate alphabet per shard.
func TestRecompileDenseMultiShard(t *testing.T) {
	d := buildTestDictionary(t,
		[][]string{{"кот", "кота"}, {"дом", "дома"}},
		[][]uint32{{1, 2}, {3, 4}},
	)
	origSuffixes := d.Suffixes

	require.NoError(t, RecompileDense(d))

	require.NotNil(t, d.Alphabet)
	require.Len(t, d.Words, 2)

	// Paradigms/Suffixes/Prefixes are untouched — RecompileDense only
	// ever replaces Words and Alphabet.
	assert.Same(t, &origSuffixes[0], &d.Suffixes[0], "Suffixes must not be reallocated")

	for word, want := range map[string]uint32{"кот": 1, "кота": 2} {
		items := d.Words[0].SimilarItems(word, d.CharPolicy, d.Alphabet)
		require.Len(t, items, 1, "word %q", word)
		assert.Equal(t, want, binary.BigEndian.Uint32(items[0].Values[0]), "word %q", word)
	}
	for word, want := range map[string]uint32{"дом": 3, "дома": 4} {
		items := d.Words[1].SimilarItems(word, d.CharPolicy, d.Alphabet)
		require.Len(t, items, 1, "word %q", word)
		assert.Equal(t, want, binary.BigEndian.Uint32(items[0].Values[0]), "word %q", word)
	}

	// The shared-alphabet claim: a rune present in both shards' words
	// ("о", in "кот" and "дом") must encode to the identical byte in
	// both — there is only one Alphabet object, referenced by both
	// shards' SimilarItems calls above, so this is really just making
	// that sharing explicit and named.
	codeO, err := d.Alphabet.Encode("о")
	require.NoError(t, err)
	assert.Len(t, codeO, 1)
}

func TestRecompileDenseEmptyShardIsHarmless(t *testing.T) {
	d := buildTestDictionary(t,
		[][]string{{"кот"}, {}},
		[][]uint32{{1}, {}},
	)

	require.NoError(t, RecompileDense(d))
	require.Len(t, d.Words, 2)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy, d.Alphabet)
	require.Len(t, items, 1)
}

// 255 distinct runes exceed width 1's capacity of 254: RecompileDense
// falls back to a width-2 alphabet instead of failing.
func TestRecompileDenseFallsBackToWidth2(t *testing.T) {
	var words []string
	for r := rune(0x400); r < 0x400+255; r++ {
		words = append(words, string(r))
	}
	values := make([]uint32, len(words))
	for i := range values {
		values[i] = uint32(i)
	}
	d := buildTestDictionary(t, [][]string{words}, [][]uint32{values})

	require.NoError(t, RecompileDense(d))
	require.NotNil(t, d.Alphabet)
	assert.Equal(t, 2, d.Alphabet.Width())
	for i, w := range words {
		items := d.Words[0].SimilarItems(w, nil, d.Alphabet)
		require.Len(t, items, 1, w)
		require.Len(t, items[0].Values, 1)
		assert.Equal(t, uint32(i), binary.BigEndian.Uint32(items[0].Values[0]))
	}
}

func TestNewDenseAlphabetForWidths(t *testing.T) {
	a, err := NewDenseAlphabetFor([]string{"кот", "мышь"})
	require.NoError(t, err)
	assert.Equal(t, 1, a.Width())

	var many []string
	for r := rune(0x10000); r < 0x10000+255; r++ {
		many = append(many, string(r))
	}
	a, err = NewDenseAlphabetFor(many)
	require.NoError(t, err)
	assert.Equal(t, 2, a.Width())

	for r := rune(0x10000 + 255); r < 0x10000+254*254+1; r++ {
		many = append(many, string(r))
	}
	_, err = NewDenseAlphabetFor(many)
	assert.Error(t, err, "64517 distinct runes exceed width 2")
}
