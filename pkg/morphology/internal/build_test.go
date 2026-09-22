package internal

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildLookup resolves word in shard 0's DAWG and returns the decoded
// (paraID, form) payload value — the same decoding parse.go's reading
// applies to words.dawg payload entries.
func buildLookup(t *testing.T, dict *Dictionary, word string) (para, form uint16) {
	t.Helper()
	items := dict.Words[0].SimilarItems(word, nil, nil)
	require.Len(t, items, 1)
	assert.Equal(t, word, items[0].Key)
	require.Len(t, items[0].Values, 1)
	v := items[0].Values[0]
	require.Len(t, v, 4)
	return binary.BigEndian.Uint16(v[:2]), binary.BigEndian.Uint16(v[2:4])
}

func TestBuildDictionaryFromEntriesCorpus(t *testing.T) {
	entries := []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "кошке", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,datv"},
		{Word: "кошкой", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,ablt"},
		{Word: "дело", Lemma: "дело", Tag: "NOUN,inan,neut,sing,nomn"},
		{Word: "делом", Lemma: "дело", Tag: "NOUN,inan,neut,sing,ablt"},
	}

	dict, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	require.NotNil(t, dict)

	// Defaults from zero-value options.
	assert.Equal(t, "ru", dict.Language)
	require.NotNil(t, dict.CharPolicy)
	assert.Equal(t, RussianCharPolicy().Substitutions, dict.CharPolicy.Substitutions)
	require.NotNil(t, dict.TagSet)
	assert.Equal(t, "builder", dict.TagSet.Name)

	// Raw dictionary contract: no Alphabet/Prediction/Info.
	assert.Nil(t, dict.Alphabet)
	assert.Nil(t, dict.Prediction)
	assert.Nil(t, dict.Info)

	// One shard, shared prefixes = [""].
	require.Len(t, dict.Suffixes, 1)
	require.Len(t, dict.Paradigms, 1)
	require.Len(t, dict.Words, 1)
	assert.Equal(t, []string{""}, dict.Prefixes)

	assert.Equal(t, []string{"а", "и", "е", "ой", "", "м"}, dict.Suffixes[0])

	para, form := buildLookup(t, dict, "кошка")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(0), form)
	para, form = buildLookup(t, dict, "кошки")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(1), form)
	para, form = buildLookup(t, dict, "кошке")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(2), form)
	para, form = buildLookup(t, dict, "кошкой")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(3), form)

	para, form = buildLookup(t, dict, "дело")
	assert.Equal(t, uint16(1), para)
	assert.Equal(t, uint16(0), form)
	para, form = buildLookup(t, dict, "делом")
	assert.Equal(t, uint16(1), para)
	assert.Equal(t, uint16(1), form)

	// The paradigm's form 0 suffix describes the lemma itself.
	assert.Equal(t, uint16(0), dict.Paradigms[0][0].Suffix(0))
}

func TestBuildDictionaryFromEntriesSynthesizesLemmaForm0(t *testing.T) {
	entries := []BuildEntry{
		{Word: "иду", Lemma: "идти", Tag: "VERB,imperf,intr,1per,sing,pres,indc"},
		{Word: "идёшь", Lemma: "идти", Tag: "VERB,imperf,intr,2per,sing,pres,indc"},
	}

	dict, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)

	// The lemma text is never a wordform, so form 0 must be synthesized
	// {Word: "идти", Tag: ""}; every other form's (paraID, form) must slot
	// in after it.
	para, form := buildLookup(t, dict, "идти")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(0), form)
	para, form = buildLookup(t, dict, "иду")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(1), form)
	para, form = buildLookup(t, dict, "идёшь")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(2), form)

	require.Len(t, dict.Paradigms[0], 1)
	p := dict.Paradigms[0][0]
	assert.Equal(t, 3, p.Len())
	// Form 0 carries the empty tag that was synthesized for it.
	assert.Equal(t, "", dict.TagSet.TagName(p.Tag(0)))
}

