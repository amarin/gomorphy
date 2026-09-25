package morphology

import (
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// FuzzyMatch — a word found by fuzzy search, with its distance to the query.
type FuzzyMatch struct {
	Word     string // the word as stored in the dictionary (with ё)
	Distance int    // Levenshtein distance to the query, in runes
	Dict     int // dictionary index in MultiDictionary; always 0 for Dictionary.Fuzzy/FuzzyTop directly
}

// Fuzzy returns dictionary words within Levenshtein distance maxDist of
// word (a rune-wise metric: inserting/deleting/replacing one rune costs 1;
// the dictionary's CharPolicy applies: a query rune that the policy
// substitutes (е for Russian) matches the substituted stored rune (ё) at
// cost 0, exactly as Parse finds «ёлка» for «елка»). The result is sorted by
// (distance, word), with duplicate words (multiple readings, including
// from different shards) collapsed. A negative maxDist is treated as 0
// (exact lookup). An empty result means no words match.
// The query is lower-cased, like Parse's input.
//
// Dictionaries with a dense alphabet (the …Dense constructors and every
// Builder, ImportTSV and Merge result) give exactly the same matches as the
// same dictionary without one. The receiver must not be nil.
func (x *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch {
	word = strings.ToLower(word)
	if x.d.Alphabet != nil && x.d.Alphabet.Width() == 0 {
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
// dictionary has been walked. maxWords ≤ 0 means distance-0 matches only,
// as Fuzzy(word, 0): the word itself and its CharPolicy variants (e.g.
// «ёлка» for «елка»).
// The query is lower-cased, like Parse's input. The receiver must not be
// nil.
func (x *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch {
	word = strings.ToLower(word)
	if x.d.Alphabet != nil && x.d.Alphabet.Width() == 0 {
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
			results[shard] = fuzzyWalkShard(dawg, x.d.Alphabet, x.d.CharPolicy, word, k)
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
// and banded Levenshtein DP. alphabet is the dictionary's Alphabet (nil
// for raw UTF-8 DAWGs); pol is its CharPolicy (nil = exact runes only).
func fuzzyWalkShard(words *internal.DAWG, alphabet internal.Alphabet, pol *internal.CharPolicy, word string, k int) []FuzzyMatch {
	f := &fuzzySearch{
		words:    words,
		alphabet: alphabet,
		pol:      pol,
		q:        []rune(word),
		k:        k,
		path:     make([]byte, 0, 32),
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
// bytes. alphabet decodes path's bytes into runes (nil = raw UTF-8); pol
// makes a substituted query rune match its target at cost 0.
type fuzzySearch struct {
	words    *internal.DAWG
	alphabet internal.Alphabet
	pol      *internal.CharPolicy
	q        []rune
	k        int
	rows     [][]int
	path     []byte
	out      []FuzzyMatch
}

func (f *fuzzySearch) visit(state uint32, depth int, row []int) {
	if f.words.HasPayloadChild(state) {
		if dist := row[len(f.q)]; dist <= f.k {
			if word, ok := f.decodeWord(); ok {
				f.out = append(f.out, FuzzyMatch{Word: word, Distance: dist})
			}
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

// decodeWord decodes the fully accumulated path into the matched word's
// text: a raw UTF-8 passthrough when alphabet is nil, otherwise
// alphabet.Decode. ok is false only if Decode errors, which isn't
// expected for a path built entirely from bytes decodeTail already
// validated rune by rune — treated as "skip this match", not a panic.
func (f *fuzzySearch) decodeWord() (string, bool) {
	if f.alphabet == nil {
		return string(f.path), true
	}
	word, err := f.alphabet.Decode(f.path)
	return word, err == nil
}

// explore completes the current rune (the byte path from position start
// onward) and applies the DP transition; recursion continues if the row is
// still within k.
func (f *fuzzySearch) explore(state uint32, depth int, row []int, start int) {
	tail := f.path[start:]

	r, ok, more := f.decodeTail(tail)
	if more {
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
	if !ok {
		return
	}

	nrow := f.nextRow(depth, row, r)
	if minRow(nrow) <= f.k {
		f.visit(state, depth+1, nrow)
	}
}

// decodeTail tries to decode tail — the bytes accumulated since the
// previous completed rune — into exactly one rune, using f.alphabet if
// non-nil or raw UTF-8 otherwise. more=true means tail isn't a complete
// encoded unit yet (recurse deeper along the DAWG to accumulate more
// bytes). ok=false with more=false means tail is complete-length but
// invalid (malformed UTF-8, or a dense code the alphabet doesn't
// recognize) — abandon this path.
func (f *fuzzySearch) decodeTail(tail []byte) (r rune, ok bool, more bool) {
	if f.alphabet == nil {
		if !utf8.FullRune(tail) {
			return 0, false, true
		}
		r, _ = utf8.DecodeRune(tail)
		if r == utf8.RuneError {
			return 0, false, false
		}
		return r, true, false
	}

	width := f.alphabet.Width()
	if len(tail) < width {
		return 0, false, true
	}
	decoded, err := f.alphabet.Decode(tail)
	if err != nil {
		return 0, false, false
	}
	rr := []rune(decoded)
	if len(rr) != 1 {
		return 0, false, false
	}
	return rr[0], true, false
}

func (f *fuzzySearch) nextRow(depth int, row []int, r rune) []int {
	nrow := f.rowFor(depth + 1)
	nrow[0] = row[0] + 1

	for j := 1; j <= len(f.q); j++ {
		cost := 1
		if f.q[j-1] == r || f.substitutes(f.q[j-1], r) {
			cost = 0
		}
		nrow[j] = min3(row[j]+1, nrow[j-1]+1, row[j-1]+cost)
	}
	return nrow
}

// substitutes reports whether the CharPolicy lets query rune q match the
// stored rune r (one direction only, like SimilarItems).
func (f *fuzzySearch) substitutes(q, r rune) bool {
	if f.pol == nil {
		return false
	}
	for _, s := range f.pol.Substitutions {
		if s.From == q && s.To == r {
			return true
		}
	}
	return false
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
	width := 0
	if x.d.Alphabet != nil {
		width = x.d.Alphabet.Width()
	}
	best := 0
	for _, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		if m := maxWordRunesInShard(dawg, width); m > best {
			best = m
		}
	}
	return best
}

// maxWordRunesInShard walks one shard's DAG, returning the length (in
// runes) of its longest key. The walk deduplicates nodes by the greatest
// byte depth reached — otherwise shared suffixes would be counted
// exponentially.
//
// For width == 0 (raw UTF-8 keys), it counts non-continuation bytes:
// 0x80-0xBF continuation bytes don't add a rune. For width > 0 (a
// fixed-width Alphabet), it counts every byte and divides the final
// max by width: every encoded rune is exactly width bytes, and — since a
// fixed-width encoding only ever produces whole-rune keys — every DAWG
// node in such a shard sits on a rune boundary on any path from the
// root, so this division is always exact, not an approximation.
func maxWordRunesInShard(words *internal.DAWG, width int) int {
	seen := make(map[uint32]int)
	var rec func(state uint32, bytes int)
	rec = func(state uint32, bytes int) {
		if prev, ok := seen[state]; ok && prev >= bytes {
			return
		}
		seen[state] = bytes
		words.ForEachChild(state, func(label byte, next uint32) {
			if label == internal.PayloadSeparator {
				return
			}
			step := 1
			if width == 0 && label >= 0x80 && label <= 0xBF {
				step = 0
			}
			rec(next, bytes+step)
		})
	}
	rec(0, 0)

	max := 0
	for _, depth := range seen {
		if depth > max {
			max = depth
		}
	}
	if width > 1 {
		max /= width
	}
	return max
}
