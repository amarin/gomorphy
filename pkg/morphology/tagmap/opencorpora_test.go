package tagmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizeOpenCorporaSplitsOnComma(t *testing.T) {
	assert.Equal(t,
		[]string{"NOUN", "anim", "masc", "sing", "nomn"},
		tokenizeOpenCorpora("NOUN,anim,masc,sing,nomn"))
}

func TestTokenizeOpenCorporaSingleToken(t *testing.T) {
	assert.Equal(t, []string{"NOUN"}, tokenizeOpenCorpora("NOUN"))
}

func TestTokenizeOpenCorporaEmptyTagIsNil(t *testing.T) {
	assert.Nil(t, tokenizeOpenCorpora(""))
}

func TestOpenCorporaTableCoversKotNominativeSingular(t *testing.T) {
	// "кот" (NOUN,anim,masc,sing,nomn) — the exact example already used
	// throughout docs/todo.md and docs/implementation/ for this dictionary.
	b := buildBundle(tokenizeOpenCorpora("NOUN,anim,masc,sing,nomn"), openCorporaTable)

	assert.Equal(t, []Feature{
		{DimPartOfSpeech, "N"},
		{DimAnimacy, "ANIM"},
		{DimCase, "NOM"},
		{DimNumber, "SG"},
		{DimGender, "MASC"},
	}, b.Features)
	assert.Nil(t, b.Unmapped)
}
