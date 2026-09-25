package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fuzzyDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	m := map[string]uint32{}
	stdWords(m)
	return buildFixture(t, m, nil, nil)
}

func countByDistance(t *testing.T, got []morphology.FuzzyMatch) map[string]int {
	t.Helper()
	out := make(map[string]int, len(got))
	for _, m := range got {
		if _, dup := out[m.Word]; dup {
			t.Errorf("duplicate fuzzy match %q", m.Word)
		}
		out[m.Word] = m.Distance
	}
	return out
}

func TestFuzzyKot(t *testing.T) {
	d := fuzzyDict(t)

	got := d.Fuzzy("кот", 1)
	want := map[string]int{"кот": 0, "код": 1, "крот": 1, "кота": 1}

	require.Len(t, got, len(want))
	assert.Equal(t, want, countByDistance(t, got))
}

// The fixture uses RussianCharPolicy (е→ё): a query «е» matches a stored
// «ё» at cost 0, like Parse. The substitution is one-way: a query «ё» still
// costs 1 against a stored «е».
func TestFuzzyRuneMetricYo(t *testing.T) {
	d := fuzzyDict(t)

	assert.Equal(t, map[string]int{"ежик": 0, "ёжик": 0},
		countByDistance(t, d.Fuzzy("ежик", 1)))
	assert.Equal(t, map[string]int{"ёж": 0, "ёжик": 2},
		countByDistance(t, d.Fuzzy("ёж", 2)))
}

func TestFuzzyOrderingAndRange(t *testing.T) {
	d := fuzzyDict(t)

	got := d.Fuzzy("кот", 2)
	prevDist, prevWord := -1, ""
	for _, m := range got {
		if m.Distance < prevDist || (m.Distance == prevDist && m.Word <= prevWord) {
			t.Fatalf("not ordered by (distance, text): %+v", m)
		}
		prevDist, prevWord = m.Distance, m.Word
	}
}

func TestFuzzyExactProbe(t *testing.T) {
	d := fuzzyDict(t)

	assert.Len(t, d.Fuzzy("кот", 0), 1)
	assert.Empty(t, d.Fuzzy("неттакогослова", 0))
	assert.Len(t, d.Fuzzy("кот", -1), 1, "отрицательное maxDist = точный поиск")
}

func TestFuzzyTopNearest(t *testing.T) {
	d := fuzzyDict(t)

	got := d.FuzzyTop("кот", 3)
	require.Len(t, got, 3)
	assert.Equal(t, "кот", got[0].Word)
	assert.Equal(t, 0, got[0].Distance)
	assert.Equal(t, "код", got[1].Word)
	assert.Equal(t, "кота", got[2].Word)
	assert.Equal(t, 1, got[1].Distance)
	assert.Equal(t, 1, got[2].Distance)
}

func TestFuzzyTopAll(t *testing.T) {
	d := fuzzyDict(t)

	got := d.FuzzyTop("кот", 100)
	assert.Len(t, got, 15, "число уникальных слов фикстуры")
}

func TestFuzzyTopExactProbe(t *testing.T) {
	d := fuzzyDict(t)

	got := d.FuzzyTop("кот", 0)
	require.Len(t, got, 1)
	assert.Equal(t, "кот", got[0].Word)
	assert.Empty(t, d.FuzzyTop("неттакогослова", 0))
}

// TestFuzzyDenseAlphabetMatchesRawResults verifies that Fuzzy/FuzzyTop on a
// dense-alphabet dictionary (OpenPyMorphyDense) give exactly the same
// matches as the identical fixture opened without a dense alphabet
// (OpenPyMorphy): the DAWG traversal now decodes dense-coded bytes through
// Dictionary.Alphabet instead of assuming raw UTF-8.
func TestFuzzyDenseAlphabetMatchesRawResults(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)

	raw, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	for _, tc := range []struct {
		word    string
		maxDist int
	}{
		{"кот", 1},
		{"кот", 2},
		{"ежик", 1},
		{"ёж", 2},
		{"неттакогослова", 0},
	} {
		assert.Equal(t, raw.Fuzzy(tc.word, tc.maxDist), dense.Fuzzy(tc.word, tc.maxDist),
			"Fuzzy(%q, %d)", tc.word, tc.maxDist)
	}

	assert.Equal(t, raw.FuzzyTop("кот", 5), dense.FuzzyTop("кот", 5))
	assert.Equal(t, raw.FuzzyTop("кот", 100), dense.FuzzyTop("кот", 100))
}
