package internal

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// engineDict builds a dense single-shard dictionary from entries.
func engineDict(t *testing.T, entries ...BuildEntry) *Dictionary {
	t.Helper()
	d, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	require.NoError(t, RecompileDense(d))
	return d
}

func e(word, lemma, tag string) BuildEntry { return BuildEntry{Word: word, Lemma: lemma, Tag: tag} }

func TestDecideOverlaysFold(t *testing.T) {
	base := engineDict(t, e("кот", "кот", "B"))
	o1 := engineDict(t, e("кот", "кот", "O1"), e("пёс", "пёс", "O1"))
	o2 := engineDict(t, e("кот", "кот", "O2"), e("пёс", "пёс", "O2"))
	overlays := []*Dictionary{o1, o2}

	add, err := decideOverlays(base, overlays, MergeAdd)
	require.NoError(t, err)
	assert.NotContains(t, add, "кот", "add: a base word is never taken")
	require.Contains(t, add, "пёс")
	assert.Equal(t, 0, add["пёс"].overlay, "add: the first overlay to supply a new word wins")

	rep, err := decideOverlays(base, overlays, MergeReplace)
	require.NoError(t, err)
	assert.Equal(t, 1, rep["кот"].overlay, "replace: the last overlay wins over the base")
	assert.Equal(t, 1, rep["пёс"].overlay, "replace: the last overlay wins over an earlier overlay")
}

func TestHasWordExactKey(t *testing.T) {
	d := engineDict(t, e("ёж", "ёж", "N"))
	assert.True(t, hasWord(d, "ёж"))
	assert.False(t, hasWord(d, "еж"), "no CharPolicy substitution")
	assert.False(t, hasWord(d, "ё"), "a key prefix is not a word")
	assert.False(t, hasWord(d, "wifi"), "a word the alphabet can't encode is absent")
}

func TestWordReadingsDecodesDenseKeys(t *testing.T) {
	d := engineDict(t, e("кот", "кот", "N"), e("кота", "кот", "G"))
	rs, err := wordReadings(d)
	require.NoError(t, err)
	assert.Len(t, rs["кот"], 1)
	assert.Len(t, rs["кота"], 1)
	assert.Equal(t, uint16(1), rs["кота"][0].form)
}

func TestPlaceRemapsIntoBaseIDSpace(t *testing.T) {
	base := engineDict(t, e("кот", "кот", "N,nomn"), e("кота", "кот", "N,gent"))
	baseTags := append([]string(nil), base.TagSet.Tags...)
	baseSuffixes := append([]string(nil), base.Suffixes[0]...)
	nBase := len(base.Paradigms[0])

	o := engineDict(t, e("кит", "кит", "N,nomn"), e("кита", "кит", "N,gent"), e("ура", "ура", "INTJ"))
	rs, err := wordReadings(o)
	require.NoError(t, err)

	m := newMerger(base)
	loc, err := m.place(0, o, rs["кита"][0])
	require.NoError(t, err)
	assert.Equal(t, paraLoc{shard: 0, para: 0}, loc, "a paradigm shaped like base paradigm 0 dedups onto it")
	assert.Len(t, m.shards[0].paradigms, nBase)

	again, err := m.place(0, o, rs["кит"][0])
	require.NoError(t, err)
	assert.Equal(t, loc, again, "same overlay paradigm → cached location")

	loc2, err := m.place(0, o, rs["ура"][0])
	require.NoError(t, err)
	assert.Equal(t, paraLoc{shard: 0, para: uint16(nBase)}, loc2, "a new shape is appended")

	assert.Equal(t, baseTags, m.tagSet.Tags[:len(baseTags)], "base tag ids are stable")
	assert.Contains(t, m.tagSet.Tags, "INTJ")
	assert.Equal(t, baseSuffixes, m.shards[0].suffixes[:len(baseSuffixes)], "base suffix ids are stable")
	assert.Equal(t, baseTags, base.TagSet.Tags, "the base TagSet is not mutated")
	assert.Equal(t, baseSuffixes, base.Suffixes[0], "the base suffixes are not mutated")
}

