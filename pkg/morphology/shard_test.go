package morphology_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTwoShardDict compiles a synthetic OpenCorpora dict.xml with exactly
// 65536 unique suffixes, forcing FillOnDemand to split it into 2 shards —
// same technique as opencorpora.TestImportFromXMLShardsOnSuffixOverflow,
// but through the public morphology.CompileFromXML entry point so the
// resulting *Dictionary can be exercised with Parse/SaveTo/Lemma/Fuzzy
// like any other dictionary.
func buildTwoShardDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	const uniqueSuffixes = 1 << 16

	var xml strings.Builder
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?><dictionary><lemmata>`)
	for i := 0; i < uniqueSuffixes; i++ {
		fmt.Fprintf(&xml, `<lemma id="%d"><l t="слово"/><f t="слово"/><f t="слово%06d"/></lemma>`, i, i)
	}
	xml.WriteString(`</lemmata></dictionary>`)

	d, err := morphology.CompileFromXML(strings.NewReader(xml.String()), nil)
	require.NoError(t, err)
	return d
}

// TestShardedDictionaryPublicAPIRoundtrip exercises the sharded dictionary
// through the same path a real caller uses: Parse, then SaveTo/Open
// roundtrip, verifying both an early-shard and a late-shard (forced into
// shard 1 by the FillOnDemand boundary) wordform resolve before and after.
func TestShardedDictionaryPublicAPIRoundtrip(t *testing.T) {
	d := buildTwoShardDict(t)

	// "слово000010" lands in shard 0 (well before the boundary); shard 0
	// holds lemmas 0..65534 and shard 1 holds only lemma 65535, per the
	// FillOnDemand boundary trace in TestImportFromXMLShardsOnSuffixOverflow.
	early := "слово000010"
	late := "слово065535"

	rEarly := d.Parse(early)
	require.NotEmpty(t, rEarly, "early wordform must resolve")
	rLate := d.Parse(late)
	require.NotEmpty(t, rLate, "late wordform (forced into shard 1) must resolve")

	sawNonZeroShard := false
	for _, r := range append(append([]morphology.Reading{}, rEarly...), rLate...) {
		if r.Shard != 0 {
			sawNonZeroShard = true
		}
	}
	assert.True(t, sawNonZeroShard, "expected at least one reading from a non-zero shard — this dictionary must genuinely use more than shard 0")

	path := t.TempDir() + "/sharded.dat"
	require.NoError(t, d.SaveTo(path))

	opened, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { _ = opened.Close() }()

	for _, word := range []string{early, late} {
		before := d.Parse(word)
		after := opened.Parse(word)
		require.NotEmpty(t, after, "word %q must resolve after SaveTo/Open roundtrip", word)
		assert.Equal(t, len(before), len(after), "reading count for %q must survive roundtrip", word)
	}
}

// TestShardedDictionaryConcurrentParse exercises the goroutine fan-out in
// exact()/fuzzyWalk() with actual N>1 shards under the race detector — the
// first test in this repo to do so (every other fixture is single-shard).
func TestShardedDictionaryConcurrentParse(t *testing.T) {
	d := buildTwoShardDict(t)

	words := []string{"слово000001", "слово032000", "слово065535", "слово000500"}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		for _, w := range words {
			wg.Add(1)
			go func(word string) {
				defer wg.Done()
				readings := d.Parse(word)
				assert.NotEmpty(t, readings, "word %q must resolve under concurrent access", word)
				_ = d.Fuzzy(word, 1)
				_ = d.Lemma(word)
			}(w)
		}
	}
	wg.Wait()
}
