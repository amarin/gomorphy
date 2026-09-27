package morphology_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func assertAllPredicted(t *testing.T, d *morphology.Dictionary, word string) {
	t.Helper()
	readings := d.Parse(word)
	require.NotEmpty(t, readings, "%s must be predicted", word)
	for _, r := range readings {
		assert.True(t, r.Predicted, "%s: %+v", word, r)
	}
}

// predictionXML: three nouns sharing one paradigm (""/"а"), so the
// pymorphy2-style pruning (≥3 lemmas per paradigm, ≥2 readings per ending)
// keeps their endings.
const predictionXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">сущ</grammeme><grammeme id="inan">неод</grammeme><grammeme id="masc">м</grammeme>
  <grammeme id="sing">ед</grammeme><grammeme id="nomn">им</grammeme><grammeme id="gent">род</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот"><l t="кот"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l><f t="кот"><g v="nomn"/></f><f t="кота"><g v="gent"/></f></lemma>
  <lemma id="2" text="лот"><l t="лот"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l><f t="лот"><g v="nomn"/></f><f t="лота"><g v="gent"/></f></lemma>
  <lemma id="3" text="скот"><l t="скот"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l><f t="скот"><g v="nomn"/></f><f t="скота"><g v="gent"/></f></lemma>
 </lemmata>
</dictionary>`

const predictionTSV = "кот\tкот\tN;NOM;SG\nкот\tкота\tN;GEN;SG\n" +
	"лот\tлот\tN;NOM;SG\nлот\tлота\tN;GEN;SG\n" +
	"скот\tскот\tN;NOM;SG\nскот\tскота\tN;GEN;SG\n"

func TestCompileFromXMLPredictsByDefault(t *testing.T) {
	for name, compile := range map[string]func() (*morphology.Dictionary, error){
		"CompileFromXML": func() (*morphology.Dictionary, error) {
			return morphology.CompileFromXML(strings.NewReader(predictionXML), nil)
		},
		"CompileFromXMLDense": func() (*morphology.Dictionary, error) {
			return morphology.CompileFromXMLDense(strings.NewReader(predictionXML), nil)
		},
	} {
		d, err := compile()
		require.NoError(t, err, name)
		assertAllPredicted(t, d, "бота") // like кота
	}

	d, err := morphology.CompileFromXMLWithOptions(strings.NewReader(predictionXML),
		morphology.XMLOptions{Dense: true, NoPrediction: true})
	require.NoError(t, err)
	assert.Nil(t, d.Parse("бота"), "NoPrediction keeps the 1.2 behaviour")
	assert.NotEmpty(t, d.Parse("кота"))

	tiny, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
	require.NoError(t, err)
	assert.Nil(t, tiny.Parse("бота"), "pruning drops paradigms with fewer than 3 lemmas")
}

func TestCompileFromUniMorphPredictsByDefault(t *testing.T) {
	opts := morphology.UniMorphOptions{Language: "ru"}
	d, err := morphology.CompileFromUniMorphDense(strings.NewReader(predictionTSV), opts)
	require.NoError(t, err)
	assertAllPredicted(t, d, "бота")

	opts.NoPrediction = true
	d, err = morphology.CompileFromUniMorphDense(strings.NewReader(predictionTSV), opts)
	require.NoError(t, err)
	assert.Nil(t, d.Parse("бота"))
}

// UniMorph rus has five parts of speech, all open classes; productive()
// splits on "," and must filter none of them (spec L, "Tag filter"). Each
// POS gets three lemmas sharing its paradigm, so pymorphy2-style pruning
// (≥3 lemmas per paradigm, ≥2 readings per ending, one winner per
// suffix+POS) keeps at least one probe per POS. Each probe ends with its
// form's suffix (стулами/«ами»): a key never is shorter than the form
// suffix (ruling R14), so «стульями» would not reach the N paradigm.
func TestCompileFromUniMorphPredictsAllPOS(t *testing.T) {
	tsv := ""
	for _, s := range []string{"стол", "вол", "кол"} {
		tsv += s + "\t" + s + "ами\tN;INS;PL\n" + s + "\t" + s + "\tN;NOM;SG\n"
	}
	for _, s := range []string{"син", "зимн", "летн"} {
		tsv += s + "ий\t" + s + "ими\tADJ;INS;PL\n" + s + "ий\t" + s + "ий\tADJ;NOM;SG;MASC\n"
	}
	for _, s := range []string{"чит", "кат", "мот"} {
		tsv += s + "ать\t" + s + "ать\tV;NFIN\n" +
			s + "ать\t" + s + "али\tV;PST;PL\n" +
			s + "ать\t" + s + "авшими\tV.PTCP;ACT;PST;INS;PL\n" +
			s + "ать\t" + s + "ая\tV.CVB;PRS\n"
	}
	d, err := morphology.CompileFromUniMorph(strings.NewReader(tsv), morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)

	pos := map[string]bool{}
	for _, w := range []string{"стулами", "красными", "писали", "писавшими", "пиная"} {
		assertAllPredicted(t, d, w)
		for _, r := range d.Parse(w) {
			p, _, _ := strings.Cut(r.Tag, ";")
			pos[p] = true
		}
	}
	assert.Equal(t, map[string]bool{"N": true, "ADJ": true, "V": true, "V.PTCP": true, "V.CVB": true}, pos)
}

// TestRealDictionaryPredictsUnknownWords runs against a real .dat built by
// this version (GOMORPHY_BENCH_DICT, as for BenchmarkRealDict).
func TestRealDictionaryPredictsUnknownWords(t *testing.T) {
	path := os.Getenv("GOMORPHY_BENCH_DICT")
	if path == "" {
		t.Skip("GOMORPHY_BENCH_DICT not set")
	}
	d, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { _ = d.Close() }()
	for _, w := range []string{"шмуклерами", "шмурдяковый", "перепрокрустить"} {
		assertAllPredicted(t, d, w)
	}
}
