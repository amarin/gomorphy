package morphology_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readingsSnapshot сравнивает результат Parse до и после roundtrip.
func readingsSnapshot(d *morphology.Dictionary, words ...string) map[string][]morphology.Reading {
	out := make(map[string][]morphology.Reading, len(words))
	for _, w := range words {
		for _, r := range d.Parse(w) {
			r.Prob = 0
			out[w] = append(out[w], r)
		}
	}
	return out
}

func cmpSnapshots(t *testing.T, want, got map[string][]morphology.Reading) {
	t.Helper()
	require.Equal(t, len(want), len(got), "набор слов различается")
	for w, wantRs := range want {
		assert.Equalf(t, wantRs, got[w], "разбор слова %q", w)
	}
}

func TestSaveOpenRoundtrip(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	prediction := map[string]uint32{
		"ёнок" + payloadSeparator + b64(predictionValue(3, 2, 1)): 0,
		"ёнка" + payloadSeparator + b64(predictionValue(1, 2, 1)): 0,
	}

	d := buildFixture(t, words, prediction, map[string]uint32{
		"кот:NOUN,anim,masc,sing,nomn": 500,
	})

	out := filepath.Join(t.TempDir(), "pymorphy2.dat")
	require.NoError(t, d.SaveTo(out))

	got, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()

	assert.Equal(t, d.Language(), got.Language())

	cmpSnapshots(t,
		readingsSnapshot(d, "кот", "кота", "мышь", "мыши", "код", "котёнка"),
		readingsSnapshot(got, "кот", "кота", "мышь", "мыши", "код", "котёнка"),
	)

	before := d.Lemma("кота")
	after := got.Lemma("кота")
	assert.Equal(t, before, after)

	// Вероятности из p_t_given_w.intdawg переживают roundtrip.
	wantProb := maxReadingProb(d.Parse("кот"))
	assert.Equal(t, wantProb, maxReadingProb(got.Parse("кот")))
	assert.NotZero(t, wantProb, "фикстура должна содержать вероятность")
}

func maxReadingProb(rs []morphology.Reading) float64 {
	var m float64
	for _, r := range rs {
		if r.Prob > m {
			m = r.Prob
		}
	}
	return m
}

func TestSaveOpenFuzzyIdentical(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)

	out := filepath.Join(t.TempDir(), "d.dat")
	require.NoError(t, d.SaveTo(out))

	got, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()

	assert.Equal(t, d.Fuzzy("кот", 1), got.Fuzzy("кот", 1))
	assert.Equal(t, d.FuzzyTop("кот", 3), got.FuzzyTop("кот", 3))
}

func TestOpenMissingFile(t *testing.T) {
	_, err := morphology.Open(filepath.Join(t.TempDir(), "nope.dat"))
	require.Error(t, err)
}

func TestOpenCorruptedFile(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)
	out := filepath.Join(t.TempDir(), "d.dat")
	require.NoError(t, d.SaveTo(out))

	// Повреждаем байт в конце (ломает checksum).
	f, err := os.OpenFile(out, os.O_RDWR, 0)
	require.NoError(t, err)
	info, err := f.Stat()
	require.NoError(t, err)
	_, err = f.WriteAt([]byte{0xff}, info.Size()-1)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = morphology.Open(out)
	require.Error(t, err)
}

func TestOpenTruncatedFile(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)
	out := filepath.Join(t.TempDir(), "d.dat")
	require.NoError(t, d.SaveTo(out))

	// Обрезаем до заголовка: секции не читаются.
	require.NoError(t, os.Truncate(out, 18))

	_, err := morphology.Open(out)
	require.Error(t, err)
}

func TestDictionaryCloseIdempotent(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)
	out := filepath.Join(t.TempDir(), "d.dat")
	require.NoError(t, d.SaveTo(out))

	got, err := morphology.Open(out)
	require.NoError(t, err)
	require.NoError(t, got.Close())
	require.NoError(t, got.Close())
}

func TestDictionaryCloseNilSafe(t *testing.T) {
	d := buildFixture(t, nil, nil, nil)
	require.NoError(t, d.Close())
}

func TestSaveToBadPath(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)
	err := d.SaveTo(filepath.Join(t.TempDir(), "missing", "x.dat"))
	require.Error(t, err)
}
