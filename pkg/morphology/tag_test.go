package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
)

func TestGrammemes(t *testing.T) {
	assert.Equal(t, []string{"NOUN", "anim", "masc", "Surn", "sing", "ablt"},
		morphology.Grammemes("NOUN,anim,masc,Surn sing,ablt"))
	assert.Equal(t, []string{"N", "GEN", "SG"}, morphology.Grammemes("N;GEN;SG"))
	assert.Nil(t, morphology.Grammemes(""))
	assert.Nil(t, morphology.Grammemes(", ;"))
}

func TestHasGrammeme(t *testing.T) {
	tag := "NOUN,anim,masc,Surn sing,ablt"
	assert.True(t, morphology.HasGrammeme(tag, "Surn"))
	assert.True(t, morphology.HasGrammeme(tag, "ablt"))
	assert.True(t, morphology.HasGrammeme("N;GEN;SG", "GEN"))
	assert.False(t, morphology.HasGrammeme(tag, "Sur"), "whole tokens only")
	assert.False(t, morphology.HasGrammeme(tag, "Surn sing"), "a separator is never part of a token")
	assert.False(t, morphology.HasGrammeme(tag, ""))
	assert.False(t, morphology.HasGrammeme("", "NOUN"))

	if !raceEnabled {
		allocs := testing.AllocsPerRun(100, func() { _ = morphology.HasGrammeme(tag, "ablt") })
		assert.Equal(t, 0.0, allocs, "HasGrammeme must not allocate")
	}
}

func TestPOS(t *testing.T) {
	assert.Equal(t, "NOUN", morphology.POS("NOUN,anim,masc sing,nomn"))
	assert.Equal(t, "INFN", morphology.POS("INFN,impf,tran"), "POS is the raw first grammeme, no class mapping")
	assert.Equal(t, "N", morphology.POS("N;GEN;SG"))
	assert.Equal(t, "", morphology.POS(""))
}

func TestReadingHasGrammeme(t *testing.T) {
	rs := parseDict(t).Parse("кота")
	assert.True(t, rs[0].HasGrammeme("gent"))
	assert.False(t, rs[0].HasGrammeme("nomn"))
}