// TestPlaceRemapsOverlayOnlyIDs uses an overlay paradigm whose tag/suffix
// names are entirely new to the base (unlike TestPlaceRemapsIntoBaseIDSpace's
// N,nomn/N,gent/"а" shapes, which the builder happens to assign identical
// ids to in both base and overlay — a place() that copied overlay ids
// verbatim would pass that test's assertions too). Here the overlay's own
// local ids and the merged output ids are forced apart, so this test can
// only pass if place() actually translates suffix/tag/prefix ids via their
// names rather than copying them through.
func TestPlaceRemapsOverlayOnlyIDs(t *testing.T) {
	base := engineDict(t, e("кот", "кот", "N,nomn"), e("кота", "кот", "N,gent"))

	// "два"/"двух": stem "дв", suffixes "а" (id0) and "ух" (id1); tags
	// "NUM,nomn" (id0) and "NUM,gen" (id1) — none of these names exist in
	// base, and even "а" (base's own suffix id1) lands at a different
	// local id here (0), so every form's ids diverge from the output.
	o := engineDict(t, e("два", "два", "NUM,nomn"), e("двух", "два", "NUM,gen"))
	rs, err := wordReadings(o)
	require.NoError(t, err)
	r := rs["два"][0]

	m := newMerger(base)
	loc, err := m.place(0, o, r)
	require.NoError(t, err)

	op := o.Paradigms[r.shard][r.para]
	outPara := m.shards[loc.shard].paradigms[loc.para]
	require.Equal(t, op.Len(), outPara.Len())

	diverged := false
	for i := 0; i < op.Len(); i++ {
		wantSuffix := stringAt(o.Suffixes[r.shard], op.Suffix(i))
		gotSuffix := stringAt(m.shards[loc.shard].suffixes, outPara.Suffix(i))
		assert.Equal(t, wantSuffix, gotSuffix, "form %d suffix text", i)

		wantTag := o.TagSet.TagName(op.Tag(i))
		gotTag := m.tagSet.TagName(outPara.Tag(i))
		assert.Equal(t, wantTag, gotTag, "form %d tag name", i)

		wantPrefix := stringAt(o.Prefixes, op.Prefix(i))
		gotPrefix := stringAt(m.prefixes, outPara.Prefix(i))
		assert.Equal(t, wantPrefix, gotPrefix, "form %d prefix text", i)

		if op.Suffix(i) != outPara.Suffix(i) || op.Tag(i) != outPara.Tag(i) {
			diverged = true
		}
	}
	assert.True(t, diverged, "remapped ids must differ from the overlay's own local ids for at least one form — otherwise this test can't tell a real remap from a verbatim copy")
}

func TestPlaceOpensNewShardOnSuffixOverflow(t *testing.T) {
	old := mergeSuffixLimit
	mergeSuffixLimit = 2
	t.Cleanup(func() { mergeSuffixLimit = old })

	base := engineDict(t, e("кот", "кот", "N"), e("кота", "кот", "G"))   // suffixes "", "а"
	o := engineDict(t, e("стол", "стол", "N"), e("столом", "стол", "I")) // needs new suffix "ом"
	rs, err := wordReadings(o)
	require.NoError(t, err)

	m := newMerger(base)
	loc, err := m.place(0, o, rs["столом"][0])
	require.NoError(t, err)
	assert.Equal(t, 1, loc.shard)
	assert.Len(t, m.shards, 2)
	assert.Nil(t, m.shards[1].base, "an appended shard has no base DAWG")
}

func allProductive(string) bool { return true }

// twoShardBase glues two single-shard builds into one 2-shard dense
// dictionary. Both builds register tags "N" then "G", so shard 1's tag
// ids are valid under shard 0's TagSet.
func twoShardBase(t *testing.T) *Dictionary {
	t.Helper()
	a, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{e("кот", "кот", "N"), e("кота", "кот", "G")})
	require.NoError(t, err)
	b, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{e("мышь", "мышь", "N"), e("мыши", "мышь", "G")})
	require.NoError(t, err)
	d := NewDictionary("ru", a.TagSet,
		[][]string{a.Suffixes[0], b.Suffixes[0]}, a.Prefixes,
		[][]Paradigm{a.Paradigms[0], b.Paradigms[0]},
		[]*DAWG{a.Words[0], b.Words[0]}, RussianCharPolicy())
	require.NoError(t, RecompileDense(d))
	return d
}

