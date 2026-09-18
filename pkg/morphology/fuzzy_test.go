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

func TestFuzzyRuneMetricYo(t *testing.T) {
	d := fuzzyDict(t)

	assert.Equal(t, map[string]int{"ежик": 0, "ёжик": 1},
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

// TestFuzzyDenseAlphabetReturnsNil — a finding from the final review: on a
// dense-alphabet dictionary (OpenPyMorphyDense) the internal traversal in
// fuzzy.go decodes DAWG bytes as raw UTF-8, which for 1-byte dense codes
// "successfully" produces garbage strings with no error. Fuzzy/FuzzyTop must
// instead return nil, not garbage, and must not panic.
func TestFuzzyDenseAlphabetReturnsNil(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)

	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	assert.Nil(t, dense.Fuzzy("кот", 3))
	assert.Nil(t, dense.FuzzyTop("кот", 3))
}
