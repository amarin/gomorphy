package internal

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// predictionCorpus builds the small multi-lemma raw dictionary the
// prediction tests run against. Paradigm ids assigned by first appearance:
//
//	0 — кошка / мышка (dedup: identical (suffix, tag) sequences)
//	1 — дело
//	2 — окно (distinct tags → distinct paradigm)
//	3 — и (CONJ, filtered out by the non-productive predicate)
func predictionCorpus(t *testing.T) *Dictionary {
	t.Helper()
	entries := []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "кошке", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,datv"},
		{Word: "кошкой", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,ablt"},
		{Word: "мышка", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "мышки", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "мышке", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,datv"},
		{Word: "мышкой", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,ablt"},
		{Word: "дело", Lemma: "дело", Tag: "NOUN,inan,neut,sing,nomn"},
		{Word: "делом", Lemma: "дело", Tag: "NOUN,inan,neut,sing,ablt"},
		{Word: "окно", Lemma: "окно", Tag: "NOUN,inan,neut,sing,accs"},
		{Word: "окна", Lemma: "окно", Tag: "NOUN,inan,neut,sing,gent"},
		{Word: "окну", Lemma: "окно", Tag: "NOUN,inan,neut,sing,datv"},
		{Word: "и", Lemma: "и", Tag: "CONJ"},
		{Word: "их", Lemma: "и", Tag: "CONJ"},
	}
	d, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)
	return d
}

// predProductive is the test predicate: everything is productive except the
// CONJ tag constructed in predictionCorpus.
func predProductive(tag string) bool { return tag != "CONJ" }

// decodePredValue decodes a prediction payload: count(BE16) + para(BE16) +
// form(BE16) — the exact shape parse.go's predictForPrefix consumes.
func decodePredValue(t *testing.T, v []byte) (count, para, form uint16) {
	t.Helper()
	require.Len(t, v, 6)
	return binary.BigEndian.Uint16(v[:2]),
		binary.BigEndian.Uint16(v[2:4]),
		binary.BigEndian.Uint16(v[4:6])
}

// predValues returns the decoded payload triples for an exact prediction
// suffix key (empty when the key is absent).
func predValues(t *testing.T, pred *DAWG, key string) []struct{ Count, Para, Form uint16 } {
	t.Helper()
	items := pred.SimilarItems(key, RussianCharPolicy(), nil)
	if len(items) == 0 {
		return nil
	}
	require.Len(t, items, 1)
	require.Equal(t, key, items[0].Key)
	out := make([]struct{ Count, Para, Form uint16 }, 0, len(items[0].Values))
	for _, v := range items[0].Values {
		c, p, f := decodePredValue(t, v)
		out = append(out, struct{ Count, Para, Form uint16 }{c, p, f})
	}
	return out
}

func TestBuildPrediction(t *testing.T) {
	d := predictionCorpus(t)

	require.NoError(t, BuildPrediction(d, predProductive))

	require.Len(t, d.Prediction, 1, "one prediction DAWG for the single prefix (\"\")")
	pred := d.Prediction[0]
	require.NotNil(t, pred)

	// "а" is a multi-value key: (para 0, form 0) attested twice (кошка,
	// мышка) and (para 2, form 1) once (окна).
	values := predValues(t, pred, "а")
	require.Len(t, values, 2, "key %q must carry two payload leaves", "а")
	byTriple := map[[2]uint16]uint16{}
	for _, v := range values {
		byTriple[[2]uint16{v.Para, v.Form}] = v.Count
	}
	assert.Equal(t, uint16(2), byTriple[[2]uint16{0, 0}], "suffix twice present must accumulate count 2")
	assert.Equal(t, uint16(1), byTriple[[2]uint16{2, 1}])

	// Count accumulation also for "и" (кошки + мышки → 2).
	values = predValues(t, pred, "и")
	require.Len(t, values, 1)
	assert.Equal(t, uint16(2), values[0].Count)
	assert.Equal(t, uint16(0), values[0].Para)
	assert.Equal(t, uint16(1), values[0].Form)

	// "о" is multi-value as well: дело (para 1, form 0) and окно (para 2, form 0).
	values = predValues(t, pred, "о")
	require.Len(t, values, 2)
	byTriple = map[[2]uint16]uint16{}
	for _, v := range values {
		byTriple[[2]uint16{v.Para, v.Form}] = v.Count
	}
	assert.Contains(t, byTriple, [2]uint16{1, 0})
	assert.Contains(t, byTriple, [2]uint16{2, 0})

	// Non-productive readings are excluded. "их" exists in the corpus only
	// as a CONJ wordform, so no suffix key survives from it.
	assert.Empty(t, predValues(t, pred, "их"), "non-productive CONJ readings must not enter prediction")

	// "и" is present productively (кошки/мышки, para 0 form 1) but must not
	// carry the CONJ paradigm (para 3).
	values = predValues(t, pred, "и")
	for _, v := range values {
		assert.NotEqual(t, uint16(3), v.Para, "CONJ paradigm must be filtered out")
	}

	// A 5-rune word produces suffix keys of lengths 1..5 (кошка → а, ка,
	// шка, ошка, кошка), each resolving to (para 0, form 0).
	for _, key := range []string{"а", "ка", "шка", "ошка", "кошка"} {
		values := predValues(t, pred, key)
		require.NotEmpty(t, values, "suffix key %q (len %d) must be present", key, len([]rune(key)))
		found := false
		for _, v := range values {
			if v.Para == 0 && v.Form == 0 {
				found = true
			}
		}
		assert.True(t, found, "suffix key %q must carry (para 0, form 0)", key)
	}

	// Rune-correctness: a 6-rune wordform's full text is never a suffix key
	// (max suffix length is 5).
	assert.Empty(t, predValues(t, pred, "кошкой"), "6-rune word text must not appear as a suffix key")
}

func TestBuildPredictionNoOpWhenSharded(t *testing.T) {
	emptyDAWG := func() *DAWG {
		d, err := BuildDAWG(nil)
		require.NoError(t, err)
		return d
	}
	dict := NewDictionary(
		"ru",
		NewTagSet("test"),
		[][]string{{}, {}},
		[]string{""},
		[][]Paradigm{{}, {}},
		[]*DAWG{emptyDAWG(), emptyDAWG()},
		nil,
	)
	require.Len(t, dict.Words, 2)

	require.NoError(t, BuildPrediction(dict, predProductive))
	assert.Nil(t, dict.Prediction, "prediction is built only for unsharded dictionaries")
}
