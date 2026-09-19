package tagmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizeUniMorphSplitsOnSemicolon(t *testing.T) {
	assert.Equal(t, []string{"N", "NOM", "SG"}, tokenizeUniMorph("N;NOM;SG"))
}

func TestTokenizeUniMorphSingleToken(t *testing.T) {
	assert.Equal(t, []string{"N"}, tokenizeUniMorph("N"))
}

func TestTokenizeUniMorphEmptyTagIsNil(t *testing.T) {
	assert.Nil(t, tokenizeUniMorph(""))
}

func TestUniMorphTableCoversKotNominativeSingular(t *testing.T) {
	// "кот" (N;NOM;SG) — the real rus row for this dictionary's other
	// worked example (see docs/en/todo.md, docs/en/implementation/); no
	// ANIM/gender token here, unlike OpenCorpora's tag for the same word
	// — the real rus data doesn't carry them for this paradigm slot.
	b := buildBundle(tokenizeUniMorph("N;NOM;SG"), unimorphTable)

	assert.Equal(t, []Feature{
		{DimPartOfSpeech, "N"},
		{DimCase, "NOM"},
		{DimNumber, "SG"},
	}, b.Features)
	assert.Nil(t, b.Unmapped)
}

func TestUniMorphTableLeavesCompoundPOSMarkersUnmapped(t *testing.T) {
	// V.PTCP ("participle") and LGSPEC1 (a language-specific feature,
	// the rus reflexive "-ся" marker) are real tokens from the rus
	// dataset with no corresponding Dimension in this package.
	b := buildBundle(tokenizeUniMorph("V.PTCP;NOM;MASC;SG;LGSPEC1"), unimorphTable)

	assert.Equal(t, []Feature{
		{DimCase, "NOM"},
		{DimNumber, "SG"},
		{DimGender, "MASC"},
	}, b.Features)
	assert.Equal(t, []string{"V.PTCP", "LGSPEC1"}, b.Unmapped)
}
