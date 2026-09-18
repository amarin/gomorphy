package pymorphy2_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeHomonymFixtureDir is like makeFixtureDir (import_test.go), but
// "кот" carries 2 distinct payload values (a homonym) instead of 1. A
// local copy, not an extension of makeFixtureDir: other tests in
// import_test.go (e.g. TestImportFromDir) assert
// len(items[0].Values) == 1 for "кот", so changing its payload after
// the fact would break their build.
//
// Needed to exercise the fan-out loop `for _, v := range vals` in
// RecompileDense.reading — with one value per word (makeFixtureDir) the
// loop always runs exactly one iteration, so the composition of Walk +
// loop + BuildDAWGWithValues dropping a homonym's extra values would go
// unnoticed. See TestDAWGWalk (pkg/morphology/internal/dawg_test.go) for
// the same technique applied directly at the DAWG.Walk level.
func makeHomonymFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeParadigms(t, dir, [][]uint16{
		{10, 20, 0, 1, 0, 0}, // 2 forms: suffixes[10,20], tags[0,1], prefixes[0,0]
	})

	writeFile(t, dir, "suffixes.json", []byte(`["","кот","кота","x"]`))
	writeFile(t, dir, "paradigm-prefixes.json", []byte(`["","по","наи"]`))
	writeFile(t, dir, "gramtab-opencorpora-int.json", []byte(`["NOUN,anim,masc,sing,nomn","NOUN,anim,masc,sing,gent"]`))

	words, guide := testdawg.Build(map[string]uint32{
		"кот" + payloadSeparator + b64([]byte{0, 0, 0, 0}):  0, // homonym reading 1 (para 0, form 0)
		"кот" + payloadSeparator + b64([]byte{0, 0, 0, 1}):  0, // homonym reading 2 (para 0, form 1)
		"кота" + payloadSeparator + b64([]byte{0, 0, 0, 1}): 0,
	})
	writeFile(t, dir, "words.dawg", testdawg.Marshal(words, guide))

	return dir
}

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

// TestRecompileDense_Homonym checks that RecompileDense doesn't drop
// values for a word with several payload values (a homonym). Separate
// from TestRecompileDense, whose fixture (makeFixtureDir) gives every
// word exactly one value and therefore doesn't really exercise the
// fan-out `for _, v := range vals` loop in RecompileDense (a loop with 1
// iteration is indistinguishable from having no loop at all).
func TestRecompileDense_Homonym(t *testing.T) {
	dir := makeHomonymFixtureDir(t)

	raw, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)

	dense, err := pymorphy2.RecompileDense(dir)
	require.NoError(t, err)
	require.NotNil(t, dense)
	require.NotNil(t, dense.Alphabet, "RecompileDense must set Alphabet")

	rawItems := raw.Words[0].SimilarItems("кот", raw.CharPolicy, nil)
	require.Len(t, rawItems, 1)
	require.Len(t, rawItems[0].Values, 2, "fixture must actually contain a homonym (2+ payload values) for this test to be meaningful")

	denseItems := dense.Words[0].SimilarItems("кот", dense.CharPolicy, dense.Alphabet)
	require.Len(t, denseItems, 1)
	assert.Equal(t, rawItems[0].Key, denseItems[0].Key)
	// ElementsMatch, not Equal: RecompileDense rebuilds the DAWG from a
	// Walk over the original one, which need not visit/emit a homonym's
	// values in the same order ImportFromDir's raw DAWG stored them in —
	// order isn't part of the contract, all values being present is.
	assert.ElementsMatch(t, rawItems[0].Values, denseItems[0].Values,
		"RecompileDense must preserve every payload value of a homonym, not just the first")

	// Sanity: the non-homonym word in the same fixture still round-trips
	// with a single value, same as TestRecompileDense already covers.
	rawKota := raw.Words[0].SimilarItems("кота", raw.CharPolicy, nil)
	denseKota := dense.Words[0].SimilarItems("кота", dense.CharPolicy, dense.Alphabet)
	require.Len(t, rawKota, 1)
	require.Len(t, denseKota, 1)
	assert.Equal(t, rawKota[0].Values, denseKota[0].Values)
}

func TestRecompileDense_MissingDir(t *testing.T) {
	_, err := pymorphy2.RecompileDense(t.TempDir() + "/does-not-exist")
	require.Error(t, err)
}
