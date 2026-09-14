package morphology_test

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/require"
)

// payloadSeparator — протокольный разделитель слово/payload в words.dawg.
const payloadSeparator = "\x01"

func b64(p []byte) string { return base64.StdEncoding.EncodeToString(p) }

func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeParadigms(t *testing.T, dir string, paradigms [][]uint16) {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint16(len(paradigms))))
	for _, p := range paradigms {
		require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint16(len(p))))
		for _, v := range p {
			require.NoError(t, binary.Write(&buf, binary.LittleEndian, v))
		}
	}
	writeFile(t, dir, "paradigms.array", buf.Bytes())
}

// readingValue — payload words.dawg: uint16 BE (para) + uint16 BE (form).
func readingValue(para, form uint16) []byte {
	return []byte{byte(para >> 8), byte(para), byte(form >> 8), byte(form)}
}

// predictionValue — payload prediction: uint16 BE (count) + uint16 BE (para) +
// uint16 BE (form).
func predictionValue(count int, para, form uint16) []byte {
	return []byte{
		byte(count >> 8), byte(count),
		byte(para >> 8), byte(para),
		byte(form >> 8), byte(form),
	}
}

// addWord добавляет чтение (word, парадигма, форма) в words.dawg.
func addWord(m map[string]uint32, word string, para, form uint16) {
	m[word+payloadSeparator+b64(readingValue(para, form))] = 0
}

// addPrediction добавляет суффикс preduction в prediction-suffixes-0.dawg:
// ключ "суффикс\x01<base64(count, para, form)>".
func addPrediction(m map[string]uint32, suffix string, count int, para, form uint16) {
	m[suffix+payloadSeparator+b64(predictionValue(count, para, form))] = 0
}

// stdWords наполняет словарь опорными словоформами:
// п.0 «кот» NOUN (nomn/gent), п.1 «кот» VERB, п.2 «мышь»/«код»/… NOUN.
func stdWords(m map[string]uint32) {
	addWord(m, "кот", 0, 0)
	addWord(m, "кот", 1, 0)
	addWord(m, "кота", 0, 1)
	addWord(m, "мышь", 2, 0)
	addWord(m, "мыши", 2, 1)
	for _, w := range []string{"код", "крот", "год", "дом", "дым", "стол", "стул", "ёж", "ёжик", "ежик", "лес"} {
		addWord(m, w, 2, 0)
	}
}

// buildFixture собирает директорию pymorphy2 в t.TempDir() и открывает её.
// words — ключи words.dawg; prediction — prediction-suffixes-0; prob —
// p_t_given_w.intdawg (nil — файл не пишется).
func buildFixture(t *testing.T, words, prediction, prob map[string]uint32) *morphology.Dictionary {
	t.Helper()
	dir := t.TempDir()

	writeParadigms(t, dir, [][]uint16{
		{0, 1, 0, 1, 0, 0}, // п.0: кот NOUN — ""(nomn) / "а"(gent)
		{0, 2, 0},          // п.1: кот VERB — ""(VERB)
		{0, 1, 0, 1, 0, 0}, // п.2: мышь NOUN
	})
	writeFile(t, dir, "suffixes.json", []byte(`["","а"]`))
	writeFile(t, dir, "paradigm-prefixes.json", []byte(`["","по","наи"]`))
	writeFile(t, dir, "gramtab-opencorpora-int.json", []byte(
		`["NOUN,anim,masc,sing,nomn","NOUN,anim,masc,sing,gent","VERB,impf,trans"]`,
	))

	wordsDAWG, guide := testdawg.Build(words)
	writeFile(t, dir, "words.dawg", testdawg.Marshal(wordsDAWG, guide))

	if len(prediction) > 0 {
		pDAWG, pGuide := testdawg.Build(prediction)
		writeFile(t, dir, "prediction-suffixes-0.dawg", testdawg.Marshal(pDAWG, pGuide))
	}
	if len(prob) > 0 {
		prDAWG, prGuide := testdawg.Build(prob)
		writeFile(t, dir, "p_t_given_w.intdawg", testdawg.Marshal(prDAWG, prGuide))
	}

	d, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	require.NotNil(t, d)
	return d
}
