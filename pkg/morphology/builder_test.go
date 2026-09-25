package morphology_test

import (
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildSmallDict assembles a small two-lemma dictionary via the public
// Builder and returns the built Dictionary.
func buildSmallDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))
	require.NoError(t, b.AddLemma("мышь", "NOUN,anim,femn,sing,nomn"))
	require.NoError(t, b.AddForm("мыши", "мышь", "NOUN,anim,femn,sing,gent"))
	d, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, d)
	return d
}

// TestBuilderParse verifies Parse returns the expected (Word, Normal, Tag)
// for a dictionary built through the public Builder without going through
// SaveTo/Open.
func TestBuilderRoundTrip(t *testing.T) {
	d := buildSmallDict(t)

	out := filepath.Join(t.TempDir(), "builder.dat")
	require.NoError(t, d.SaveTo(out))

	got, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()

	cmpSnapshots(t,
		readingsSnapshot(d, "кот", "кота", "мышь", "мыши"),
		readingsSnapshot(got, "кот", "кота", "мышь", "мыши"),
	)
}

// TestBuilderParse verifies Parse returns the expected (Word, Normal, Tag).
func TestBuilderParse(t *testing.T) {
	d := buildSmallDict(t)

	readings := d.Parse("кот")
	require.Len(t, readings, 1)
	r := readings[0]
	assert.Equal(t, "кот", r.Word)
	assert.Equal(t, "кот", r.Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", r.Tag)

	readings = d.Parse("кота")
	require.Len(t, readings, 1)
	r = readings[0]
	assert.Equal(t, "кота", r.Word)
	assert.Equal(t, "кот", r.Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,gent", r.Tag)

	readings = d.Parse("мыши")
	require.Len(t, readings, 1)
	r = readings[0]
	assert.Equal(t, "мыши", r.Word)
	assert.Equal(t, "мышь", r.Normal)
	assert.Equal(t, "NOUN,anim,femn,sing,gent", r.Tag)
}

// TestBuilderAutoLemma verifies an empty lemma makes the wordform its own
// lemma: Parse's Normal equals the queried Word.
func TestBuilderAutoLemma(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddForm("кот", "", "NOUN,anim,masc,sing,nomn"))

	d, err := b.Build()
	require.NoError(t, err)

	readings := d.Parse("кот")
	require.Len(t, readings, 1)
	assert.Equal(t, "кот", readings[0].Normal)
	assert.Equal(t, "кот", readings[0].Word)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", readings[0].Tag)
}

// TestBuilderTripleDedup verifies exact (word, lemma, tag) repeats collapse
// to a single reading while a homonym — the same word with a different tag —
// is kept as a second reading.
func TestBuilderTripleDedup(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddForm("кот", "кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddForm("кот", "кот", "NOUN,anim,masc,sing,nomn")) // exact repeat
	require.NoError(t, b.AddForm("кот", "кот", "VERB,impf,trans"))          // homonym

	d, err := b.Build()
	require.NoError(t, err)

	readings := d.Parse("кот")
	require.Len(t, readings, 2, "exact triple collapsed, but the homonym reading is kept")

	var tags []string
	for _, r := range readings {
		tags = append(tags, r.Tag)
	}
	assert.ElementsMatch(t, []string{"NOUN,anim,masc,sing,nomn", "VERB,impf,trans"}, tags)
}

// TestBuilderAddLemma verifies AddLemma registers the normal form as its own
// wordform.
func TestBuilderAddLemma(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))

	d, err := b.Build()
	require.NoError(t, err)

	readings := d.Parse("кот")
	require.Len(t, readings, 1)
	assert.Equal(t, "кот", readings[0].Word)
	assert.Equal(t, "кот", readings[0].Normal)
}

// TestBuilderAddFormEmptyWord verifies an empty or whitespace-only word form
// is rejected with an error (not ErrNoEntries).
func TestBuilderAddFormEmptyWord(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})

	assert.Error(t, b.AddForm("", "", ""))
	assert.Error(t, b.AddForm(" ", "кот", "NOUN"))
	assert.Error(t, b.AddForm("\t", "кот", "NOUN"))
}

// TestBuilderBuildNoEntries verifies Build on a Builder with no entries
// returns ErrNoEntries.
func TestBuilderBuildNoEntries(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})

	_, err := b.Build()
	assert.ErrorIs(t, err, morphology.ErrNoEntries)
}

// TestBuilderBuildOversizedCharPolicy verifies a CharPolicy with more than
// 255 substitutions is rejected at Build time with a wrapped error,
// instead of panicking later inside EncodeMeta (SaveTo/ContentHash).
func TestBuilderBuildOversizedCharPolicy(t *testing.T) {
	subs := make([]morphology.Substitution, 256)
	for i := range subs {
		subs[i] = morphology.Substitution{From: rune('a' + i), To: rune('A' + i)}
	}
	b := morphology.NewBuilder(morphology.BuilderOptions{CharPolicy: morphology.NewCharPolicy(subs...)})
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))

	_, err := b.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "256")
}

// TestBuilderClosedAfterBuild verifies the Builder is consumed by Build:
// further AddForm and Build calls are rejected with ErrBuilderClosed.
func TestBuilderClosedAfterBuild(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))

	_, err := b.Build()
	require.NoError(t, err)

	err = b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent")
	assert.ErrorIs(t, err, morphology.ErrBuilderClosed)

	_, err = b.Build()
	assert.ErrorIs(t, err, morphology.ErrBuilderClosed)
}

// TestBuilderDenseFuzzy verifies Fuzzy/FuzzyTop return plausible results on
// a Builder-made dictionary (these are alphabet- and shard-aware and work
// without prediction or probability).
func TestBuilderDenseFuzzy(t *testing.T) {
	d := buildSmallDict(t)

	fuzzy := d.Fuzzy("кот", 1)
	require.NotEmpty(t, fuzzy, "Fuzzy must find dictionary words within distance 1")
	assert.Equal(t, "кот", fuzzy[0].Word)

	top := d.FuzzyTop("кот", 3)
	require.NotEmpty(t, top, "FuzzyTop must find dictionary words")
	assert.LessOrEqual(t, len(top), 3)
	assert.Equal(t, "кот", top[0].Word)
}

// TestBuilderPrediction verifies prediction is rebuilt and functional: an
// out-of-dictionary word with a productive ending ("ота", attested by
// "кота") gets a predicted reading with a reconstructed lemma.
func TestBuilderPrediction(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))

	d, err := b.Build()
	require.NoError(t, err)

	readings := d.Parse("пилота")
	require.NotEmpty(t, readings, "prediction must produce a reading for an OOV word")
	assert.Equal(t, "пилота", readings[0].Word)
	assert.Equal(t, "пилот", readings[0].Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,gent", readings[0].Tag)
}
