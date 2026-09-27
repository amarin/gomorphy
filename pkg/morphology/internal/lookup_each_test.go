package internal

import (
	"bytes"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flatItem struct {
	key   string
	value []byte
}

func flattenSimilar(items []Item) []flatItem {
	var out []flatItem
	for _, it := range items {
		for _, v := range it.Values {
			if v != nil {
				out = append(out, flatItem{it.Key, v})
			}
		}
	}
	return out
}

func collectLookup(d *DAWG, key string, pol *CharPolicy, a Alphabet) []flatItem {
	var out []flatItem
	d.LookupEach(key, pol, a, func(found string, value []byte) {
		out = append(out, flatItem{found, bytes.Clone(value)})
	})
	return out
}

// lookupCorpus: several е/ё variants of one word, multi-value keys and
// 6-byte (prediction-shaped) values.
func lookupCorpus() ([]string, []uint32) {
	keys := []string{"ежик", "ёжик", "ежик", "ёжиек", "еле", "ёлё", "елё", "кот", "кота", "кот"}
	values := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	return keys, values
}

func TestLookupEachMatchesSimilarItems(t *testing.T) {
	keys, values := lookupCorpus()
	raw, err := BuildDAWGWithValues(keys, values)
	require.NoError(t, err)

	alphabet, err := NewDenseAlphabetFor(keys)
	require.NoError(t, err)
	encoded := make([]string, len(keys))
	for i, k := range keys {
		enc, err := alphabet.Encode(k)
		require.NoError(t, err)
		encoded[i] = string(enc)
	}
	dense, err := BuildDAWGWithValues(encoded, values)
	require.NoError(t, err)

	for _, pol := range []*CharPolicy{nil, NewCharPolicy(), RussianCharPolicy()} {
		for _, q := range []string{"ежик", "ёжик", "еле", "кот", "кота", "мышь", "", "е"} {
			want := flattenSimilar(raw.SimilarItems(q, pol, nil))
			assert.Equal(t, want, collectLookup(raw, q, pol, nil), "raw %q", q)

			wantDense := flattenSimilar(dense.SimilarItems(q, pol, alphabet))
			assert.Equal(t, wantDense, collectLookup(dense, q, pol, alphabet), "dense %q", q)
		}
	}
}

func TestLookupEachSixByteValues(t *testing.T) {
	d, err := BuildDAWGWithValuesBytes(
		[]string{"ёнка", "ёнка"},
		[][]byte{{0, 3, 0, 0, 0, 1}, {0, 1, 0, 2, 0, 0}},
	)
	require.NoError(t, err)
	want := flattenSimilar(d.SimilarItems("енка", RussianCharPolicy(), nil))
	require.Len(t, want, 2)
	assert.Equal(t, want, collectLookup(d, "енка", RussianCharPolicy(), nil))
}

func TestLookupEachFoundIsKeyWithoutSubstitution(t *testing.T) {
	d, err := BuildDAWGWithValues([]string{"кот"}, []uint32{1})
	require.NoError(t, err)
	key := "кот"
	d.LookupEach(key, RussianCharPolicy(), nil, func(found string, _ []byte) {
		assert.Equal(t, key, found)
	})
	if raceEnabled {
		return
	}
	allocs := testing.AllocsPerRun(100, func() {
		d.LookupEach(key, RussianCharPolicy(), nil, func(string, []byte) {})
	})
	assert.Equal(t, 0.0, allocs)
}

func TestFindJoined(t *testing.T) {
	d, err := BuildIntDAWG([]string{"кот:NOUN", "кот:VERB", "кота:NOUN", ":NOUN"}, []uint32{500, 100, 7, 42})
	require.NoError(t, err)
	for _, tc := range []struct{ a, b string }{
		{"кот", "NOUN"}, {"кот", "VERB"}, {"кота", "NOUN"}, {"кот", "ADJF"}, {"мышь", "NOUN"}, {"", "NOUN"},
	} {
		assert.Equal(t, d.Find(tc.a+":"+tc.b), d.FindJoined(tc.a, ':', tc.b), "%s:%s", tc.a, tc.b)
	}
	if !raceEnabled {
		assert.Equal(t, 0.0, testing.AllocsPerRun(100, func() { _ = d.FindJoined("кот", ':', "NOUN") }))
	}
}

func TestEncodeRuneMatchesEncode(t *testing.T) {
	var many []string
	for r := rune(0x400); r < 0x400+300; r++ {
		many = append(many, string(r))
	}
	for _, corpus := range [][]string{{"кот", "ёж"}, many} {
		a, err := NewDenseAlphabetFor(corpus)
		require.NoError(t, err)
		for _, w := range corpus {
			for _, r := range w {
				want, err := a.Encode(string(r))
				require.NoError(t, err)
				code, n, ok := a.EncodeRune(r)
				require.True(t, ok)
				assert.Equal(t, want, code[:n])
			}
		}
		_, _, ok := a.EncodeRune('ﬀ')
		assert.False(t, ok)
	}
}

func TestLookupEachUsesTestdawgFixtures(t *testing.T) {
	// The same fixture shape SimilarItems tests use (testdawg.Build).
	dict, guide := testdawg.Build(map[string]uint32{
		"ежик" + string(PayloadSeparator) + b64([]byte{0, 1, 0, 2}): 1,
		"ёжик" + string(PayloadSeparator) + b64([]byte{3, 4, 5, 6}): 2,
	})
	d := NewDAWG(dict, guide)
	assert.Equal(t, flattenSimilar(d.SimilarItems("ежик", RussianCharPolicy(), nil)),
		collectLookup(d, "ежик", RussianCharPolicy(), nil))
}
