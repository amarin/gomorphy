package dictionary_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/amarin/gomorphy/pkg/dictionary"
)

func mustBuildDict(t *testing.T) (*dictionary.Dictionary, string) {
	t.Helper()

	b := dictionary.NewBuilder()

	id, err := b.AddLemma("кот", "NOUN", "anim", "masc")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		text string
		gr   []string
	}{
		{"кота", []string{"NOUN", "anim", "masc", "gent"}},
		{"коту", []string{"NOUN", "anim", "masc", "datv"}},
	} {
		if err := b.AddForm(id, tc.text, tc.gr...); err != nil {
			t.Fatal(err)
		}
	}

	id2, err := b.AddLemma("пила", "NOUN", "inanim", "femn")
	if err != nil {
		t.Fatal(err)
	}

	if err := b.AddForm(id2, "пилу", []string{"NOUN", "inanim", "femn", "accs"}...); err != nil {
		t.Fatal(err)
	}

	d := b.Compile()

	path := filepath.Join(t.TempDir(), "test.dict")
	if err := d.SaveTo(path); err != nil {
		t.Fatal(err)
	}

	return d, path
}

func TestOpenSaveRoundtrip(t *testing.T) {
	_, path := mustBuildDict(t)

	d, err := dictionary.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = d.Close() }()

	forms, err := d.Lookup("кота")
	if err != nil {
		t.Fatal(err)
	}

	if len(forms) != 1 || forms[0].Ancode != "NOUN,anim,masc,gent" || forms[0].Text != "кота" {
		t.Fatalf("unexpected forms: %+v", forms)
	}
}

func TestLookupNotFound(t *testing.T) {
	d, _ := mustBuildDict(t)

	defer func() { _ = d.Close() }()

	if _, err := d.Lookup("несуществующее"); !errors.Is(err, dictionary.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestLemmas(t *testing.T) {
	d, _ := mustBuildDict(t)

	defer func() { _ = d.Close() }()

	refs, err := d.Lemmas("кота")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 1 || refs[0].ID != 0 || refs[0].Text != "кот" {
		t.Fatalf("unexpected lemmas: %+v", refs)
	}
}

func TestNewEmptyAndSaveOpen(t *testing.T) {
	d := dictionary.NewEmpty()

	path := filepath.Join(t.TempDir(), "empty.dict")
	if err := d.SaveTo(path); err != nil {
		t.Fatal(err)
	}

	if _, err := dictionary.Open(path); err != nil {
		t.Fatalf("open empty: %v", err)
	}

	if _, err := d.Lookup("кот"); !errors.Is(err, dictionary.ErrNotFound) {
		t.Fatalf("empty lookup: want ErrNotFound, got %v", err)
	}
}

func TestClosedDictionary(t *testing.T) {
	d, _ := mustBuildDict(t)

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	if err := d.Close(); err != nil { // idempotent
		t.Fatal(err)
	}

	for name, call := range map[string]func() error{
		"lookup":  func() error { _, err := d.Lookup("кот"); return err },
		"lemmas":  func() error { _, err := d.Lemmas("кот"); return err },
		"fuzzy":   func() error { _, err := d.Fuzzy("кот", 1); return err },
		"save":    func() error { return d.SaveTo(filepath.Join(t.TempDir(), "x")) },
		"builder": func() error { _, err := d.Builder(); return err },
	} {
		if err := call(); !errors.Is(err, dictionary.ErrClosed) {
			t.Errorf("%s after Close: want ErrClosed, got %v", name, err)
		}
	}
}

// TestConcurrentReads verifies FT2/FT5 reads are race-free under -race.
func TestConcurrentReads(t *testing.T) {
	_, path := mustBuildDict(t)

	d, err := dictionary.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = d.Close() }()

	var wg sync.WaitGroup

	errs := make(chan error, 64)

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func(g int) {
			defer wg.Done()

			words := []string{"кота", "коту", "пилу"}

			for i := 0; i < 200; i++ {
				w := words[(g+i)%len(words)]

				forms, err := d.Lookup(w)
				if err != nil {
					errs <- fmt.Errorf("lookup %q: %w", w, err)

					return
				}

				if len(forms) == 0 {
					errs <- fmt.Errorf("lookup %q: no forms", w)

					return
				}

				if _, err := d.Lemmas(w); err != nil {
					errs <- fmt.Errorf("lemmas %q: %w", w, err)

					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestParallelInstances verifies two independent dictionaries do not share
// state (FT9): separate files, separate mappings, simultaneous use.
func TestParallelInstances(t *testing.T) {
	_, pathA := mustBuildDict(t)
	_, pathB := mustBuildDict(t)

	da, err := dictionary.Open(pathA)
	if err != nil {
		t.Fatal(err)
	}

	db, err := dictionary.Open(pathB)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = da.Close()
		_ = db.Close()
	}()

	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				_, _ = da.Lookup("кота")
			}
		}()

		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				_, _ = db.Lookup("пилу")
			}
		}()
	}

	wg.Wait()
}

