package tagmap_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapOpenCorporaKnownTag(t *testing.T) {
	b, ok := tagmap.Map("opencorpora", "NOUN,anim,masc,sing,nomn")
	require.True(t, ok)

	assert.Equal(t, []tagmap.Feature{
		{Dim: tagmap.DimPartOfSpeech, Value: "N"},
		{Dim: tagmap.DimAnimacy, Value: "ANIM"},
		{Dim: tagmap.DimCase, Value: "NOM"},
		{Dim: tagmap.DimNumber, Value: "SG"},
		{Dim: tagmap.DimGender, Value: "MASC"},
	}, b.Features)
}

func TestMapUnregisteredDictNameReturnsFalse(t *testing.T) {
	b, ok := tagmap.Map("some-future-source", "whatever,tag")

	assert.False(t, ok)
	assert.Equal(t, tagmap.Bundle{}, b)
}

func TestMapUnmappedTokenPassesThrough(t *testing.T) {
	b, ok := tagmap.Map("opencorpora", "NOUN,Slng")
	require.True(t, ok)

	assert.Equal(t, []string{"Slng"}, b.Unmapped)
}

func TestMapOpenCorporaAndOpenCorporaIntAgree(t *testing.T) {
	// The real documented example (docs/todo.md's tag-mapping section):
	// same word "кот", same grammatical meaning, two different native
	// tag syntaxes — must normalize to the same Bundle.Features.
	oc, ok := tagmap.Map("opencorpora", "NOUN,anim,masc,sing,nomn")
	require.True(t, ok)
	pm, ok := tagmap.Map("opencorpora-int", "NOUN,anim,masc sing,nomn")
	require.True(t, ok)

	assert.Equal(t, oc.Features, pm.Features)
}
