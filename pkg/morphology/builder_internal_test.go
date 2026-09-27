package morphology

import (
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilderDenseAndPrediction white-box check that a Builder-made
// dictionary is dense by default (Alphabet set) and carries a rebuilt
// prediction DAWG (len(Prediction) == 1 for the unsharded case).
func TestBuilderDenseAndPrediction(t *testing.T) {
	b := NewBuilder(BuilderOptions{})
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))

	d, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, d)
	require.NotNil(t, d.d)

	assert.NotNil(t, d.d.Alphabet, "output must be dense by default")
	assert.Len(t, d.d.Prediction, 1, "prediction must be rebuilt by default")
}

// TestBuilderInfoDefaults verifies BuilderOptions defaults reach the
// built dictionary: Language "ru", Source "builder", TagSet name "builder".
func TestBuilderInfoDefaults(t *testing.T) {
	b := NewBuilder(BuilderOptions{})
	require.NoError(t, b.AddForm("дело", "", "NOUN,neut,sing,nomn"))

	d, err := b.Build()
	require.NoError(t, err)

	assert.Equal(t, "ru", d.Language())
	assert.Equal(t, "builder", d.TagSetName())

	info := d.Info()
	require.NotNil(t, info)
	assert.Equal(t, "builder", info.Source)
}

// TestBuilderOptionsOverride verifies explicit BuilderOptions reach the
// built dictionary.
func TestBuilderOptionsOverride(t *testing.T) {
	b := NewBuilder(BuilderOptions{Language: "en", Source: "custom"})
	require.NoError(t, b.AddForm("дело", "", "NOUN,neut,sing,nomn"))

	d, err := b.Build()
	require.NoError(t, err)

	assert.Equal(t, "en", d.Language())
	require.NotNil(t, d.Info())
	assert.Equal(t, "custom", d.Info().Source)
}

// TestBuildFromEntriesSourceFallback verifies the shared helper's Info
// fallback: an empty BuilderOptions.Source falls back to tagSetName —
// the path Tasks 5/6 exercise when they name their own TagSets.
func TestBuildFromEntriesSourceFallback(t *testing.T) {
	entries := []internal.BuildEntry{
		{Word: "кот", Lemma: "кот", Tag: "NOUN,anim,masc,sing,nomn"},
	}
	d, err := buildFromEntries(BuilderOptions{Language: "ru"}, entries, "thematic")
	require.NoError(t, err)

	require.NotNil(t, d.Info())
	assert.Equal(t, "thematic", d.Info().Source)
}

// TestBuilderWideAlphabetFallback builds a dictionary over 300 distinct
// runes — more than a 1-byte dense alphabet holds — and checks it works end
// to end: 2-byte alphabet, Parse, Fuzzy, SaveTo/Open.
func TestBuilderWideAlphabetFallback(t *testing.T) {
	b := NewBuilder(BuilderOptions{Language: "zh"})
	var words []string
	for i := 0; i < 300; i++ {
		w := string(rune(0x4E00 + i))
		words = append(words, w)
		require.NoError(t, b.AddLemma(w, "X"))
	}
	d, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, d.d.Alphabet)
	assert.Equal(t, 2, d.d.Alphabet.Width())

	path := filepath.Join(t.TempDir(), "wide.dat")
	require.NoError(t, d.SaveTo(path))
	opened, err := Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, opened.Close()) }()

	for _, dict := range []*Dictionary{d, opened} {
		for _, w := range []string{words[0], words[150], words[299]} {
			rs := dict.Parse(w)
			require.Len(t, rs, 1, w)
			assert.False(t, rs[0].Predicted)
			assert.Equal(t, []FuzzyMatch{{Word: w}}, dict.Fuzzy(w, 0))
		}
	}
}
