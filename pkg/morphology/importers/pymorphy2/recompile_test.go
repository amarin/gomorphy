package pymorphy2_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeHomonymFixtureDir — как makeFixtureDir (import_test.go), но "кот"
// несёт 2 разных payload-значения (омоним), а не 1. Локальная копия, а не
// расширение makeFixtureDir: другие тесты в import_test.go (например,
// TestImportFromDir) утверждают len(items[0].Values) == 1 для "кот", так
// что менять её payload задним числом сломало бы их сборку.
//
// Нужна для проверки fan-out цикла `for _, v := range vals` в
// RecompileDense.reading — с одним значением на слово (makeFixtureDir)
// цикл всегда делает одну итерацию, и композиция Walk + цикл +
// BuildDAWGWithValues, теряющая лишние значения омонима, осталась бы
// незамеченной. См. TestDAWGWalk (pkg/morphology/internal/dawg_test.go)
// для того же приёма на уровне DAWG.Walk напрямую.
func makeHomonymFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeParadigms(t, dir, [][]uint16{
		{10, 20, 0, 1, 0, 0}, // 2 формы: суффиксы[10,20], теги[0,1], префиксы[0,0]
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

// TestRecompileDense_Homonym проверяет, что RecompileDense не теряет
// значения для слова с несколькими payload-значениями (омонима). Отдельно
// от TestRecompileDense, чья фикстура (makeFixtureDir) даёт каждому слову
// ровно одно значение и потому не задействует fan-out `for _, v := range
// vals` в RecompileDense по-настоящему (цикл на 1 итерации неотличим от
// его отсутствия).
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
