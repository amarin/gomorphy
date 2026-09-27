package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAppendMatchesParse(t *testing.T) {
	for name, d := range map[string]*morphology.Dictionary{
		"builder":   buildSmallDict(t),
		"pymorphy":  predictionFixture(t),
		"twoShards": buildTwoShardDict(t),
	} {
		for _, w := range []string{"кот", "КОТА", "мыши", "ежик", "бота", "котёнка", "слово000010", "слово065535", "неттакого"} {
			want := d.Parse(w)
			prefix := []morphology.Reading{{Word: "sentinel"}}
			got := d.ParseAppend(prefix, w)
			require.Equal(t, "sentinel", got[0].Word, "%s/%s: dst prefix kept", name, w)
			assert.Equal(t, want, nilIfEmpty(got[1:]), "%s/%s", name, w)
		}
	}
	var nilDict *morphology.Dictionary
	assert.Nil(t, nilDict.ParseAppend(nil, "кот"))
}

func nilIfEmpty(rs []morphology.Reading) []morphology.Reading {
	if len(rs) == 0 {
		return nil
	}
	return rs
}

// TestParseAllocs bounds allocations on a single-shard dense Builder
// dictionary. Planning-time baseline (1.1.0, Parse): кот 22, кота 24,
// бота (predicted) 44. When the measured value is lower than the bound,
// tighten the bound to it (+1 slack) and note both numbers in the
// implementation write-up.
func TestParseAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under -race")
	}
	d := buildSmallDict(t)
	buf := make([]morphology.Reading, 0, 16)

	for _, tc := range []struct {
		word      string
		appendMax float64 // ParseAppend with a reused buffer
		parseMax  float64 // Parse (allocates the result slice)
	}{
		{"кот", 1, 2},  // form 0: Normal is the word itself; measured 0/1
		{"кота", 1, 2}, // form 1: Normal = prefix0+stem+suffix0 (one concat); measured 0/1
		{"бота", 3, 4}, // predicted: Word and Normal concats, seen map; measured 2/3
	} {
		require.NotEmpty(t, d.ParseAppend(buf[:0], tc.word))
		got := testing.AllocsPerRun(200, func() { buf = d.ParseAppend(buf[:0], tc.word) })
		t.Logf("ParseAppend(%q): %v allocs", tc.word, got)
		assert.LessOrEqual(t, got, tc.appendMax, "ParseAppend(%q)", tc.word)

		got = testing.AllocsPerRun(200, func() { _ = d.Parse(tc.word) })
		t.Logf("Parse(%q): %v allocs", tc.word, got)
		assert.LessOrEqual(t, got, tc.parseMax, "Parse(%q)", tc.word)
	}
}
