package dictionary_test

import (
	"errors"
	"sort"
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

// refLev is a plain O(n*m) rune Levenshtein used as the test oracle.
func refLev(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		cur[0] = i

		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}

		prev, cur = cur, prev
	}

	return prev[len(b)]
}

func TestFuzzyMatchesBruteforceOracle(t *testing.T) {
	words := []string{"кот", "код", "крот", "год", "дом", "дым", "стол", "стул", "ёж", "ёжик", "ежик", "лес"}
	d := buildFuzzyDict(t, words...)

	defer func() { _ = d.Close() }()

	for _, query := range []string{"кот", "стол", "ёжик", "нету"} {
		qr := []rune(query)

		want := map[string]int{}
		for _, w := range words {
			if dist := refLev(qr, []rune(w)); dist <= 2 {
				want[w] = dist
			}
		}

		got, err := d.Fuzzy(query, 2)
		if len(want) == 0 {
			if !errors.Is(err, dictionary.ErrNotFound) {
				t.Fatalf("Fuzzy(%q, 2): err=%v, want ErrNotFound for empty oracle", query, err)
			}

			continue
		}

		if err != nil {
			t.Fatalf("Fuzzy(%q, 2): %v", query, err)
		}

		assertDistEqual(t, distMap(t, got), want)
	}
}

func TestFuzzyTopNearestN(t *testing.T) {
	words := []string{"кот", "код", "крот", "год", "дом", "дым", "стол", "стул", "ёж", "ёжик", "ежик", "лес"}
	d := buildFuzzyDict(t, words...)

	defer func() { _ = d.Close() }()

	qr := []rune("кот")

	sorted := append([]string(nil), words...)
	sort.Slice(sorted, func(i, j int) bool {
		di, dj := refLev(qr, []rune(sorted[i])), refLev(qr, []rune(sorted[j]))
		if di != dj {
			return di < dj
		}

		return sorted[i] < sorted[j]
	})

	t.Run("one word means the exact hit", func(t *testing.T) {
		got, err := d.FuzzyTop("кот", 1)
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 || got[0].Text != "кот" || got[0].Distance != 0 {
			t.Fatalf("FuzzyTop(кот, 1) = %v, want [{кот 0}]", got)
		}
	})

	t.Run("result equals brute-force nearest prefix", func(t *testing.T) {
		for n := 1; n <= len(words); n++ {
			got, err := d.FuzzyTop("кот", n)
			if err != nil {
				t.Fatalf("FuzzyTop(кот, %d): %v", n, err)
			}

			if len(got) != n {
				t.Fatalf("FuzzyTop(кот, %d) returned %d matches", n, len(got))
			}

			distMap(t, got) // also asserts ordering

			for i, m := range got {
				if want := sorted[i]; m.Text != want {
					t.Fatalf("n=%d: position %d = %q, want %q", n, i, m.Text, want)
				}

				if m.Distance != refLev(qr, []rune(m.Text)) {
					t.Fatalf("n=%d: %q distance %d, oracle %d", n, m.Text, m.Distance, refLev(qr, []rune(m.Text)))
				}
			}
		}
	})

	t.Run("larger request returns the whole dictionary", func(t *testing.T) {
		got, err := d.FuzzyTop("кот", len(words)+10)
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != len(words) {
			t.Fatalf("got %d matches, want all %d words", len(got), len(words))
		}
	})
}

func TestFuzzyTopZeroIsExactProbe(t *testing.T) {
	words := []string{"кот", "код", "ёж"}
	d := buildFuzzyDict(t, words...)

	defer func() { _ = d.Close() }()

	got, err := d.FuzzyTop("код", 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].Text != "код" || got[0].Distance != 0 {
		t.Fatalf("FuzzyTop(код, 0) = %v, want [{код 0}]", got)
	}

	if _, err := d.FuzzyTop("крот", 0); !errors.Is(err, dictionary.ErrNotFound) {
		t.Fatalf("absent word probe: err=%v, want ErrNotFound", err)
	}

	if _, err := d.FuzzyTop("кот", -1); !errors.Is(err, dictionary.ErrInvalidMaxWords) {
		t.Fatalf("negative maxWords: err=%v, want ErrInvalidMaxWords", err)
	}
}
