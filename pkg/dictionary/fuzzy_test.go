package dictionary_test

import (
	"errors"
	"testing"

	"github.com/amarin/gomorphy/pkg/dictionary"
)

// buildFuzzyDict compiles a small noun-only dictionary for fuzzy tests.
func buildFuzzyDict(t *testing.T, words ...string) *dictionary.Dictionary {
	t.Helper()

	b := dictionary.NewBuilder()

	for _, w := range words {
		id, err := b.AddLemma(w, "NOUN", "inan", "masc")
		if err != nil {
			t.Fatalf("AddLemma(%q): %v", w, err)
		}

		if err := b.AddForm(id, w, "sing", "nomn"); err != nil {
			t.Fatalf("AddForm(%q): %v", w, err)
		}
	}

	return b.Compile()
}

func distMap(t *testing.T, matches []dictionary.FuzzyMatch) map[string]int {
	t.Helper()

	out := make(map[string]int, len(matches))
	prevDist := -1

	var prevText string

	for i, m := range matches {
		out[m.Text] = m.Distance

		if i > 0 && (m.Distance < prevDist || (m.Distance == prevDist && m.Text <= prevText)) {
			t.Errorf("results not ordered by (distance, text): %q:%d after %q:%d",
				m.Text, m.Distance, prevText, prevDist)
		}

		prevDist, prevText = m.Distance, m.Text
	}

	return out
}

func TestFuzzyBasic(t *testing.T) {
	d := buildFuzzyDict(t,
		"кот", "код", "крот", "год", "дом", "дым", "стол", "стул", "ёж", "ёжик", "ежик", "лес")

	defer func() { _ = d.Close() }()

	t.Run("distance one neighbours of кот", func(t *testing.T) {
		got, err := d.Fuzzy("кот", 1)
		if err != nil {
			t.Fatal(err)
		}

		want := map[string]int{"кот": 0, "код": 1, "крот": 1}
		assertDistEqual(t, distMap(t, got), want)
	})

	t.Run("distance two widens the net", func(t *testing.T) {
		got, err := d.Fuzzy("кот", 2)
		if err != nil {
			t.Fatal(err)
		}

		want := map[string]int{"кот": 0, "код": 1, "крот": 1, "год": 2, "дом": 2}
		assertDistEqual(t, distMap(t, got), want)
	})

	t.Run("rune metric treats ё and е as single substitutions", func(t *testing.T) {
		got, err := d.Fuzzy("ежик", 1)
		if err != nil {
			t.Fatal(err)
		}

		want := map[string]int{"ежик": 0, "ёжик": 1}
		assertDistEqual(t, distMap(t, got), want)
	})

	t.Run("ёж to ёжик is two insertions", func(t *testing.T) {
		got, err := d.Fuzzy("ёж", 2)
		if err != nil {
			t.Fatal(err)
		}

		want := map[string]int{"ёж": 0, "ёжик": 2}
		assertDistEqual(t, distMap(t, got), want)
	})

	t.Run("дом and дым are one substitution apart", func(t *testing.T) {
		got, err := d.Fuzzy("дом", 1)
		if err != nil {
			t.Fatal(err)
		}

		want := map[string]int{"дом": 0, "дым": 1}
		assertDistEqual(t, distMap(t, got), want)
	})
}

func TestFuzzyZeroDistanceIsExactSearch(t *testing.T) {
	words := []string{"кот", "код", "крот", "год", "дом", "дым", "стол", "стул", "ёж", "ёжик", "ежик", "лес"}
	d := buildFuzzyDict(t, words...)

	defer func() { _ = d.Close() }()

	for _, w := range words {
		got, err := d.Fuzzy(w, 0)
		if err != nil {
			t.Fatalf("Fuzzy(%q, 0): %v", w, err)
		}

		if len(got) != 1 || got[0].Text != w || got[0].Distance != 0 {
			t.Fatalf("Fuzzy(%q, 0) = %v, want [{%s 0}]", w, got, w)
		}
	}

	if _, err := d.Fuzzy("нетуслова", 0); !errors.Is(err, dictionary.ErrNotFound) {
		t.Fatalf("unknown word at k=0: err=%v, want ErrNotFound", err)
	}
}

func TestFuzzyErrorsAndOrdering(t *testing.T) {
	d := buildFuzzyDict(t, "кот", "код")

	if _, err := d.Fuzzy("кот", -1); !errors.Is(err, dictionary.ErrInvalidMaxDist) {
		t.Fatalf("negative maxDist: err=%v, want ErrInvalidMaxDist", err)
	}

	t.Run("empty query matches by rune length", func(t *testing.T) {
		got, err := d.Fuzzy("", 5)
		if err != nil {
			t.Fatal(err)
		}

		want := map[string]int{"кот": 3, "код": 3}
		assertDistEqual(t, distMap(t, got), want)
	})

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := d.Fuzzy("кот", 1); !errors.Is(err, dictionary.ErrClosed) {
		t.Fatalf("closed dict: err=%v, want ErrClosed", err)
	}
}

func assertDistEqual(t *testing.T, got, want map[string]int) {
	t.Helper()

	for text, dist := range want {
		gd, ok := got[text]
		if !ok {
			t.Errorf("missing match %q (want distance %d)", text, dist)

			continue
		}

		if gd != dist {
			t.Errorf("match %q: got distance %d, want %d", text, gd, dist)
		}
	}

	for text := range got {
		if _, ok := want[text]; !ok {
			t.Errorf("unexpected match %q (distance %d)", text, got[text])
		}
	}
}
