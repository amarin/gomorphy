package morphology

import (
	"sort"
	"sync"
	"unicode/utf8"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// FuzzyMatch — a word found by fuzzy search, with its distance to the query.
type FuzzyMatch struct {
	Word     string
	Distance int
	Dict     int // dictionary index in MultiDictionary; always 0 for Dictionary.Fuzzy/FuzzyTop directly
}

// Fuzzy returns dictionary words within Levenshtein distance maxDist of
// word (a rune-wise metric: inserting/deleting/replacing one rune costs 1;
// "ё/е" counts as one substitution). The result is sorted by
// (distance, word), with duplicate words (multiple readings, including
// from different shards) collapsed. A negative maxDist is treated as 0
// (exact lookup). An empty result means no words match.
//
// Dictionaries with a dense alphabet (Dictionary.Alphabet != nil, e.g.
// opened via OpenPyMorphyDense) are not yet supported: the internal walk
// decodes DAWG bytes as raw UTF-8, which for dense-coded bytes produces
// silent garbage rather than an error. For such a dictionary, Fuzzy
// returns nil.
func (x *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch {
	if x.d.Alphabet != nil {
		return nil
	}
	if maxDist < 0 {
		maxDist = 0
	}
	return dedupeFuzzy(x.fuzzyWalk(word, maxDist))
}

// FuzzyTop returns up to maxWords nearest words, ordered by
// (distance, word). The distance is widened iteratively from 0 up to an
// upper bound (len(query) + the longest word length in runes across all
// shards), until either maxWords words are collected or the whole
// dictionary has been walked. maxWords ≤ 0 means exact lookup (the word
// itself, or nothing).
//
// Like Fuzzy, this does not support dictionaries with a dense alphabet
// (Dictionary.Alphabet != nil) — it returns nil.
func (x *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch {
	if x.d.Alphabet != nil {
		return nil
	}
	if maxWords <= 0 {
		return x.Fuzzy(word, 0)
	}

	maxRunes := x.maxWordRunes()
	if maxRunes == 0 {
		return nil
	}

	bound := utf8.RuneCountInString(word) + maxRunes
	seen := make(map[string]bool, maxWords)
	var out []FuzzyMatch
	for dist := 0; dist <= bound && len(out) < maxWords; dist++ {
		for _, m := range x.fuzzyWalk(word, dist) {
			if seen[m.Word] {
				continue
			}
			seen[m.Word] = true
			out = append(out, m)
			if len(out) >= maxWords {
				break
			}
		}
	}
	return out
}

func dedupeFuzzy(matches []FuzzyMatch) []FuzzyMatch {
	if len(matches) == 0 {
		return nil
	}
	out := make([]FuzzyMatch, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		if seen[m.Word] {
			continue
		}
		seen[m.Word] = true
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// fuzzyWalk walks each shard in parallel (one goroutine per shard) and
// concatenates the results. Shards are independent DAWGs, so the walks
// share no mutable state.
func (x *Dictionary) fuzzyWalk(word string, k int) []FuzzyMatch {
	results := make([][]FuzzyMatch, len(x.d.Words))

	var wg sync.WaitGroup
	for shard, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		wg.Add(1)
		go func(shard int, dawg *internal.DAWG) {
			defer wg.Done()
			results[shard] = fuzzyWalkShard(dawg, word, k)
		}(shard, dawg)
	}
	wg.Wait()

	var out []FuzzyMatch
	for _, r := range results {
		out = append(out, r...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// fuzzyWalkShard — a single pass of the joint traversal of one DAWG shard
// and banded Levenshtein DP.
func fuzzyWalkShard(words *internal.DAWG, word string, k int) []FuzzyMatch {
	f := &fuzzySearch{
		words: words,
		q:     []rune(word),
		k:     k,
		path:  make([]byte, 0, 32),
	}

	row := f.rowFor(0)
	for j := range row {
		row[j] = j
	}
	f.visit(0, 0, row)
	return f.out
}

// fuzzySearch carries the state of a single traversal: rows[depth] is the
// DP row after depth runes of the path, path is the current DAWG path's
// bytes.
type fuzzySearch struct {
	words *internal.DAWG
	q     []rune
	k     int
	rows  [][]int
	path  []byte
	out   []FuzzyMatch
}

func (f *fuzzySearch) visit(state uint32, depth int, row []int) {
	if f.words.HasPayloadChild(state) {
		if dist := row[len(f.q)]; dist <= f.k {
			f.out = append(f.out, FuzzyMatch{Word: string(f.path), Distance: dist})
		}
	}

	f.words.ForEachChild(state, func(label byte, next uint32) {
		if label == internal.PayloadSeparator {
			return
		}
		start := len(f.path)
		f.path = append(f.path, label)
		f.explore(next, depth, row, start)
		f.path = f.path[:start]
	})
}

// explore completes the current rune (the byte path from position start
// onward) and applies the DP transition; recursion continues if the row is
// still within k.
func (f *fuzzySearch) explore(state uint32, depth int, row []int, start int) {
	tail := f.path[start:]

	if !utf8.FullRune(tail) {
		f.words.ForEachChild(state, func(label byte, next uint32) {
			if label == internal.PayloadSeparator {
				return
			}
			f.path = append(f.path, label)
			f.explore(next, depth, row, start)
			f.path = f.path[:len(f.path)-1]
		})
		return
	}

	r, _ := utf8.DecodeRune(tail)
	if r == utf8.RuneError {
		return
	}

	nrow := f.nextRow(depth, row, r)
	if minRow(nrow) <= f.k {
		f.visit(state, depth+1, nrow)
	}
}

func (f *fuzzySearch) nextRow(depth int, row []int, r rune) []int {
	nrow := f.rowFor(depth + 1)
	nrow[0] = row[0] + 1

	for j := 1; j <= len(f.q); j++ {
		cost := 1
		if f.q[j-1] == r {
			cost = 0
		}
		nrow[j] = min3(row[j]+1, nrow[j-1]+1, row[j-1]+cost)
	}
	return nrow
}

func (f *fuzzySearch) rowFor(depth int) []int {
	for len(f.rows) <= depth {
		f.rows = append(f.rows, make([]int, len(f.q)+1))
	}
	return f.rows[depth]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func minRow(row []int) int {
	m := row[0]
	for _, v := range row[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// maxWordRunes returns the longest dictionary wordform's length in runes,
// across all shards.
func (x *Dictionary) maxWordRunes() int {
	best := 0
	for _, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		if m := maxWordRunesInShard(dawg); m > best {
			best = m
		}
	}
	return best
}

// maxWordRunesInShard walks one shard's DAG. The walk deduplicates nodes
// by the greatest depth reached — otherwise shared suffixes would be
// counted exponentially. UTF-8 continuation bytes (0x80-0xBF) do not add
// a rune.
func maxWordRunesInShard(words *internal.DAWG) int {
	seen := make(map[uint32]int)
	var rec func(state uint32, runes int)
	rec = func(state uint32, runes int) {
		if prev, ok := seen[state]; ok && prev >= runes {
			return
		}
		seen[state] = runes
		words.ForEachChild(state, func(label byte, next uint32) {
			if label == internal.PayloadSeparator {
				return
			}
			step := 1
			if label >= 0x80 && label <= 0xBF {
				step = 0
			}
			rec(next, runes+step)
		})
	}
	rec(0, 0)

	max := 0
	for _, depth := range seen {
		if depth > max {
			max = depth
		}
	}
	return max
}