func TestBuildDictionaryFromEntriesAutoLemma(t *testing.T) {
	entries := []BuildEntry{
		{Word: "кот", Lemma: "", Tag: "NOUN,anim,masc,sing,nomn"},
		{Word: "кота", Lemma: "", Tag: "NOUN,anim,masc,sing,gent"},
	}

	dict, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)

	// Empty lemma falls back to the word itself, so each entry is its own
	// single-form lemma group.
	para, form := buildLookup(t, dict, "кот")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(0), form)
	para, form = buildLookup(t, dict, "кота")
	assert.Equal(t, uint16(1), para)
	assert.Equal(t, uint16(0), form)
}

func TestBuildDictionaryFromEntriesParadigmDedup(t *testing.T) {
	entries := []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "кошке", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,datv"},
		{Word: "кошкой", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,ablt"},
		{Word: "мышка", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "мышки", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "мышке", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,datv"},
		{Word: "мышкой", Lemma: "мышка", Tag: "NOUN,anim,femn,sing,ablt"},
	}

	dict, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)

	// Both lemmas share the same (suffixID, tagID) sequences despite
	// different stems, so one paradigm is shared between them.
	require.Len(t, dict.Paradigms[0], 1)
	para, form := buildLookup(t, dict, "кошка")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(0), form)
	para, form = buildLookup(t, dict, "мышка")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(0), form)
	para, form = buildLookup(t, dict, "мышкой")
	assert.Equal(t, uint16(0), para)
	assert.Equal(t, uint16(3), form)
}

func TestBuildDictionaryFromEntriesDeterministic(t *testing.T) {
	entries := []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "кошке", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,datv"},
		{Word: "кошкой", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,ablt"},
		{Word: "дело", Lemma: "дело", Tag: "NOUN,inan,neut,sing,nomn"},
		{Word: "делом", Lemma: "дело", Tag: "NOUN,inan,neut,sing,ablt"},
	}

	a, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	b, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)

	assert.Equal(t, EncodeStrings(a.Suffixes[0]), EncodeStrings(b.Suffixes[0]))
	assert.Equal(t, EncodeParadigms(a.Paradigms[0]), EncodeParadigms(b.Paradigms[0]))
	assert.Equal(t, a.Words[0].Bytes(), b.Words[0].Bytes())
}

func TestBuildDictionaryFromEntriesEmptyCorpus(t *testing.T) {
	dict, err := BuildDictionaryFromEntries(BuildOptions{}, nil)
	require.NoError(t, err)
	require.NotNil(t, dict)

	// One empty shard: an empty DAWG that resolves nothing.
	require.Len(t, dict.Suffixes, 1)
	assert.Empty(t, dict.Suffixes[0])
	require.Len(t, dict.Paradigms, 1)
	assert.Empty(t, dict.Paradigms[0])
	require.Len(t, dict.Words, 1)
	assert.NotNil(t, dict.Words[0])
	assert.Equal(t, []string{""}, dict.Prefixes)
}

func TestBuildDictionaryFromEntriesParadigmLimit(t *testing.T) {
	old := paradigmLimit
	paradigmLimit = 2
	defer func() { paradigmLimit = old }()

	entries := []BuildEntry{
		{Word: "кот", Lemma: "кот", Tag: "T1"},
		{Word: "пёс", Lemma: "пёс", Tag: "T2"},
		{Word: "кит", Lemma: "кит", Tag: "T3"},
	}

	_, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unique paradigms"),
		"error %q should mention the unique-paradigms cap", err)
}

func TestBuildDictionaryFromEntriesProgress(t *testing.T) {
	entries := []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "кошке", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,datv"},
		{Word: "кошкой", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,ablt"},
		{Word: "дело", Lemma: "дело", Tag: "NOUN,inan,neut,sing,nomn"},
		{Word: "делом", Lemma: "дело", Tag: "NOUN,inan,neut,sing,ablt"},
	}

	var calls [][2]int
	_, err := BuildDictionaryFromEntries(BuildOptions{
		Progress: func(processed, total int) {
			calls = append(calls, [2]int{processed, total})
		},
	}, entries)
	require.NoError(t, err)

	require.NotEmpty(t, calls)
	for _, c := range calls {
		assert.Equal(t, len(entries), c[1],
			"total is always the combined entry count, never a 0 marker")
		assert.LessOrEqual(t, c[0], c[1])
	}
}
