package dictionary

import (
	"sort"
	"unicode/utf8"

	"github.com/amarin/gomorphy/internal/build"
)

// Fuzzy returns dictionary words within maxDist edit distance of word
// (FT6), ordered by (distance, text). The distance is Levenshtein over
// runes — insertion, deletion and substitution of one character each, no
// transpositions. The whole CSR trie is walked with a banded DP row: a
// subtree is cut as soon as every its completion is provably beyond k.
//
// A word that exists in the dictionary but has no neighbours within k
// yields ErrNotFound, mirroring Lookup semantics; use maxDist=0 for an
// exact-match probe.
func (d *Dictionary) Fuzzy(word string, maxDist int) ([]FuzzyMatch, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}

	if maxDist < 0 {
		return nil, ErrInvalidMaxDist
	}

	out := walkTrie(d.snap, word, maxDist)
	if len(out) == 0 {
		return nil, ErrNotFound
	}

	sortMatches(out)

	return out, nil
}

// FuzzyTop returns up to maxWords dictionary words nearest to word (FT6),
// ordered by (distance, text) so the cut at the boundary is deterministic.
// The threshold widens iteratively from distance zero until enough words
// are collected or the dictionary is exhausted — the result is always the
// true nearest neighbourhood, not whatever happened to fall under some k.
//
// maxWords=0 degenerates into an exact-match probe: the word itself when
// present, ErrNotFound otherwise. Negative values are rejected with
// ErrInvalidMaxWords.
func (d *Dictionary) FuzzyTop(word string, maxWords int) ([]FuzzyMatch, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}

	if maxWords < 0 {
		return nil, ErrInvalidMaxWords
	}

	if maxWords == 0 {
		return d.Fuzzy(word, 0)
	}

	total := d.snap.TextCount()
	found := make([]FuzzyMatch, 0, min(maxWords, total))
	seen := make(map[string]struct{}, maxWords)

	for dist := 0; len(found) < maxWords && len(seen) < total; dist++ {
		for _, m := range walkTrie(d.snap, word, dist) {
			if _, dup := seen[m.Text]; dup {
				continue
			}

			seen[m.Text] = struct{}{}
			found = append(found, m)

			if len(found) == maxWords {
				break
			}
		}
	}

	if len(found) == 0 {
		return nil, ErrNotFound
	}

	sortMatches(found)

	return found, nil
}

// walkTrie runs one banded-DP pass over the trie and returns unsorted hits
// within k. Every hit carries its exact distance (≤ k), so successive
// widening passes in FuzzyTop only surface genuinely new words.
func walkTrie(s *build.Snapshot, word string, k int) []FuzzyMatch {
	f := fuzzySearch{
		s:    s,
		q:    []rune(word),
		k:    k,
		path: make([]byte, 0, 32),
	}

	row := f.rowFor(0)
	for j := range row {
		row[j] = j
	}

	f.visit(0, 0, row)

	return f.out
}

func sortMatches(out []FuzzyMatch) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}

		return out[i].Text < out[j].Text
	})
}

// fuzzySearch carries the state of one Fuzzy walk. rows[depth] holds the DP
// row after consuming `depth` query runes; path accumulates the bytes of the
// current trie walk.
type fuzzySearch struct {
	s    *build.Snapshot
	q    []rune
	k    int
	rows [][]int
	path []byte
	out  []FuzzyMatch
}

func (f *fuzzySearch) visit(state uint32, depth int, row []int) {
	if f.s.IsFinal(state) {
		if dist := row[len(f.q)]; dist <= f.k {
			f.out = append(f.out, FuzzyMatch{Text: string(f.path), Distance: dist})
		}
	}

	labels, targets := f.s.Edges(state)

	for i := range labels {
		start := len(f.path)
		f.path = append(f.path, labels[i])
		f.explore(targets[i], depth, row, start)
		f.path = f.path[:start]
	}
}

// explore completes one rune from the byte prefix f.path[start:] reached at
// state and applies the DP transition for it. Multi-byte runes are consumed
// by descending up to three continuation edges before the transition fires.
func (f *fuzzySearch) explore(state uint32, depth int, row []int, start int) {
	tail := f.path[start:]

	if !utf8.FullRune(tail) {
		labels, targets := f.s.Edges(state)

		for i := range labels {
			f.path = append(f.path, labels[i])
			f.explore(targets[i], depth, row, start)
			f.path = f.path[:len(f.path)-1]
		}

		return
	}

	r, _ := utf8.DecodeRune(tail)
	if r == utf8.RuneError {
		return // invalid UTF-8 branch: stored keys are always valid
	}

	nrow := f.nextRow(depth, row, r)
	if minRow(nrow) <= f.k {
		f.visit(state, depth+1, nrow)
	}
}

// nextRow advances the DP one rune deeper, writing into the pooled buffer
// of depth+1. The caller owns the result until its subtree is done.
func (f *fuzzySearch) nextRow(depth int, row []int, r rune) []int {
	nrow := f.rowFor(depth + 1)
	nrow[0] = row[0] + 1

	for j := 1; j <= len(f.q); j++ {
		cost := 1
		if f.q[j-1] == r {
			cost = 0
		}

		nrow[j] = min(row[j]+1, nrow[j-1]+1, row[j-1]+cost)
	}

	return nrow
}

func (f *fuzzySearch) rowFor(depth int) []int {
	for len(f.rows) <= depth {
		f.rows = append(f.rows, make([]int, len(f.q)+1))
	}

	return f.rows[depth]
}

// minRow reports the row minimum; once it exceeds k no completion of the
// subtree can come back under the threshold (rows grow monotonically).
func minRow(row []int) int {
	m := row[0]

	for _, v := range row[1:] {
		if v < m {
			m = v
		}
	}

	return m
}
