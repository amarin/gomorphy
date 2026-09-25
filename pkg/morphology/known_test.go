package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// predictionFixture is the pymorphy2 fixture with two prediction suffixes,
// the same one TestParsePredictedForm uses: "котёнка" is absent from the
// dictionary and is predicted via "ёнка".
func predictionFixture(t *testing.T) *morphology.Dictionary {
	t.Helper()
	m := map[string]uint32{}
	stdWords(m)
	pred := map[string]uint32{}
	addPrediction(pred, "ёнк", 2, 0, 0)
	addPrediction(pred, "ёнка", 3, 0, 1)
	return buildFixture(t, m, pred, nil)
}

func TestParsePredictedFlagPyMorphy(t *testing.T) {
	d := predictionFixture(t)

	exact := d.Parse("кота")
	require.NotEmpty(t, exact)
	for _, r := range exact {
		assert.False(t, r.Predicted, "dictionary reading %+v", r)
	}

	predicted := d.Parse("котёнка")
	require.NotEmpty(t, predicted)
	for _, r := range predicted {
		assert.True(t, r.Predicted, "predicted reading %+v", r)
	}
}

// Every Builder dictionary gets a prediction DAWG (buildFromEntries calls
// BuildPrediction), so even a 4-word dictionary "predicts" "бота" from "кота".
func TestParsePredictedFlagBuilder(t *testing.T) {
	d := buildSmallDict(t)

	for _, r := range d.Parse("кота") {
		assert.False(t, r.Predicted, "dictionary reading %+v", r)
	}
	predicted := d.Parse("бота")
	require.NotEmpty(t, predicted, "builder dictionaries predict unknown words")
	for _, r := range predicted {
		assert.True(t, r.Predicted, "predicted reading %+v", r)
	}
}

func TestLemmaPredictedFlag(t *testing.T) {
	d := buildSmallDict(t)

	refs := d.Lemma("кота")
	require.Len(t, refs, 1)
	assert.False(t, refs[0].Predicted)

	refs = d.Lemma("бота")
	require.NotEmpty(t, refs)
	for _, ref := range refs {
		assert.True(t, ref.Predicted, "%+v", ref)
	}
}

// botDict knows "бот"/"бота" exactly; buildSmallDict only predicts "бота".
func botDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	return buildFromTriples(t,
		[3]string{"бот", "бот", "NOUN,inan,masc,sing,nomn"},
		[3]string{"бота", "бот", "NOUN,inan,masc,sing,gent"},
	)
}

func TestMultiDictionaryParseMixesPredictedAndExact(t *testing.T) {
	m := morphology.NewMultiDictionary(buildSmallDict(t), botDict(t))

	var sawPredicted, sawExact bool
	for _, r := range m.Parse("бота") {
		switch r.Dict {
		case 0:
			assert.True(t, r.Predicted, "dict 0 does not know бота: %+v", r)
			sawPredicted = true
		case 1:
			assert.False(t, r.Predicted, "dict 1 knows бота: %+v", r)
			sawExact = true
		}
	}
	assert.True(t, sawPredicted)
	assert.True(t, sawExact)
}

func TestIsKnown(t *testing.T) {
	d := buildSmallDict(t)

	assert.True(t, d.IsKnown("кота"))
	assert.True(t, d.IsKnown("КОТА"), "input is lower-cased like Parse")
	assert.False(t, d.IsKnown("бота"), "predicted by Parse, but not in the dictionary")
	assert.NotEmpty(t, d.Parse("бота"))
	assert.False(t, d.IsKnown(""))

	var nilDict *morphology.Dictionary
	assert.False(t, nilDict.IsKnown("кот"))
}

// parseDict's pymorphy2 fixture stores "ёж" and uses RussianCharPolicy:
// IsKnown applies the same е→ё substitution as Parse.
func TestIsKnownAppliesCharPolicy(t *testing.T) {
	d := parseDict(t)
	assert.True(t, d.IsKnown("еж"))
	assert.True(t, d.IsKnown("ёж"))
}

// IsKnown(w) is true exactly when Parse(w) returns non-predicted readings.
func TestIsKnownAgreesWithParse(t *testing.T) {
	d := predictionFixture(t)
	for _, w := range []string{"кот", "кота", "мышь", "ежик", "котёнка", "неттакогослова"} {
		rs := d.Parse(w)
		known := len(rs) > 0 && !rs[0].Predicted
		assert.Equal(t, known, d.IsKnown(w), w)
	}
}

func TestMultiDictionaryIsKnown(t *testing.T) {
	m := morphology.NewMultiDictionary(buildSmallDict(t), botDict(t))
	assert.True(t, m.IsKnown("кота"))
	assert.True(t, m.IsKnown("бота"))
	assert.False(t, m.IsKnown("зебра"))
	assert.False(t, morphology.NewMultiDictionary().IsKnown("кот"))
}
