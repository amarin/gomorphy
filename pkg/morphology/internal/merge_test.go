package internal

import (
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
