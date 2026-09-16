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
