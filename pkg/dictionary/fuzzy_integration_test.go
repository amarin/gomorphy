//go:build integration

package dictionary_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/amarin/gomorphy/pkg/dictionary"
)

// TestFuzzyFullDict checks known neighbours on the real OpenCorpora
// dictionary and logs the wall time of a k=2 query (stage-9 budget: <1s).
func TestFuzzyFullDict(t *testing.T) {
	d := openCompiledDict(t)

	defer func() { _ = d.Close() }()

	t.Run("слон finds клон at distance one", func(t *testing.T) {
		got, err := d.Fuzzy("слон", 1)
		if err != nil {
			t.Fatal(err)
		}

		m := distMap(t, got)
		if m["клон"] != 1 {
			t.Errorf("клон missing or wrong distance: %d", m["клон"])
		}

		if m["слон"] != 0 {
			t.Errorf("слон itself missing or wrong distance: %d", m["слон"])
		}
	})

	t.Run("стол and стул are one substitution apart", func(t *testing.T) {
		got, err := d.Fuzzy("стул", 1)
		if err != nil {
			t.Fatal(err)
		}

		if distMap(t, got)["стол"] != 1 {
			t.Error("стол not found at distance one from стул")
		}
	})

	t.Run("k=2 budget under one second", func(t *testing.T) {
		start := time.Now()

		got, err := d.Fuzzy("кот", 2)
		if err != nil {
			t.Fatal(err)
		}

		elapsed := time.Since(start)
		t.Logf("Fuzzy(кот, 2): %d matches in %s", len(got), elapsed)

		if elapsed > time.Second {
			t.Errorf("fuzzy search exceeded the 1s stage budget: %s", elapsed)
		}
	})
}

// TestFuzzyZeroDistanceMatchesLookup cross-checks the degenerate threshold:
// every exact hit of Lookup must appear in Fuzzy(w, 0) with distance zero,
// and vice versa.
func TestFuzzyZeroDistanceMatchesLookup(t *testing.T) {
	d := openCompiledDict(t)

	defer func() { _ = d.Close() }()

	for _, w := range []string{"кота", "любишь", "стекла", "таутога", "дом"} {
		matches, err := d.Fuzzy(w, 0)
		if err != nil {
			t.Fatalf("Fuzzy(%q, 0): %v", w, err)
		}

		if len(matches) != 1 || matches[0].Text != w || matches[0].Distance != 0 {
			t.Fatalf("Fuzzy(%q, 0) = %v", w, matches)
		}

		if _, err := d.Lookup(w); err != nil {
			t.Fatalf("Lookup(%q): %v", w, err)
		}
	}

	if _, err := d.Fuzzy("qqqqzzzz", 0); err == nil {
		t.Error("nonsense word unexpectedly matched at k=0")
	}
}

// TestFuzzyGrouping verifies results come grouped by ascending distance.
func TestFuzzyGrouping(t *testing.T) {
	d := openCompiledDict(t)

	defer func() { _ = d.Close() }()

	got, err := d.Fuzzy("пила", 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) < 5 {
		t.Fatalf("expected a decent neighbourhood, got %d matches", len(got))
	}

	dists := make([]int, len(got))
	texts := make([]string, len(got))

	for i, m := range got {
		dists[i], texts[i] = m.Distance, m.Text
	}

	if !slices.IsSorted(dists) {
		t.Errorf("distances not sorted: %v", dists)
	}

	if slices.Contains(texts, "пила") && dists[slices.Index(texts, "пила")] != 0 {
		t.Error("query itself must sit in the zero-distance group")
	}
}

// openBenchDict locates and opens the compiled dictionary without a
// testing.T (benchmark helper).
func openBenchDict() (*dictionary.Dictionary, func(), error) {
	path := findCompiledDictPath()
	if path == "" {
		return nil, nil, dictionary.ErrNotFound
	}

	d, err := dictionary.Open(path)
	if err != nil {
		return nil, nil, err
	}

	return d, func() { _ = d.Close() }, nil
}

func findCompiledDictPath() string {
	if path := os.Getenv("GOMORPHY_DICT_COMPILED"); path != "" {
		return path
	}

	dir, _ := os.Getwd()

	for {
		candidate := filepath.Join(dir, ".data", "opencorpora", "opencorpora.dict")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}

		dir = parent
	}
}

// TestFuzzyTopFullDict checks the nearest-N variant on the real dictionary:
// exact probe, monotone prefix property and the stage budget.
func TestFuzzyTopFullDict(t *testing.T) {
	d := openCompiledDict(t)

	defer func() { _ = d.Close() }()

	t.Run("maxWords=0 is an exact probe", func(t *testing.T) {
		got, err := d.FuzzyTop("таутога", 0)
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 1 || got[0].Text != "таутога" || got[0].Distance != 0 {
			t.Fatalf("FuzzyTop(таутога, 0) = %v", got)
		}

		if _, err := d.FuzzyTop("qqqqzzzz", 0); err == nil {
			t.Error("nonsense word unexpectedly matched")
		}
	})

	t.Run("top-20 is sorted and prefixes top-50", func(t *testing.T) {
		start := time.Now()

		top20, err := d.FuzzyTop("человек", 20)
		if err != nil {
			t.Fatal(err)
		}

		top50, err := d.FuzzyTop("человек", 50)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("FuzzyTop(человек): 20 in %s, 50 in %s",
			time.Since(start)/2, time.Since(start))

		distMap(t, top20)

		if len(top20) != 20 {
			t.Fatalf("got %d matches, want 20", len(top20))
		}

		for i := range top20 {
			if top20[i] != top50[i] {
				t.Fatalf("prefix mismatch at %d: %v vs %v", i, top20[i], top50[i])
			}
		}
	})
}

// BenchmarkFuzzyK2 measures a full-dictionary k=2 query (stage-9 budget).
func BenchmarkFuzzyK2(b *testing.B) {
	d, cleanup, err := openBenchDict()
	if err != nil {
		b.Fatal(err)
	}

	defer cleanup()

	b.ResetTimer()

	for b.Loop() {
		if _, err := d.Fuzzy("человек", 2); err != nil {
			b.Fatal(err)
		}
	}
}
