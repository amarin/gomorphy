package opencorpora_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/opencorpora"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecompileDenseMatchesRaw confirms internal.RecompileDense (the
// source-agnostic dense-alphabet rebuild pymorphy2.RecompileDense also
// delegates to) gives byte-for-byte the same SimilarItems results on an
// OpenCorpora-imported dictionary as before the recompile — the same
// invariant pymorphy2's own recompile tests already establish for its
// own importer.
func TestRecompileDenseMatchesRaw(t *testing.T) {
	raw, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	dense, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.NoError(t, internal.RecompileDense(dense))

	require.NotNil(t, dense.Alphabet)
	assert.Equal(t, "dense-1", dense.Alphabet.Name())
	require.Equal(t, len(raw.Words), len(dense.Words))

	for _, word := range []string{"кот", "кота", "мышь", "мыши", "дом"} {
		for shard := range raw.Words {
			rawItems := raw.Words[shard].SimilarItems(word, raw.CharPolicy, nil)
			denseItems := dense.Words[shard].SimilarItems(word, dense.CharPolicy, dense.Alphabet)
			require.Len(t, denseItems, len(rawItems), "word %q shard %d", word, shard)
			for i := range rawItems {
				assert.Equal(t, rawItems[i].Key, denseItems[i].Key, "word %q shard %d item %d", word, shard, i)
				assert.Equal(t, rawItems[i].Values, denseItems[i].Values, "word %q shard %d item %d", word, shard, i)
			}
		}
	}

	// Everything except Words must be copied through unchanged.
	assert.Equal(t, raw.Suffixes, dense.Suffixes)
	assert.Equal(t, raw.Prefixes, dense.Prefixes)
	assert.Equal(t, raw.Paradigms, dense.Paradigms)
	assert.Equal(t, raw.TagSet, dense.TagSet)
}

// TestRecompileDenseAcrossRealShards forces a real multi-shard import
// (the same 65536-unique-suffix technique as
// TestImportFromXMLShardsOnSuffixOverflow) and confirms
// internal.RecompileDense correctly rebuilds every shard under one
// alphabet shared across all of them ("Agreed design decisions", item 1,
// docs/en/implementation/pymorphy2-dense-alphabet.md) — not a fresh,
// mutually-incompatible alphabet per shard.
func TestRecompileDenseAcrossRealShards(t *testing.T) {
	const uniqueSuffixes = 1 << 16

	var xml strings.Builder
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?><dictionary><lemmata>`)
	for i := 0; i < uniqueSuffixes; i++ {
		fmt.Fprintf(&xml, `<lemma id="%d"><l t="слово"/><f t="слово"/><f t="слово%06d"/></lemma>`, i, i)
	}
	xml.WriteString(`</lemmata></dictionary>`)

	d, err := opencorpora.CompileFromXML(strings.NewReader(xml.String()), nil)
	require.NoError(t, err)
	require.Greater(t, len(d.Words), 1, "fixture must actually shard for this test to be meaningful")
	shardCountBefore := len(d.Words)

	require.NoError(t, internal.RecompileDense(d))

	require.NotNil(t, d.Alphabet)
	require.Len(t, d.Words, shardCountBefore, "RecompileDense must not change the number of shards")

	found := 0
	const sampleStride = 997
	for i := 0; i < uniqueSuffixes; i += sampleStride {
		word := fmt.Sprintf("слово%06d", i)
		for _, dawg := range d.Words {
			if len(dawg.SimilarItems(word, d.CharPolicy, d.Alphabet)) > 0 {
				found++
				break
			}
		}
	}
	assert.Equal(t, (uniqueSuffixes+sampleStride-1)/sampleStride, found,
		"every sampled sharded wordform must still be findable in some shard after the dense recompile")
}
