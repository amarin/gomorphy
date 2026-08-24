package build_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/amarin/gomorphy/internal/build"
)

// genWords produces n pseudo-random words over a small alphabet, ensuring
// heavy prefix sharing to stress the trie.
func genWords(seed int64, n int) []string {
	r := rand.New(rand.NewSource(seed))

	prefixes := []string{"ко", "по", "при", "на", "за", "ст", "про"}
	suffixes := []string{"т", "ла", "лить", "ка", "но", "стье", "жик"}

	words := make([]string, 0, n)
	seen := map[string]bool{}

	for len(words) < n {
		w := prefixes[r.Intn(len(prefixes))] + suffixes[r.Intn(len(suffixes))]
		if r.Intn(3) == 0 {
			w += fmt.Sprintf("%d", r.Intn(100))
		}

		if seen[w] {
			continue
		}

		seen[w] = true
		words = append(words, w)
	}

	return words
}

func TestPropertyRandomWords(t *testing.T) {
	const (
		nLemmas = 2000
		nProbes = 5000
	)

	words := genWords(42, nLemmas)

	b := build.NewBuilder()

	lemmaIDs := make([]int, len(words))
	for i, w := range words {
		id, err := b.AddLemma(w, "NOUN")
		if err != nil {
			t.Fatalf("AddLemma(%q): %v", w, err)
		}

		lemmaIDs[i] = id

		if err := b.AddForm(id, w+"а", "NOUN", "gent"); err != nil {
			t.Fatal(err)
		}
	}

	snap := b.Build()

	if snap.LemmaCount() != nLemmas {
		t.Fatalf("lemmas: got %d want %d", snap.LemmaCount(), nLemmas)
	}

	// Every inserted word (base and form) must be found with correct lemma.
	for i, w := range words {
		state, ok := snap.LookupWord([]byte(w))
		if !ok {
			t.Fatalf("word %q not found", w)
		}

		if !hasLemma(snap, state, uint32(lemmaIDs[i])) {
			t.Fatalf("word %q: lemma %d missing from postings", w, lemmaIDs[i])
		}

		if _, ok := snap.LookupWord([]byte(w + "а")); !ok {
			t.Fatalf("form %q not found", w+"а")
		}
	}

	// Absent words must not be found: mutate known words.
	r := rand.New(rand.NewSource(7))

	for i := 0; i < 1000; i++ {
		w := words[r.Intn(len(words))]
		mutated := "щ" + w + "щ"

		if _, ok := snap.LookupWord([]byte(mutated)); ok {
			t.Fatalf("absent word %q found", mutated)
		}
	}
}

func hasLemma(snap *build.Snapshot, state uint32, want uint32) bool {
	for _, p := range snap.PostingsOf(state) {
		for _, l := range snap.PairLemmasOf(p) {
			if l == want {
				return true
			}
		}
	}

	return false
}

func TestConcurrentReads(t *testing.T) {
	snap := mustBuildSmall(t)

	var wg sync.WaitGroup

	for g := 0; g < 8; g++ {
		wg.Add(1)

		go func(g int) {
			defer wg.Done()

			for i := 0; i < 2000; i++ {
				state, ok := snap.LookupWord([]byte("пила"))
				if !ok || !snap.IsFinal(state) {
					t.Error("concurrent lookup failed")

					return
				}

				_ = snap.PostingsOf(state)

				if n := snap.LemmaCount(); n > 0 {
					_ = snap.Text(snap.LemmaTexts[uint32(g)%uint32(n)])
				}
			}
		}(g)
	}

	wg.Wait()
}
