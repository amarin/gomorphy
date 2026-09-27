package morphology

import (
	"cmp"
	"encoding/binary"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Reading — one parse of a wordform.
type Reading struct {
	Word   string  // wordform as stored in the dictionary (with "ё")
	Normal string  // lemma (base form)
	Tag    string  // grammeme tag, e.g. "NOUN,anim,masc,sing,nomn"
	Para   uint16  // paradigm id — unique only together with Shard
	Form   uint16  // form index within the paradigm
	Shard  int     // dictionary shard index; always 0 for unsharded dictionaries
	Dict   int     // dictionary index in MultiDictionary; always 0 for Dictionary.Parse directly
	Prob   float64 // probability of this reading (0 if probability data is unavailable)
	// Predicted is true when the reading was produced by suffix prediction
	// (the word is absent from the dictionary), false for dictionary
	// readings. Parse returns either only dictionary readings or only
	// predicted ones for a given word; see also Dictionary.IsKnown.
	Predicted bool
}

// Parse parses word and returns all dictionary readings, sorted by
// probability (descending) when the dictionary carries probability data
// (pymorphy2), otherwise in storage order. For out-of-dictionary words, it
// tries to predict readings from the prediction-DAWG (suffixes); such
// readings have Predicted set. Returns nil if no readings are found, or
// for a nil receiver. The input is lower-cased and the dictionary's
// CharPolicy applied. Use IsKnown to check membership without prediction.
func (x *Dictionary) Parse(word string) []Reading {
	out := x.ParseAppend(nil, word)
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParseAppend is Parse that appends the readings to dst and returns the
// extended slice (dst unchanged when there are none). Reusing dst across
// calls avoids allocating the result slice; the appended readings are in
// the same order as Parse returns them.
func (x *Dictionary) ParseAppend(dst []Reading, word string) []Reading {
	if x == nil || x.d == nil || len(x.d.Words) == 0 {
		return dst
	}
	word = strings.ToLower(word)
	n := len(dst)
	if dst = x.exactAppend(dst, word); len(dst) > n {
		return dst
	}
	return x.predictAppend(dst, word)
}

// shardExactResult — the result of exactInShard for a single shard.
type shardExactResult struct {
	readings []Reading
	hasProb  bool
}

// exactAppend appends the readings of a word found in the dictionary,
// accounting for CharPolicy substitutions (е→ё), sorted by probability.
// A single-shard dictionary is searched inline; several shards are
// queried in parallel (one goroutine per shard) and concatenated in shard
// order.
func (x *Dictionary) exactAppend(dst []Reading, word string) []Reading {
	start := len(dst)
	hasProb := false
	if len(x.d.Words) == 1 {
		dst, hasProb = x.exactInShard(dst, 0, x.d.Words[0], word)
	} else {
		results := make([]shardExactResult, len(x.d.Words))
		var wg sync.WaitGroup
		for shard, dawg := range x.d.Words {
			wg.Add(1)
			go func(shard int, dawg *internal.DAWG) {
				defer wg.Done()
				rs, hp := x.exactInShard(nil, shard, dawg, word)
				results[shard] = shardExactResult{readings: rs, hasProb: hp}
			}(shard, dawg)
		}
		wg.Wait()
		for _, r := range results {
			dst = append(dst, r.readings...)
			hasProb = hasProb || r.hasProb
		}
	}
	if hasProb {
		slices.SortStableFunc(dst[start:], func(a, b Reading) int { return cmp.Compare(b.Prob, a.Prob) })
	}
	return dst
}

// exactInShard appends a word's readings from a single shard. Read-only:
// safe to run in parallel with other shards.
func (x *Dictionary) exactInShard(dst []Reading, shard int, dawg *internal.DAWG, word string) ([]Reading, bool) {
	hasProb := false
	dawg.LookupEach(word, x.d.CharPolicy, x.d.Alphabet, func(found string, v []byte) {
		r, ok := x.reading(shard, found, v)
		if !ok {
			return
		}
		if x.d.Probability != nil {
			r.Prob = float64(x.d.Probability.FindJoined(found, ':', r.Tag)) / 1e6
			if r.Prob > 0 {
				hasProb = true
			}
		}
		dst = append(dst, r)
	})
	return dst, hasProb
}

// predictMaxSuffix is the longest word ending (in runes) looked up in the
// prediction-DAWG (internal.predictionMaxSuffix).
const predictMaxSuffix = 5

// readingKey dedups predicted readings by (Word, Normal, Tag).
type readingKey struct{ word, normal, tag string }

// predictAppend appends readings for an out-of-dictionary word predicted
// from its ending (pymorphy2's KnownSuffixAnalyzer, as in opennota/morph).
func (x *Dictionary) predictAppend(dst []Reading, word string) []Reading {
	if len(x.d.Prediction) == 0 {
		return dst
	}
	var splitBuf [predictMaxSuffix]int
	splits := suffixSplits(splitBuf[:0], word, predictMaxSuffix)
	if len(splits) == 0 {
		return dst
	}
	seen := make(map[readingKey]bool)
	for id, pref := range x.d.Prefixes {
		if id >= len(x.d.Prediction) || x.d.Prediction[id] == nil {
			continue
		}
		if !strings.HasPrefix(word, pref) {
			continue
		}
		dst = x.predictForPrefix(dst, id, word, splits, seen)
	}
	return dst
}

// predictForPrefix predicts readings against a single prefix's
// prediction-DAWG (x.d.Prediction[id]), widening from the longest suffix
// split toward shorter ones until at least 2 total matches accumulate —
// pymorphy2's KnownSuffixAnalyzer heuristic. splits are byte offsets into
// word (see suffixSplits). seen dedups (word, lemma, tag) across all
// prefixes tried by the caller and is mutated in place.
//
// A 6-byte value (count|para|form) resolves against shard 0; an 8-byte
// value (count|para|form|shard, sharded dictionaries) carries its shard.
// Values naming a missing shard or paradigm are skipped, and so are values
// whose form prefix/suffix the candidate word does not start/end with
// (they neither count toward the 2-match threshold nor yield a reading).
func (x *Dictionary) predictForPrefix(dst []Reading, id int, word string, splits []int, seen map[readingKey]bool) []Reading {
	totalCount := 0

	for i := len(splits) - 1; i >= 0; i-- {
		wordStart, wordEnd := word[:splits[i]], word[splits[i]:]
		// Prediction DAWGs are never recompiled under Dictionary.Alphabet
		// (see docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md's
		// non-goals): the alphabet is always nil here.
		x.d.Prediction[id].LookupEach(wordEnd, x.d.CharPolicy, nil, func(found string, v []byte) {
			if len(v) < 6 {
				return
			}
			count := int(binary.BigEndian.Uint16(v[:2]))
			paraNum := binary.BigEndian.Uint16(v[2:4])
			form := binary.BigEndian.Uint16(v[4:6])
			shard := 0 // 6-byte values (prediction-N) always mean shard 0
			if len(v) >= 8 {
				shard = int(binary.BigEndian.Uint16(v[6:8]))
			}

			para, ok := x.paradigm(shard, paraNum) // false for a missing shard
			if !ok || form >= uint16(para.Len()) {
				return
			}
			if !productive(x.paradigmTag(para, int(form))) {
				return
			}
			// A value whose form affixes the candidate lacks would build a
			// lemma from a suffix the word does not have (keys shorter than
			// the form suffix in Builder/merged files before ruling R14).
			candidate := wordStart + found
			prefix, suffix := x.paradigmAffix(shard, para, int(form))
			if !strings.HasPrefix(candidate, prefix) || !strings.HasSuffix(candidate, suffix) {
				return
			}
			totalCount += count

			r := x.readingForm(shard, candidate, paraNum, form)
			r.Predicted = true
			k := readingKey{r.Word, r.Normal, r.Tag}
			if seen[k] {
				return
			}
			seen[k] = true
			dst = append(dst, r)
		})
		if totalCount > 1 {
			break
		}
	}
	return dst
}

// reading decodes a words.dawg payload entry (4 bytes BE: para, form) in
// the given shard.
func (x *Dictionary) reading(shard int, word string, value []byte) (Reading, bool) {
	if len(value) < 4 {
		return Reading{}, false
	}
	para := binary.BigEndian.Uint16(value[:2])
	form := binary.BigEndian.Uint16(value[2:4])
	return x.readingForm(shard, word, para, form), true
}

// readingForm builds a Reading from a paradigm and form in the given shard
// (normal form = prefix₀ + stem + suffix₀ for form≠0, otherwise the word
// itself).
func (x *Dictionary) readingForm(shard int, word string, paraNum, form uint16) Reading {
	para, ok := x.paradigm(shard, paraNum)
	if !ok || int(form) >= para.Len() {
		return Reading{Word: word, Para: paraNum, Form: form, Shard: shard}
	}

	prefix, suffix := x.paradigmAffix(shard, para, int(form))
	norm := word
	if form != 0 {
		stem := strings.TrimPrefix(word, prefix)
		stem = strings.TrimSuffix(stem, suffix)
		p0, s0 := x.paradigmAffix(shard, para, 0)
		norm = p0 + stem + s0
	}

	return Reading{
		Word:   word,
		Normal: norm,
		Tag:    x.paradigmTag(para, int(form)),
		Para:   paraNum,
		Form:   form,
		Shard:  shard,
	}
}

func (x *Dictionary) paradigm(shard int, id uint16) (internal.Paradigm, bool) {
	if shard < 0 || shard >= len(x.d.Paradigms) {
		return internal.Paradigm{}, false
	}
	if int(id) < len(x.d.Paradigms[shard]) {
		return x.d.Paradigms[shard][id], true
	}
	return internal.Paradigm{}, false
}

// paradigmAffix returns the prefix and suffix of a paradigm form for the
// given shard (empty when out of bounds). Prefixes is shared across all
// shards; Suffixes is per-shard.
func (x *Dictionary) paradigmAffix(shard int, para internal.Paradigm, form int) (prefix, suffix string) {
	if form >= para.Len() {
		return "", ""
	}
	var suffixes []string
	if shard >= 0 && shard < len(x.d.Suffixes) {
		suffixes = x.d.Suffixes[shard]
	}
	return strAt(x.d.Prefixes, para.Prefix(form)), strAt(suffixes, para.Suffix(form))
}

// paradigmTag returns the tag name of a paradigm form. TagSet is shared
// across all shards, so no shard is needed.
func (x *Dictionary) paradigmTag(para internal.Paradigm, form int) string {
	if form >= para.Len() {
		return ""
	}
	if x.d.TagSet == nil {
		return ""
	}
	return x.d.TagSet.TagName(para.Tag(form))
}

func strAt(ar []string, i uint16) string {
	if int(i) < len(ar) {
		return ar[i]
	}
	return ""
}

// productive reports whether the grammeme is not in nonproductiveGrammemes.
func productive(tag string) bool {
	if tag == "" {
		return false
	}
	for part := range strings.SplitSeq(tag, ",") {
		if slices.Contains(nonproductiveGrammemes, part) {
			return false
		}
	}
	return true
}

// nonproductiveGrammemes — grammemes for which prediction does not yield
// productive readings (pymorphy2).
var nonproductiveGrammemes = []string{"NUMR", "NPRO", "PRED", "PREP", "CONJ", "PRCL", "INTJ", "Apro"}

// suffixSplits appends to dst the byte offsets that split word into
// (start, end) with end = the last 1, 2, …, max runes (shortest end
// first). No allocation beyond dst's growth.
func suffixSplits(dst []int, word string, max int) []int {
	i := len(word)
	for n := 0; n < max && i > 0; n++ {
		_, size := utf8.DecodeLastRuneInString(word[:i])
		i -= size
		dst = append(dst, i)
	}
	return dst
}