func TestBuilderReconstruction(t *testing.T) {
	src, _ := mustBuildDict(t)

	defer func() { _ = src.Close() }()

	b, err := src.Builder()
	if err != nil {
		t.Fatal(err)
	}

	rebuilt := b.Compile()

	for _, w := range []string{"кот", "кота", "коту", "пила", "пилу"} {
		aForms, errA := src.Lookup(w)
		bForms, errB := rebuilt.Lookup(w)

		if (errA == nil) != (errB == nil) {
			t.Fatalf("word %q: err mismatch %v vs %v", w, errA, errB)
		}

		if errA != nil {
			continue
		}

		if len(aForms) != len(bForms) {
			t.Fatalf("word %q: %d forms vs %d", w, len(aForms), len(bForms))
		}

		for i := range aForms {
			if aForms[i].Text != bForms[i].Text || aForms[i].Ancode != bForms[i].Ancode {
				t.Fatalf("word %q form %d: %+v vs %+v", w, i, aForms[i], bForms[i])
			}
		}

		aRefs, _ := src.Lemmas(w)
		bRefs, _ := rebuilt.Lemmas(w)

		if len(aRefs) != len(bRefs) {
			t.Fatalf("word %q: %d lemmas vs %d", w, len(aRefs), len(bRefs))
		}
	}
}

// mustBuildHomonyms creates a dictionary with classic Russian homonyms:
// "стекла" (стекло/стечь), "пила" (пила/пить) and the unique "кот".
func mustBuildHomonyms(t *testing.T) *dictionary.Dictionary {
	t.Helper()

	b := dictionary.NewBuilder()

	add := func(lemma, form string, lGr, fGr []string) {
		id, err := b.AddLemma(lemma, lGr...)
		if err != nil {
			t.Fatal(err)
		}

		if err := b.AddForm(id, form, fGr...); err != nil {
			t.Fatal(err)
		}
	}

	add("стекло", "стекла", []string{"NOUN", "inan", "neut"}, []string{"sing", "gent"})
	add("стечь", "стекла", []string{"VERB", "impf", "intr"}, []string{"past", "neut"})
	add("пила", "пилу", []string{"NOUN", "inan", "femn"}, []string{"sing", "accs"})
	add("пить", "пила", []string{"VERB", "impf", "tran"}, []string{"past", "femn"})
	add("кот", "кота", []string{"NOUN", "anim", "masc"}, []string{"sing", "gent"})

	return b.Compile()
}

func TestHomonymLemmas(t *testing.T) {
	d := mustBuildHomonyms(t)

	defer func() { _ = d.Close() }()

	for _, tc := range []struct {
		word     string
		wantRefs int
		texts    []string // expected citation texts among refs
	}{
		{"стекла", 2, []string{"стекло", "стечь"}},
		{"пила", 2, []string{"пила", "пить"}},
		{"кот", 1, []string{"кот"}},
	} {
		refs, err := d.Lemmas(tc.word)
		if err != nil {
			t.Fatalf("%q: %v", tc.word, err)
		}

		if len(refs) < tc.wantRefs {
			t.Errorf("%q: want >=%d lemmas, got %d: %+v", tc.word, tc.wantRefs, len(refs), refs)

			continue
		}

		gotTexts := make(map[string]bool)
		for _, r := range refs {
			gotTexts[r.Text] = true

			if len(r.Grammemes) == 0 {
				t.Errorf("%q: lemma %q has empty base grammemes", tc.word, r.Text)
			}
		}

		for _, want := range tc.texts {
			if !gotTexts[want] {
				t.Errorf("%q: lemma %q missing among %v", tc.word, want, gotTexts)
			}
		}
	}
}

func TestFullGrammarMerge(t *testing.T) {
	d := mustBuildHomonyms(t)

	defer func() { _ = d.Close() }()

	forms, err := d.Lookup("кота")
	if err != nil {
		t.Fatal(err)
	}

	if len(forms) != 1 {
		t.Fatalf("want 1 form, got %+v", forms)
	}

	f := forms[0]

	want := []string{"NOUN", "anim", "masc", "sing", "gent"}
	if len(f.Grammemes) != len(want) {
		t.Fatalf("grammemes %+v, want %v", f.Grammemes, want)
	}

	for i := range want {
		if f.Grammemes[i] != want[i] {
			t.Fatalf("grammemes %+v, want %v", f.Grammemes, want)
		}
	}

	if f.Ancode != "NOUN,anim,masc,sing,gent" {
		t.Errorf("ancode %q", f.Ancode)
	}
}
