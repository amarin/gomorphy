//go:build integration

package dictionary_test

import (
	"bufio"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/xmlscan"
	"github.com/amarin/gomorphy/pkg/dictionary"
)

// refEntry is one expected reading of a wordform, straight from dict.xml.
type refEntry struct {
	lemma uint32 // dense lemma id in event order
	grams string // merged parse: lemma base tags + form tags
}

// refFeed is an independent xmlscan handler used to build the reference
// data without touching the Builder pipeline.
type refFeed struct {
	onWord func(word string, e refEntry) // called on every form end

	curLemma    uint32
	dense       uint32
	inForm      bool
	lGrams      []string
	fGrams      []string
	lemmaTexts  map[uint32]string
	curFormText string
}

func (f *refFeed) OnGrammeme(_, _ []byte) error { return nil }

func (f *refFeed) OnGrammemeRef(v []byte) error {
	if f.inForm {
		f.fGrams = append(f.fGrams, string(v))
	} else {
		f.lGrams = append(f.lGrams, string(v))
	}

	return nil
}

func (f *refFeed) OnLemma(_ uint32, text []byte) error {
	// Dense ids are zero-based, matching the Builder's assignment order.
	f.curLemma = f.dense
	f.dense++

	f.lemmaTexts[f.curLemma] = string(text)
	f.lGrams = f.lGrams[:0]
	f.inForm = false

	return nil
}

func (f *refFeed) OnLemmaEnd() error { return nil }

func (f *refFeed) OnForm(text []byte) error {
	f.curFormText = string(text)
	f.fGrams = f.fGrams[:0]
	f.inForm = true

	return nil
}

func (f *refFeed) OnFormEnd() error {
	merged := mergeRefGrams(f.lGrams, f.fGrams)

	f.onWord(f.curFormText, refEntry{lemma: f.curLemma, grams: merged})

	f.inForm = false

	return nil
}

// mergeRefGrams concatenates base and form tags dropping duplicates,
// mirroring fullGramNames in pkg/dictionary.
func mergeRefGrams(base, form []string) string {
	seen := make(map[string]struct{}, len(base)+len(form))

	var sb strings.Builder

	for _, part := range [][]string{base, form} {
		for _, g := range part {
			if _, dup := seen[g]; dup {
				continue
			}

			seen[g] = struct{}{}

			if sb.Len() > 0 {
				sb.WriteByte(',')
			}

			sb.WriteString(g)
		}
	}

	return sb.String()
}

func openDictXML(t *testing.T) (*os.File, func()) {
	t.Helper()

	path := os.Getenv("GOMORPHY_DICT_XML")
	if path == "" {
		dir, _ := os.Getwd()

		for {
			candidate := filepath.Join(dir, ".data", "opencorpora", "dict.xml")
			if _, err := os.Stat(candidate); err == nil {
				path = candidate

				break
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				t.Skip("dict.xml not found")
			}

			dir = parent
		}
	}

	f, err := os.Open(path) //nolint:gosec // trusted local path
	if err != nil {
		t.Fatalf("open dict.xml: %v", err)
	}

	return f, func() { _ = f.Close() }
}

const referenceSampleSize = 100

// TestReferenceSample checks Dictionary lookups against an independently
// parsed sample of dict.xml: two streaming passes, deterministic sampling.
func TestReferenceSample(t *testing.T) {
	xmlFile, closeXML := openDictXML(t)
	defer closeXML()

	// Pass 1: collect all distinct surface wordforms.
	words := make(map[string]struct{})

	collector := &refFeed{
		lemmaTexts: map[uint32]string{},
		onWord:     func(word string, _ refEntry) { words[word] = struct{}{} },
	}

	if err := xmlscan.New(bufio.NewReaderSize(xmlFile, 1<<20), collector).Scan(); err != nil {
		t.Fatalf("scan pass1: %v", err)
	}

	keys := make([]string, 0, len(words))
	for w := range words {
		keys = append(keys, w)
	}

	rnd := rand.New(rand.NewSource(42)) //nolint:gosec // deterministic sampling
	rnd.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	sample := make(map[string]bool, referenceSampleSize)
	for _, w := range keys[:referenceSampleSize] {
		sample[w] = true
	}

	t.Logf("dictionary has %d distinct wordforms, sampling %d", len(keys), len(sample))

	// Pass 2: collect expected readings for the sample only.
	expected := make(map[string][]refEntry, referenceSampleSize)

	gatherer := &refFeed{
		lemmaTexts: map[uint32]string{},
		onWord: func(word string, e refEntry) {
			if sample[word] {
				expected[word] = append(expected[word], e)
			}
		},
	}

	if _, err := xmlFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	if err := xmlscan.New(bufio.NewReaderSize(xmlFile, 1<<20), gatherer).Scan(); err != nil {
		t.Fatalf("scan pass2: %v", err)
	}

	closeXML()

	// Compile the real dictionary and cross-check the sample.
	dict, err := dictionary.CompileFromXMLFile(dictXMLPath(t))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = dict.Close() }()

	for w, want := range expected {
		wantSet := dedupEntries(want)

		got, err := dict.Lookup(w)
		if err != nil {
			t.Errorf("%q: lookup: %v", w, err)

			continue
		}

		if len(got) != len(wantSet) {
			t.Errorf("%q: got %d readings, want %d", w, len(got), len(wantSet))

			continue
		}

		for _, f := range got {
			key := refEntryKey(uint32(f.LemmaID), strings.Join(f.Grammemes, ","))
			if !wantSet[key] {
				t.Errorf("%q: unexpected reading lemma=%d grams=%q", w, f.LemmaID, strings.Join(f.Grammemes, ","))

				break
			}
		}

		refs, err := dict.Lemmas(w)
		if err != nil {
			t.Errorf("%q: lemmas: %v", w, err)

			continue
		}

		wantLemmas := make(map[uint32]bool)
		for _, e := range want {
			wantLemmas[e.lemma] = true
		}

		if len(refs) != len(wantLemmas) {
			t.Errorf("%q: got %d lemmas, want %d", w, len(refs), len(wantLemmas))
		}
	}
}

func dedupEntries(entries []refEntry) map[string]bool {
	out := make(map[string]bool, len(entries))

	for _, e := range entries {
		out[refEntryKey(e.lemma, e.grams)] = true
	}

	return out
}

func refEntryKey(lemma uint32, grams string) string {
	return strconv.FormatUint(uint64(lemma), 10) + "|" + grams
}

// dictXMLPath re-resolves the dict.xml path for CompileFromXMLFile.
func dictXMLPath(t *testing.T) string {
	t.Helper()

	if p := os.Getenv("GOMORPHY_DICT_XML"); p != "" {
		return p
	}

	dir, _ := os.Getwd()

	for {
		candidate := filepath.Join(dir, ".data", "opencorpora", "dict.xml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("dict.xml not found")
		}

		dir = parent
	}
}
