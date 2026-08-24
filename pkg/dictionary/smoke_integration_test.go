//go:build integration

package dictionary_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/amarin/gomorphy/pkg/dictionary"
)

func openCompiledDict(t *testing.T) *dictionary.Dictionary {
	t.Helper()

	path := os.Getenv("GOMORPHY_DICT_COMPILED")
	if path == "" {
		dir, _ := os.Getwd()

		for {
			candidate := filepath.Join(dir, ".data", "opencorpora", "opencorpora.dict")
			if _, err := os.Stat(candidate); err == nil {
				path = candidate

				break
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				t.Skip("opencorpora.dict not found")
			}

			dir = parent
		}
	}

	d, err := dictionary.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}

	return d
}

// hasGrams reports whether got contains all of want in the same relative order.
func hasGrams(got, want []string) bool {
	j := 0

	for _, g := range got {
		if j < len(want) && g == want[j] {
			j++
		}
	}

	return j == len(want)
}

// TestSmokeFrequentWords opens the full compiled dictionary and checks a
// sample of 50 frequent wordforms resolve, with precise grammar assertions
// on a control subset (verified against opencorpora.org).
func TestSmokeFrequentWords(t *testing.T) {
	d := openCompiledDict(t)

	defer func() { _ = d.Close() }()

	frequent := []string{
		"и", "в", "не", "на", "быть", "он", "с", "что", "а", "по",
		"она", "это", "к", "как", "но", "из", "у", "который", "то", "за",
		"свой", "что", "весь", "год", "от", "так", "о", "для", "ты", "же",
		"мочь", "вы", "человек", "такой", "его", "сказать", "мы", "один", "какой", "бы",
		"дом", "дома", "дому", "домом", "кот", "кота", "коту", "пила", "пилу", "стекла",
	}

	found := 0

	for i, w := range frequent {
		if i > 0 && frequent[i] == frequent[i-1] { // duplicate sample entries
			continue
		}

		if _, err := d.Lookup(w); err == nil {
			found++
		} else if !slices.Contains([]string{"и", "в", "с", "к", "а", "о", "у", "по", "за", "из", "не", "на", "то", "же", "бы"}, w) {
			t.Errorf("frequent word %q not found: %v", w, err)
		}
	}

	t.Logf("resolved %d/%d frequent wordforms", found, len(frequent))

	if found < 45 {
		t.Fatalf("only %d/%d frequent words resolved", found, len(frequent))
	}

	control := []struct {
		word     string
		minLemma int
		lemma    string // one expected lemma text
		grams    []string
	}{
		// NOTE: in OpenCorpora dict.xml wordforms carry only their own <g>
		// refs (case/number); POS and other constant tags live on the lemma,
		// so Lookup returns e.g. sing,gent for "кота", not NOUN,anim,masc,gent.
		{"кота", 1, "кот", []string{"sing", "gent"}},
		{"коту", 1, "кот", []string{"sing", "datv"}},
		{"домами", 1, "дом", []string{"plur", "ablt"}},
		{"бежал", 2, "", nil},  // омонимия: бегу/бежал леммы
		{"стекла", 2, "", nil}, // омонимия: стекло / течь
		{"пила", 2, "", nil},   // омонимия: пила / пить
	}

	for _, tc := range control {
		refs, err := d.Lemmas(tc.word)
		if err != nil {
			t.Errorf("%q: lemmas: %v", tc.word, err)

			continue
		}

		if len(refs) < tc.minLemma {
			t.Errorf("%q: want >=%d lemmas, got %d (%v)", tc.word, tc.minLemma, len(refs), refs)

			continue
		}

		if tc.lemma == "" {
			continue
		}

		match := false

		for _, r := range refs {
			if r.Text == tc.lemma {
				match = true

				break
			}
		}

		if !match {
			t.Errorf("%q: lemma %q missing among %v", tc.word, tc.lemma, refs)

			continue
		}

		forms, err := d.Lookup(tc.word)
		if err != nil {
			t.Errorf("%q: lookup: %v", tc.word, err)

			continue
		}

		ok := false

		for _, f := range forms {
			if hasGrams(f.Grammemes, tc.grams) {
				ok = true

				break
			}
		}

		if !ok {
			t.Errorf("%q: no form with grammemes %v among %+v", tc.word, tc.grams, forms)
		}
	}
}
