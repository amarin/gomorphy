package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// twoDictFixture builds two independent pymorphy2 fixture dictionaries:
// dictA knows "кот" and "яблоко"; dictB knows "кот" and "груша". "кот" is
// deliberately present in both, to exercise the overlap case; "яблоко" and
// "груша" each exist in exactly one dictionary. dictA also carries
// meta.json (source_version/source_revision), dictB does not, to exercise
// DictInfo's SourceVersion for a populated vs. absent case side by side.
func twoDictFixture(t *testing.T) (dictA, dictB *morphology.Dictionary) {
	t.Helper()

	wordsA := map[string]uint32{}
	addWord(wordsA, "кот", 0, 0)
	addWord(wordsA, "яблоко", 2, 0)
	dirA := buildFixtureDir(t, wordsA, nil, nil)
	writeFile(t, dirA, "meta.json", []byte(`[["source_version","0.92"],["source_revision","417257"]]`))
	dictA, err := morphology.OpenPyMorphy(dirA)
	require.NoError(t, err)

	wordsB := map[string]uint32{}
	addWord(wordsB, "кот", 0, 0)
	addWord(wordsB, "груша", 2, 0)
	dirB := buildFixtureDir(t, wordsB, nil, nil)
	dictB, err = morphology.OpenPyMorphy(dirB)
	require.NoError(t, err)

	return dictA, dictB
}

func TestMultiDictionary_Len(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)
	assert.Equal(t, 2, m.Len())

	empty := morphology.NewMultiDictionary()
	assert.Equal(t, 0, empty.Len())
}

func TestMultiDictionary_ParseWordInOneDict(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	readings := m.Parse("яблоко")
	require.NotEmpty(t, readings)
	for _, r := range readings {
		assert.Equal(t, 0, r.Dict, "яблоко only exists in dictA (index 0)")
	}

	readings = m.Parse("груша")
	require.NotEmpty(t, readings)
	for _, r := range readings {
		assert.Equal(t, 1, r.Dict, "груша only exists in dictB (index 1)")
	}
}

func TestMultiDictionary_ParseWordInBothDicts(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	readings := m.Parse("кот")
	require.NotEmpty(t, readings)

	var dicts []int
	for _, r := range readings {
		dicts = append(dicts, r.Dict)
	}
	assert.Contains(t, dicts, 0, "кот exists in dictA")
	assert.Contains(t, dicts, 1, "кот exists in dictB")

	// Registration order: every dictA reading must precede every dictB
	// reading (Parse concatenates per-dictionary results in order, it
	// does not interleave or sort across dictionaries).
	lastA := -1
	firstB := len(readings)
	for i, r := range readings {
		if r.Dict == 0 && i > lastA {
			lastA = i
		}
		if r.Dict == 1 && i < firstB {
			firstB = i
		}
	}
	assert.Less(t, lastA, firstB, "all dictA readings must come before all dictB readings")
}

func TestMultiDictionary_ParseWordInNoDict(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	assert.Nil(t, m.Parse("несуществующееслово"))
}

func TestMultiDictionary_Lemma(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	refs := m.Lemma("кот")
	require.NotEmpty(t, refs)

	var dicts []int
	for _, r := range refs {
		dicts = append(dicts, r.Dict)
	}
	assert.Contains(t, dicts, 0)
	assert.Contains(t, dicts, 1)
}

func TestMultiDictionary_DictInfo(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	infoA := m.DictInfo(0)
	require.NotNil(t, infoA)
	assert.Equal(t, "pymorphy2", infoA.Source)
	assert.Equal(t, "0.92/417257", infoA.SourceVersion)

	infoB := m.DictInfo(1)
	require.NotNil(t, infoB)
	assert.Equal(t, "pymorphy2", infoB.Source)
	assert.Empty(t, infoB.SourceVersion, "dictB has no meta.json")

	assert.Nil(t, m.DictInfo(-1))
	assert.Nil(t, m.DictInfo(2))
}

func TestMultiDictionary_Fuzzy(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	// "яблоко" only exists in dictA, "груша" only in dictB - an exact
	// (maxDist=0) fuzzy search for each should find exactly that one word,
	// tagged with its dictionary's index.
	matches := m.Fuzzy("яблоко", 0)
	require.NotEmpty(t, matches)
	for _, mt := range matches {
		assert.Equal(t, "яблоко", mt.Word)
		assert.Equal(t, 0, mt.Dict)
	}

	matches = m.Fuzzy("груша", 0)
	require.NotEmpty(t, matches)
	for _, mt := range matches {
		assert.Equal(t, "груша", mt.Word)
		assert.Equal(t, 1, mt.Dict)
	}
}

func TestMultiDictionary_FuzzyTop_MergesAcrossDicts(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	// "кот" is an exact match in both dictionaries (distance 0). Asking
	// for the top 1 word overall must not silently prefer dictA just
	// because it was registered first: with maxWords=1, only one of the
	// two distance-0 "кот" matches can survive the merge. Assert the cap
	// is honored globally, not per-dictionary (a naive concatenation of
	// each dict's own FuzzyTop would return 2 results here, not 1).
	top := m.FuzzyTop("кот", 1)
	require.Len(t, top, 1)
	assert.Equal(t, "кот", top[0].Word)
	assert.Equal(t, 0, top[0].Distance)

	// A larger cap must surface matches from both dictionaries, each
	// tagged correctly, sorted by (distance, word).
	top = m.FuzzyTop("кот", 10)
	require.NotEmpty(t, top)
	var dicts []int
	for _, mt := range top {
		dicts = append(dicts, mt.Dict)
	}
	assert.Contains(t, dicts, 0)
	assert.Contains(t, dicts, 1)
	for i := 1; i < len(top); i++ {
		prev, cur := top[i-1], top[i]
		if prev.Distance == cur.Distance {
			assert.LessOrEqual(t, prev.Word, cur.Word, "ties must be ordered by word")
		} else {
			assert.Less(t, prev.Distance, cur.Distance)
		}
	}
}

func TestMultiDictionary_FuzzyTop_ZeroMaxWordsIsExactSearch(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	top := m.FuzzyTop("кот", 0)
	require.NotEmpty(t, top)
	for _, mt := range top {
		assert.Equal(t, "кот", mt.Word)
		assert.Equal(t, 0, mt.Distance)
	}
}

func TestMultiDictionary_Close(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	assert.NoError(t, m.Close())
}

func TestMultiDictionary_Empty(t *testing.T) {
	m := morphology.NewMultiDictionary()

	assert.Nil(t, m.Parse("кот"))
	assert.Nil(t, m.Lemma("кот"))
	assert.NoError(t, m.Close())
}
