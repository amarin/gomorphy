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

func TestBuildPredictionFromMatchesBuildPrediction(t *testing.T) {
	all := func(string) bool { return true }
	d, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{
		{Word: "кот", Lemma: "кот", Tag: "NOUN,nomn"},
		{Word: "кота", Lemma: "кот", Tag: "NOUN,gent"},
		{Word: "мышь", Lemma: "мышь", Tag: "NOUN,nomn"},
		{Word: "мыши", Lemma: "мышь", Tag: "NOUN,gent"},
	})
	require.NoError(t, err)

	var pairs []WordValue
	d.Words[0].Walk(func(w string, vals [][]byte) {
		for _, v := range vals {
			pairs = append(pairs, WordValue{Word: w, Value: binary.BigEndian.Uint32(v[:4])})
		}
	})
	pred, sharded, err := BuildPredictionFrom([][]WordValue{pairs}, d.Paradigms, d.TagSet, all)
	require.NoError(t, err)
	assert.False(t, sharded, "one shard keeps 6-byte values")

	require.NoError(t, BuildPrediction(d, all))
	assert.False(t, d.PredictionSharded)
	assert.Equal(t, d.Prediction[0].Bytes(), pred.Bytes())
}

// twoShardCorpus is a raw two-shard dictionary sharing one TagSet: shard 0
// holds кошка/кошки, shard 1 окно/окна. Each shard's only lemma is its
// paradigm 0, so only the shard number tells their predictions apart.
func twoShardCorpus(t *testing.T) *Dictionary {
	t.Helper()
	a, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,sing,gent"},
	})
	require.NoError(t, err)
	b, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{
		{Word: "окно", Lemma: "окно", Tag: "NOUN,sing,nomn"},
		{Word: "окна", Lemma: "окно", Tag: "NOUN,sing,gent"},
	})
	require.NoError(t, err)
	// b's tag ids must mean the same tags in a's TagSet (same insertion order).
	for f := 0; f < 2; f++ {
		require.Equal(t, b.TagSet.TagName(b.Paradigms[0][0].Tag(f)), a.TagSet.TagName(b.Paradigms[0][0].Tag(f)))
	}
	return NewDictionary("ru", a.TagSet,
		[][]string{a.Suffixes[0], b.Suffixes[0]}, a.Prefixes,
		[][]Paradigm{a.Paradigms[0], b.Paradigms[0]},
		[]*DAWG{a.Words[0], b.Words[0]}, RussianCharPolicy())
}

type shardedPredValue struct{ Count, Para, Form, Shard uint16 }

// shardedPredValues returns the decoded 8-byte payloads for an exact
// prediction suffix key (empty when the key is absent).
func shardedPredValues(t *testing.T, pred *DAWG, key string) []shardedPredValue {
	t.Helper()
	var out []shardedPredValue
	for _, it := range pred.SimilarItems(key, nil, nil) {
		if it.Key != key {
			continue
		}
		for _, v := range it.Values {
			require.Len(t, v, 8)
			out = append(out, shardedPredValue{
				Count: binary.BigEndian.Uint16(v[0:2]),
				Para:  binary.BigEndian.Uint16(v[2:4]),
				Form:  binary.BigEndian.Uint16(v[4:6]),
				Shard: binary.BigEndian.Uint16(v[6:8]),
			})
		}
	}
	return out
}

func TestBuildPredictionSharded(t *testing.T) {
	d := twoShardCorpus(t)
	require.NoError(t, BuildPrediction(d, predProductive))

	require.Len(t, d.Prediction, 1, "one DAWG for prefix 0, shared by all shards")
	assert.True(t, d.PredictionSharded)
	assert.Equal(t, []shardedPredValue{{Count: 1, Para: 0, Form: 1, Shard: 1}},
		shardedPredValues(t, d.Prediction[0], "на"), "«на» comes only from окна (shard 1)")
	assert.Equal(t, []shardedPredValue{{Count: 1, Para: 0, Form: 1, Shard: 0}},
		shardedPredValues(t, d.Prediction[0], "ки"), "«ки» comes only from кошки (shard 0)")
	assert.ElementsMatch(t, []shardedPredValue{
		{Count: 1, Para: 0, Form: 0, Shard: 0}, // кошка
		{Count: 1, Para: 0, Form: 1, Shard: 1}, // окна
	}, shardedPredValues(t, d.Prediction[0], "а"))
}

func TestBuildPredictionShardedDenseMatchesRaw(t *testing.T) {
	raw := twoShardCorpus(t)
	require.NoError(t, BuildPrediction(raw, predProductive))

	dense := twoShardCorpus(t)
	require.NoError(t, RecompileDense(dense))
	require.NotNil(t, dense.Alphabet)
	require.NoError(t, BuildPrediction(dense, predProductive))

	assert.True(t, dense.PredictionSharded)
	assert.Equal(t, raw.Prediction[0].Bytes(), dense.Prediction[0].Bytes(),
		"keys are decoded through the alphabet, so a dense dictionary predicts the same")
}
