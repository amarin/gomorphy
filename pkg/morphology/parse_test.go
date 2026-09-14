package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	m := map[string]uint32{}
	stdWords(m)
	return buildFixture(t, m, nil, nil)
}

func TestParseKota(t *testing.T) {
	d := parseDict(t)

	readings := d.Parse("кота")
	require.Len(t, readings, 1)

	r := readings[0]
	assert.Equal(t, "кота", r.Word)
	assert.Equal(t, "кот", r.Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,gent", r.Tag)
	assert.Equal(t, uint16(0), r.Para)
	assert.Equal(t, uint16(1), r.Form)
}

func TestParseKotLowercased(t *testing.T) {
	d := parseDict(t)

	readings := d.Parse("КОТ")
	require.Len(t, readings, 2)
	for _, r := range readings {
		assert.Equal(t, "кот", r.Word)
		assert.Equal(t, "кот", r.Normal)
	}
}

func TestParseSortedByProbability(t *testing.T) {
	m := map[string]uint32{}
	stdWords(m)
	d := buildFixture(t, m, nil, map[string]uint32{
		"кот:NOUN,anim,masc,sing,nomn": 500,
		"кот:VERB,impf,trans":          100,
	})

	readings := d.Parse("кот")
	require.Len(t, readings, 2)

	assert.Equal(t, "NOUN,anim,masc,sing,nomn", readings[0].Tag)
	assert.Equal(t, "VERB,impf,trans", readings[1].Tag)
	assert.InDelta(t, 0.0005, readings[0].Prob, 1e-9)
	assert.InDelta(t, 0.0001, readings[1].Prob, 1e-9)
}

func TestParseProbZeroWhenAbsent(t *testing.T) {
	d := parseDict(t)

	readings := d.Parse("мышь")
	require.Len(t, readings, 1)
	assert.Equal(t, "мышь", readings[0].Normal)
	assert.Equal(t, uint16(2), readings[0].Para)
	assert.Equal(t, uint16(0), readings[0].Form)
	assert.Equal(t, 0.0, readings[0].Prob)
}

func TestParseUnknownNoPrediction(t *testing.T) {
	d := parseDict(t)
	assert.Nil(t, d.Parse("котёнок"))
}

func TestParsePredictedForm(t *testing.T) {
	m := map[string]uint32{}
	stdWords(m)
	pred := map[string]uint32{}
	addPrediction(pred, "ёнк", 2, 0, 0)
	addPrediction(pred, "ёнка", 3, 0, 1)
	d := buildFixture(t, m, pred, nil)

	readings := d.Parse("котёнка")
	require.Len(t, readings, 1)

	r := readings[0]
	assert.Equal(t, "котёнка", r.Word)
	assert.Equal(t, "котёнк", r.Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,gent", r.Tag)
	assert.Equal(t, uint16(0), r.Para)
	assert.Equal(t, uint16(1), r.Form)
}

func TestParseNilDictionary(t *testing.T) {
	var d *morphology.Dictionary
	assert.Nil(t, d.Parse("кот"))
}
