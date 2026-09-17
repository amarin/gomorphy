package tagmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildBundleCanonicalOrderIndependentOfInputOrder(t *testing.T) {
	table := map[string]Feature{
		"nomn": {DimCase, "NOM"},
		"NOUN": {DimPartOfSpeech, "N"},
		"sing": {DimNumber, "SG"},
	}

	// Input order deliberately scrambled relative to Dimension
	// declaration order (Case, PartOfSpeech, Number) to prove the
	// output order comes from Dimension, not from input order.
	b := buildBundle([]string{"nomn", "NOUN", "sing"}, table)

	assert.Equal(t, []Feature{
		{DimPartOfSpeech, "N"},
		{DimCase, "NOM"},
		{DimNumber, "SG"},
	}, b.Features)
	assert.Nil(t, b.Unmapped)
}

func TestBuildBundleUnknownTokenGoesToUnmapped(t *testing.T) {
	table := map[string]Feature{
		"NOUN": {DimPartOfSpeech, "N"},
	}

	b := buildBundle([]string{"NOUN", "Slng"}, table)

	assert.Equal(t, []Feature{{DimPartOfSpeech, "N"}}, b.Features)
	assert.Equal(t, []string{"Slng"}, b.Unmapped)
}

func TestBuildBundleLaterTokenWinsOnDimensionCollision(t *testing.T) {
	table := map[string]Feature{
		"nomn": {DimCase, "NOM"},
		"gent": {DimCase, "GEN"},
	}

	b := buildBundle([]string{"nomn", "gent"}, table)

	assert.Equal(t, []Feature{{DimCase, "GEN"}}, b.Features)
}

func TestBuildBundleEmptyInputProducesNilBundle(t *testing.T) {
	b := buildBundle(nil, map[string]Feature{"NOUN": {DimPartOfSpeech, "N"}})

	assert.Nil(t, b.Features)
	assert.Nil(t, b.Unmapped)
}

func TestBuildBundleTwoSourcesSameMeaningCompareEqual(t *testing.T) {
	// Simulates the cross-source scenario this whole package exists for:
	// two different token orders/tables producing the same Feature set
	// must compare equal.
	tableA := map[string]Feature{
		"NOUN": {DimPartOfSpeech, "N"},
		"sing": {DimNumber, "SG"},
	}
	tableB := map[string]Feature{
		"N_TAG":  {DimPartOfSpeech, "N"},
		"SG_TAG": {DimNumber, "SG"},
	}

	a := buildBundle([]string{"NOUN", "sing"}, tableA)
	b := buildBundle([]string{"SG_TAG", "N_TAG"}, tableB)

	assert.Equal(t, a.Features, b.Features)
}
