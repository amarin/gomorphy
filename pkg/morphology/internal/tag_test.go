package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextGrammeme(t *testing.T) {
	tag := "NOUN,anim,masc,Surn sing,ablt"
	var got []string
	for i := 0; ; {
		tok, next := NextGrammeme(tag, i)
		if tok == "" {
			break
		}
		got = append(got, tok)
		i = next
	}
	assert.Equal(t, []string{"NOUN", "anim", "masc", "Surn", "sing", "ablt"}, got)

	tok, next := NextGrammeme(" ,;", 0)
	assert.Equal(t, "", tok)
	assert.Equal(t, 3, next)
}

func TestFirstGrammeme(t *testing.T) {
	assert.Equal(t, "NOUN", FirstGrammeme("NOUN,anim,masc sing,nomn"))
	assert.Equal(t, "N", FirstGrammeme("N;GEN;SG"))
	assert.Equal(t, "", FirstGrammeme(""))
	assert.Equal(t, "INFN", FirstGrammeme(" INFN,impf"))
}

func TestPOSClass(t *testing.T) {
	for tag, want := range map[string]string{
		"NOUN,anim,masc,sing,nomn": "NOUN",
		"INFN,impf,tran":           "VERB",
		"VERB,impf,tran,sing,1per": "VERB",
		"PRTF,impf,pres,actv":      "VERB",
		"PRTS,perf,past,pssv":      "VERB",
		"GRND,impf,pres":           "VERB",
		"ADJF,Qual,masc,sing,nomn": "ADJF",
		"ADJS,masc,sing":           "ADJF",
		"COMP,Qual":                "ADJF",
		"V;PRS;1;SG":               "V",
		"V.PTCP;ACT;PRS":           "V",
		"V.CVB;PRS":                "V",
		"V.MSDR":                   "V",
		"ADJ;NOM;SG":               "ADJ",
		"":                         "",
	} {
		assert.Equal(t, want, POSClass(tag), tag)
	}
}
