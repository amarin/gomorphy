package morphology_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

// TestSaveToStampsBuildInfo проверяет, что SaveTo проставляет BuiltAt и
// LibraryVersion в секцию info при каждом сохранении, сохраняя то, что уже
// заполнил импортёр (Source), и не изменяя исходный Dictionary (он
// иммутабелен — buildFixture идёт через OpenPyMorphy, который выставляет
// Info.Source="pymorphy2", но не BuiltAt/LibraryVersion).
func TestSaveToStampsBuildInfo(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)

	before := d.Info()
	require.NotNil(t, before, "buildFixture идёт через OpenPyMorphy — Info.Source должен быть заполнен")
	assert.Equal(t, "pymorphy2", before.Source)
	assert.Zero(t, before.BuiltAt, "BuiltAt не должен быть заполнен до SaveTo")
	assert.Empty(t, before.LibraryVersion, "LibraryVersion не должен быть заполнен до SaveTo")

	saveStart := time.Now().UTC()
	out := filepath.Join(t.TempDir(), "d.dat")
	require.NoError(t, d.SaveTo(out))
	saveEnd := time.Now().UTC()

	assert.Zero(t, d.Info().BuiltAt, "SaveTo не должна мутировать исходный Dictionary")

	got, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()

	info := got.Info()
	require.NotNil(t, info)
	assert.Equal(t, "pymorphy2", info.Source, "Source, заполненный импортёром, должен пережить SaveTo/Open")
	assert.Equal(t, morphology.Version, info.LibraryVersion)
	assert.False(t, info.BuiltAt.Before(saveStart), "BuiltAt раньше начала сохранения")
	assert.False(t, info.BuiltAt.After(saveEnd), "BuiltAt позже окончания сохранения")
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

// SaveTo creates missing parent directories rather than failing, so a long
// compile isn't lost to a forgotten `mkdir -p` on the output path.
func TestSaveToCreatesMissingDir(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)
	out := filepath.Join(t.TempDir(), "missing", "nested", "x.dat")
	require.NoError(t, d.SaveTo(out))

	_, err := os.Stat(out)
	require.NoError(t, err)
}

// TestSaveToRejectsDenseAlphabet проверяет находку финального ревью: словарь
// с ненулевым Alphabet (OpenPyMorphyDense) нельзя сохранять через SaveTo —
// у Alphabet нет дискового представления, и сохранённый-и-переоткрытый
// словарь тихо мис-декодировался бы (см. docs/superpowers/specs/
// 2026-09-16-pymorphy2-dense-recompile-design.md, non-goals).
func TestSaveToRejectsDenseAlphabet(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)

	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	out := filepath.Join(t.TempDir(), "dense.dat")
	err = dense.SaveTo(out)
	require.Error(t, err)

	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr), "SaveTo не должна создавать файл при отказе")
}

// TestSaveToNilAlphabetStillWorks — регрессия: обычный (не dense) словарь
// по-прежнему сохраняется без ошибок.
func TestSaveToNilAlphabetStillWorks(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)

	out := filepath.Join(t.TempDir(), "plain.dat")
	require.NoError(t, d.SaveTo(out))

	_, err := os.Stat(out)
	require.NoError(t, err)
}

func TestSaveToBadPath(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	d := buildFixture(t, words, nil, nil)

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := d.SaveTo(filepath.Join(blocker, "missing", "x.dat"))
	require.Error(t, err)
}