// readingsOf returns word's raw values across all shards (exact key).
func readingsOf(d *Dictionary, word string) [][]byte {
	var out [][]byte
	for _, w := range d.Words {
		for _, it := range w.SimilarItems(word, nil, d.Alphabet) {
			if it.Key == word {
				out = append(out, it.Values...)
			}
		}
	}
	return out
}

// shardValue is one raw words.dawg reading located to the shard it came
// from, so its (para, form) can be resolved against that shard's
// out.Paradigms.
type shardValue struct {
	shard int
	value []byte
}

// readingsWithShard is readingsOf, but keeping track of which shard each
// value came from (readingsOf alone can't tell, since a replaced word may
// move to a different shard than the base held it in).
func readingsWithShard(d *Dictionary, word string) []shardValue {
	var out []shardValue
	for si, w := range d.Words {
		for _, it := range w.SimilarItems(word, nil, d.Alphabet) {
			if it.Key == word {
				for _, v := range it.Values {
					out = append(out, shardValue{shard: si, value: v})
				}
			}
		}
	}
	return out
}

// tagOf decodes a raw words.dawg value (para<<16|form) against out's
// paradigms/tag set for the shard it was found in.
func tagOf(t *testing.T, out *Dictionary, sv shardValue) string {
	t.Helper()
	require.GreaterOrEqual(t, len(sv.value), 4)
	val := binary.BigEndian.Uint32(sv.value[:4])
	para, form := uint16(val>>16), uint16(val)
	require.Less(t, int(para), len(out.Paradigms[sv.shard]))
	p := out.Paradigms[sv.shard][para]
	require.Less(t, int(form), p.Len())
	return out.TagSet.TagName(p.Tag(int(form)))
}

func TestMergeDictionariesZeroShardBaseRebuildPrediction(t *testing.T) {
	base := &Dictionary{Language: "ru", TagSet: NewTagSet("x")}

	var out *Dictionary
	var err error
	assert.NotPanics(t, func() {
		out, err = MergeDictionaries(base, nil, MergeOptions{RebuildPrediction: true, Productive: allProductive})
	}, "a zero-shard base with RebuildPrediction must not panic on base.Words[0]")

	if err == nil {
		require.NotNil(t, out)
		require.Len(t, out.Words, 1)
		require.Len(t, out.Prediction, 1)
	}
}

func TestMergeDictionariesReplaceWordInNonTargetShard(t *testing.T) {
	base := twoShardBase(t)
	over := engineDict(t, e("кота", "кота", "X"))
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeReplace})
	require.NoError(t, err)

	vals := readingsWithShard(out, "кота")
	require.Len(t, vals, 1, "the base reading of a replaced word must be gone")
	assert.Equal(t, "X", tagOf(t, out, vals[0]), "the surviving reading is the overlay's")

	assert.NotEmpty(t, readingsOf(out, "кот"), "an untouched word in the same shard survives")
	assert.NotEmpty(t, readingsOf(out, "мыши"), "the target shard's own word survives")
	assert.NotEqual(t, base.Words[0].Bytes(), out.Words[0].Bytes(), "shard 0 (where the replaced word lived) was rebuilt")
}

func TestMergeDictionariesAlphabetChangeCleanShard(t *testing.T) {
	base := twoShardBase(t)
	over := engineDict(t, e("wifi", "wifi", "N"))
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.NotSame(t, base.Alphabet, out.Alphabet, "the alphabet must be rebuilt to cover \"wifi\"")

	want := map[string]string{"кот": "N", "кота": "G", "мышь": "N", "мыши": "G", "wifi": "N"}
	for word, wantTag := range want {
		vals := readingsWithShard(out, word)
		require.NotEmpty(t, vals, word)
		for _, sv := range vals {
			assert.Equal(t, wantTag, tagOf(t, out, sv), word)
		}
	}
}

func TestMergeDictionariesReusesCleanShard(t *testing.T) {
	base := twoShardBase(t)
	over := engineDict(t, e("шок", "шок", "N")) // letters already in the base alphabet
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)

	require.Len(t, out.Words, 2)
	assert.Same(t, base.Alphabet, out.Alphabet, "alphabet reused when it covers the overlay")
	assert.Equal(t, base.Words[0].Bytes(), out.Words[0].Bytes(), "clean shard 0 is reused byte-for-byte")
	assert.NotSame(t, base.Words[0], out.Words[0], "…but cloned, not aliased")
	assert.NotEmpty(t, readingsOf(out, "шок"), "overlay word lands in the target (last) shard")
	assert.NotEmpty(t, readingsOf(out, "мыши"))
}

