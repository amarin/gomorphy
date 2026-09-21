package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mergeTags are the opaque grammeme tags the Merge fixture corpora use.
const (
	mergeTagMascNomn = "NOUN,anim,masc,sing,nomn"
	mergeTagMascGent = "NOUN,anim,masc,sing,gent"
	mergeTagFemnNomn = "NOUN,anim,femn,sing,nomn"
	mergeTagFemnGent = "NOUN,anim,femn,sing,gent"
	mergeTagVerb     = "VERB,impf,trans"
)

// buildFromTriples assembles a dictionary via the public Builder from
// (word, lemma, tag) triples — the raw-entries equivalent of a Merge input.
func buildFromTriples(t *testing.T, triples ...[3]string) *morphology.Dictionary {
	t.Helper()
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	for _, tr := range triples {
		require.NoError(t, b.AddForm(tr[0], tr[1], tr[2]))
	}
	d, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, d)
	return d
}

// semanticSnapshot captures Parse results keeping only the semantic fields
// (Word, Normal, Tag). Identity fields (Para/Form/Shard/Dict/Prob) are
// zeroed so two dictionaries built from the same entries compare equal
// regardless of internal paradigm numbering.
func semanticSnapshot(d *morphology.Dictionary, words ...string) map[string][]morphology.Reading {
	out := make(map[string][]morphology.Reading, len(words))
	for _, w := range words {
		for _, r := range d.Parse(w) {
			r.Para, r.Form, r.Shard, r.Dict, r.Prob = 0, 0, 0, 0, 0
			out[w] = append(out[w], r)
		}
	}
	return out
}

// TestMergeAdd verifies MergeAdd semantics: overlay-only words are added
// with their own (lemma, tag); a word present in the base keeps only the
// base's readings — the overlay's conflicting readings are dropped.
func TestMergeAdd(t *testing.T) {
	base := buildSmallDict(t) // кот/кота, мышь/мыши
	overlay := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagVerb},             // present in base: dropped
		[3]string{"котёнок", "котёнок", mergeTagMascNomn}, // new word
		[3]string{"котёнка", "котёнок", mergeTagMascGent}, // new word
	)

	merged, err := morphology.Merge(base, []*morphology.Dictionary{overlay}, morphology.MergeAdd)
	require.NoError(t, err)

	// A word in both keeps only base readings.
	readings := merged.Parse("кот")
	require.Len(t, readings, 1)
	assert.Equal(t, mergeTagMascNomn, readings[0].Tag)
	assert.Equal(t, readingsSnapshot(base, "кот")["кот"], readings, "base reading unchanged")

	// Overlay-only words added with their own (lemma, tag).
	readings = merged.Parse("котёнок")
	require.Len(t, readings, 1)
	assert.Equal(t, "котёнок", readings[0].Word)
	assert.Equal(t, "котёнок", readings[0].Normal)
	assert.Equal(t, mergeTagMascNomn, readings[0].Tag)

	readings = merged.Parse("котёнка")
	require.Len(t, readings, 1)
	assert.Equal(t, "котёнка", readings[0].Word)
	assert.Equal(t, "котёнок", readings[0].Normal)
	assert.Equal(t, mergeTagMascGent, readings[0].Tag)

	// Base-only words untouched.
	readings = merged.Parse("мышь")
	require.Len(t, readings, 1)
	assert.Equal(t, mergeTagFemnNomn, readings[0].Tag)
}

// TestMergeReplace verifies MergeReplace semantics: for words in both the
// overlay's readings fully replace the base's; words unique to either side
// are preserved; prediction is rebuilt and works via Parse on an
// out-of-dictionary word.
func TestMergeReplace(t *testing.T) {
	base := buildSmallDict(t) // кот/кота, мышь/мыши
	overlay := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagVerb},     // in both: replaced
		[3]string{"мышь", "мышь", mergeTagVerb},   // in both: replaced
		[3]string{"пёс", "пёс", mergeTagMascNomn}, // overlay-only
		[3]string{"пса", "пёс", mergeTagMascGent}, // overlay-only
	)

	merged, err := morphology.Merge(base, []*morphology.Dictionary{overlay}, morphology.MergeReplace)
	require.NoError(t, err)

	// Words in both: only the overlay readings remain.
	readings := merged.Parse("кот")
	require.Len(t, readings, 1)
	assert.Equal(t, mergeTagVerb, readings[0].Tag)

	readings = merged.Parse("мышь")
	require.Len(t, readings, 1)
	assert.Equal(t, mergeTagVerb, readings[0].Tag)

	// Base-unique words preserved.
	readings = merged.Parse("кота")
	require.Len(t, readings, 1)
	assert.Equal(t, "кот", readings[0].Normal)
	assert.Equal(t, mergeTagMascGent, readings[0].Tag)

	readings = merged.Parse("мыши")
	require.Len(t, readings, 1)
	assert.Equal(t, "мышь", readings[0].Normal)
	assert.Equal(t, mergeTagFemnGent, readings[0].Tag)

	// Overlay-unique words preserved.
	readings = merged.Parse("пёс")
	require.Len(t, readings, 1)
	assert.Equal(t, "пёс", readings[0].Normal)
	assert.Equal(t, mergeTagMascNomn, readings[0].Tag)

	readings = merged.Parse("пса")
	require.Len(t, readings, 1)
	assert.Equal(t, "пёс", readings[0].Normal)
	assert.Equal(t, mergeTagMascGent, readings[0].Tag)

	// Prediction rebuilt: an OOV word sharing кота's productive "ота"
	// suffix ("кота" is preserved as base-unique) gets a predicted reading
	// with a reconstructed lemma.
	readings = merged.Parse("пилота")
	require.NotEmpty(t, readings, "prediction must produce a reading for an OOV word")
	assert.Equal(t, "пилота", readings[0].Word)
	assert.Equal(t, "пилот", readings[0].Normal)
	assert.Equal(t, mergeTagMascGent, readings[0].Tag)
}

