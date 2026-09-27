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

func TestCompileFromXMLPredictsByDefault(t *testing.T) {
	for name, compile := range map[string]func() (*morphology.Dictionary, error){
		"CompileFromXML": func() (*morphology.Dictionary, error) {
			return morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
		},
		"CompileFromXMLDense": func() (*morphology.Dictionary, error) {
			return morphology.CompileFromXMLDense(strings.NewReader(exampleDictXML), nil)
		},
	} {
		d, err := compile()
		require.NoError(t, err, name)
		assertAllPredicted(t, d, "бота") // like кота
	}

	d, err := morphology.CompileFromXMLWithOptions(strings.NewReader(exampleDictXML),
		morphology.XMLOptions{Dense: true, NoPrediction: true})
	require.NoError(t, err)
	assert.Nil(t, d.Parse("бота"), "NoPrediction keeps the 1.2 behaviour")
	assert.NotEmpty(t, d.Parse("кота"))
}

func TestCompileFromUniMorphPredictsByDefault(t *testing.T) {
	opts := morphology.UniMorphOptions{Language: "ru"}
	d, err := morphology.CompileFromUniMorphDense(strings.NewReader(uniMorphTSV), opts)
	require.NoError(t, err)
	assertAllPredicted(t, d, "бота")

	opts.NoPrediction = true
	d, err = morphology.CompileFromUniMorphDense(strings.NewReader(uniMorphTSV), opts)
	require.NoError(t, err)
	assert.Nil(t, d.Parse("бота"))
}

// UniMorph rus has five parts of speech, all open classes; productive()
// splits on "," and must filter none of them (spec L, "Tag filter").
func TestCompileFromUniMorphPredictsAllPOS(t *testing.T) {
	tsv := "стол\tстолами\tN;INS;PL\n" +
		"синий\tсиними\tADJ;INS;PL\n" +
		"читать\tчитали\tV;PST;PL\n" +
		"читать\tчитавшими\tV.PTCP;ACT;PST;INS;PL\n" +
		"читать\tчитая\tV.CVB;PRS\n"
	d, err := morphology.CompileFromUniMorph(strings.NewReader(tsv), morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)

	pos := map[string]bool{}
	for _, w := range []string{"стульями", "красными", "писали", "писавшими", "пиная"} {
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
	for _, w := range []string{"кракозябрами", "шмурдяковый", "перепрокрустить"} {
		assertAllPredicted(t, d, w)
	}
}
