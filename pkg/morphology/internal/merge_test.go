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