func TestMergeDictionariesExtendsAlphabet(t *testing.T) {
	base := engineDict(t, e("кот", "кот", "N"))
	over := engineDict(t, e("wifi", "wifi", "N"))
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.NotSame(t, base.Alphabet, out.Alphabet)
	assert.NotEmpty(t, readingsOf(out, "wifi"))
	assert.NotEmpty(t, readingsOf(out, "кот"))
}

func TestMergeDictionariesReplaceAcrossShards(t *testing.T) {
	base := twoShardBase(t)
	over := engineDict(t, e("мыши", "мыши", "X"))
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeReplace})
	require.NoError(t, err)
	vals := readingsOf(out, "мыши")
	require.Len(t, vals, 1, "all base readings of a replaced word are dropped")
	assert.NotEmpty(t, readingsOf(out, "кот"))
	assert.NotEmpty(t, readingsOf(out, "мышь"))
}

func withProbability(t *testing.T, d *Dictionary, kv map[string]uint32) *Dictionary {
	t.Helper()
	keys, vals := sortedKV(kv)
	p, err := BuildIntDAWG(keys, vals)
	require.NoError(t, err)
	d.Probability = p
	return d
}

func TestMergeDictionariesProbability(t *testing.T) {
	newBase := func() *Dictionary {
		return withProbability(t, engineDict(t, e("кот", "кот", "N"), e("кота", "кот", "G")),
			map[string]uint32{"кот:N": 500, "кота:G": 300})
	}

	// Fast path: nothing removed, overlay without probability → verbatim copy.
	base := newBase()
	out, err := MergeDictionaries(base, []*Dictionary{engineDict(t, e("шок", "шок", "N"))}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.Equal(t, base.Probability.Bytes(), out.Probability.Bytes())
	assert.NotSame(t, base.Probability, out.Probability)

	// Overlay probability is carried for the words it wins.
	over := withProbability(t, engineDict(t, e("шок", "шок", "N")), map[string]uint32{"шок:N": 900})
	out, err = MergeDictionaries(newBase(), []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.Equal(t, uint32(500), out.Probability.Find("кот:N"))
	assert.Equal(t, uint32(900), out.Probability.Find("шок:N"))

	// A replaced word loses its base probability entries.
	out, err = MergeDictionaries(newBase(), []*Dictionary{engineDict(t, e("кот", "кот", "V"))}, MergeOptions{Mode: MergeReplace})
	require.NoError(t, err)
	assert.Equal(t, uint32(0), out.Probability.Find("кот:N"))
	assert.Equal(t, uint32(300), out.Probability.Find("кота:G"))
}

func TestMergeDictionariesPrediction(t *testing.T) {
	base := engineDictRaw(t, e("кот", "кот", "N"), e("кота", "кот", "G"))
	require.NoError(t, BuildPrediction(base, allProductive))
	require.NoError(t, RecompileDense(base))
	over := engineDict(t, e("шок", "шок", "N"))

	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	require.Len(t, out.Prediction, 1)
	assert.Equal(t, base.Prediction[0].Bytes(), out.Prediction[0].Bytes(), "prediction carried verbatim by default")
	assert.Empty(t, out.Prediction[0].SimilarItems("ок", nil, nil), "overlay words don't feed carried prediction")

	out, err = MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd, RebuildPrediction: true, Productive: allProductive})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Prediction[0].SimilarItems("ок", nil, nil), "rebuilt prediction covers overlay words")

	_, err = MergeDictionaries(twoShardBase(t), []*Dictionary{over}, MergeOptions{Mode: MergeAdd, RebuildPrediction: true, Productive: allProductive})
	assert.ErrorIs(t, err, ErrPredictionSharded)
}

// engineDictRaw is engineDict without the dense recompile.
func engineDictRaw(t *testing.T, entries ...BuildEntry) *Dictionary {
	t.Helper()
	d, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	return d
}