// TestMergeMultipleOverlays verifies overlays apply in argument order under
// MergeAdd: a conflict word stays base-only; a word introduced by a later
// overlay is added; a new word present in two overlays accumulates readings
// (later overlays append).
func TestMergeMultipleOverlays(t *testing.T) {
	base := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagMascNomn},
	)
	overlay1 := buildFromTriples(t,
		[3]string{"щенок", "щенок", mergeTagMascNomn},
	)
	overlay2 := buildFromTriples(t,
		[3]string{"пёс", "пёс", mergeTagMascNomn}, // introduced by overlay 2
		[3]string{"кот", "кот", mergeTagVerb},     // conflict with base: dropped
		[3]string{"щенок", "щенок", mergeTagVerb}, // same new word as overlay 1: appended
	)

	merged, err := morphology.Merge(base, []*morphology.Dictionary{overlay1, overlay2}, morphology.MergeAdd)
	require.NoError(t, err)

	readings := merged.Parse("кот")
	require.Len(t, readings, 1, "overlay 2's VERB reading for a base word must be dropped")
	assert.Equal(t, mergeTagMascNomn, readings[0].Tag)

	readings = merged.Parse("пёс")
	require.Len(t, readings, 1)
	assert.Equal(t, "пёс", readings[0].Normal)
	assert.Equal(t, mergeTagMascNomn, readings[0].Tag)

	readings = merged.Parse("щенок")
	require.Len(t, readings, 2, "a new word in two overlays accumulates both readings")
	var tags []string
	for _, r := range readings {
		tags = append(tags, r.Tag)
	}
	assert.ElementsMatch(t, []string{mergeTagMascNomn, mergeTagVerb}, tags)
}

// TestMergeDenseRoundTrip verifies two dense Builder-produced inputs merge
// into a dense output (white-box Alphabet check lives in
// merge_internal_test.go), Fuzzy works on the result, and Parse on the
// merged dictionary equals Parse on the equivalent raw-entries Builder
// dictionary.
func TestMergeDenseRoundTrip(t *testing.T) {
	base := buildSmallDict(t)
	overlay := buildFromTriples(t,
		[3]string{"пёс", "пёс", mergeTagMascNomn},
		[3]string{"пса", "пёс", mergeTagMascGent},
		[3]string{"котёнок", "котёнок", mergeTagMascNomn},
		[3]string{"котёнка", "котёнок", mergeTagMascGent},
	)

	merged, err := morphology.Merge(base, []*morphology.Dictionary{overlay}, morphology.MergeAdd)
	require.NoError(t, err)

	fuzzy := merged.Fuzzy("кот", 1)
	require.NotEmpty(t, fuzzy, "Fuzzy must find dictionary words on the merged output")
	assert.Equal(t, "кот", fuzzy[0].Word)

	equiv := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagMascNomn},
		[3]string{"кота", "кот", mergeTagMascGent},
		[3]string{"мышь", "мышь", mergeTagFemnNomn},
		[3]string{"мыши", "мышь", mergeTagFemnGent},
		[3]string{"пёс", "пёс", mergeTagMascNomn},
		[3]string{"пса", "пёс", mergeTagMascGent},
		[3]string{"котёнок", "котёнок", mergeTagMascNomn},
		[3]string{"котёнка", "котёнок", mergeTagMascGent},
	)

	cmpSnapshots(t,
		semanticSnapshot(merged, "кот", "кота", "мышь", "мыши", "пёс", "пса", "котёнок", "котёнка"),
		semanticSnapshot(equiv, "кот", "кота", "мышь", "мыши", "пёс", "пса", "котёнок", "котёнка"),
	)
}

// TestMergeInputsNotMutated verifies Merge never mutates its inputs: the
// base's and overlays' Parse output are unchanged after the call.
func TestMergeInputsNotMutated(t *testing.T) {
	base := buildSmallDict(t)
	overlay := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagVerb},
		[3]string{"пёс", "пёс", mergeTagMascNomn},
	)

	baseBefore := readingsSnapshot(base, "кот", "кота", "мышь", "мыши")
	overlayBefore := readingsSnapshot(overlay, "кот", "пёс")

	_, err := morphology.Merge(base, []*morphology.Dictionary{overlay}, morphology.MergeReplace)
	require.NoError(t, err)

	cmpSnapshots(t, baseBefore, readingsSnapshot(base, "кот", "кота", "мышь", "мыши"))
	cmpSnapshots(t, overlayBefore, readingsSnapshot(overlay, "кот", "пёс"))
}

// TestMergeNilInputs verifies a nil base and a nil overlay element are
// rejected with an error rather than panicking.
func TestMergeNilInputs(t *testing.T) {
	_, err := morphology.Merge(nil, nil, morphology.MergeAdd)
	require.Error(t, err)

	base := buildSmallDict(t)
	_, err = morphology.Merge(base, []*morphology.Dictionary{nil}, morphology.MergeAdd)
	require.Error(t, err)
}
