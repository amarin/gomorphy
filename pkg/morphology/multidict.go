// MultiDictionary aggregates Parse/Lemma/Close across an arbitrary set of
// already-open dictionaries. See
// docs/en/superpowers/specs/2026-09-16-multi-dict-design.md.
package morphology

import (
	"errors"
	"sort"
	"sync"
)

// MultiDictionary — a set of independently opened dictionaries, queried as
// a single whole. Each *Dictionary in the set retains its own lifecycle
// (mmap etc.) — MultiDictionary itself opens or imports nothing, it only
// aggregates Parse/Lemma and owns closing the whole set at once.
type MultiDictionary struct {
	dicts []*Dictionary
}

// NewMultiDictionary wraps already-open dictionaries into a single set.
// The order of dicts fixes the indexing of Reading.Dict/LemmaRef.Dict and
// the order in which Parse/Lemma results are concatenated — both always
// follow registration order and are never re-sorted.
func NewMultiDictionary(dicts ...*Dictionary) *MultiDictionary {
	return &MultiDictionary{dicts: dicts}
}

// Len returns the number of dictionaries in the set.
func (m *MultiDictionary) Len() int { return len(m.dicts) }

// DictInfo returns the diagnostic metadata of the dictionary at index i
// (the same index carried by Reading.Dict/LemmaRef.Dict), or nil if the
// index is out of range, or that dictionary has no info section (see
// Dictionary.Info).
func (m *MultiDictionary) DictInfo(i int) *BuildInfo {
	if i < 0 || i >= len(m.dicts) {
		return nil
	}
	return m.dicts[i].Info()
}

// Parse parses word across all dictionaries in the set in parallel (one
// goroutine per dictionary — the same pattern Dictionary.exact already
// uses for shards within a single dictionary). The result is the
// concatenation of each dictionary's Parse in the set's registration
// order, with Reading.Dict set; there is no sorting or deduplication
// across dictionaries beyond what each Dictionary.Parse already does
// internally. Returns nil if no dictionary produced any readings.
func (m *MultiDictionary) Parse(word string) []Reading {
	results := make([][]Reading, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			readings := d.Parse(word)
			for j := range readings {
				readings[j].Dict = i
			}
			results[i] = readings
		}(i, d)
	}
	wg.Wait()

	var out []Reading
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Lemma parses word across all dictionaries in the set and returns their
// lemmas. Same concatenate-in-registration-order semantics as Parse.
func (m *MultiDictionary) Lemma(word string) []LemmaRef {
	results := make([][]LemmaRef, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			refs := d.Lemma(word)
			for j := range refs {
				refs[j].Dict = i
			}
			results[i] = refs
		}(i, d)
	}
	wg.Wait()

	var out []LemmaRef
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Fuzzy searches for words within Levenshtein distance maxDist of word
// across all dictionaries in the set in parallel. The result is the
// concatenation of each dictionary's Fuzzy in the set's registration
// order, with FuzzyMatch.Dict set; no sorting/deduplication across
// dictionaries beyond what each Dictionary.Fuzzy already does internally
// (the same policy as Parse/Lemma) — Fuzzy does not cap the number of
// results, so concatenating as-is is correct.
func (m *MultiDictionary) Fuzzy(word string, maxDist int) []FuzzyMatch {
	results := make([][]FuzzyMatch, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			matches := d.Fuzzy(word, maxDist)
			for j := range matches {
				matches[j].Dict = i
			}
			results[i] = matches
		}(i, d)
	}
	wg.Wait()

	var out []FuzzyMatch
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// FuzzyTop returns up to maxWords nearest words across the whole set,
// ordered by (distance, word). Unlike Parse/Lemma/Fuzzy, maxWords is a cap
// on the overall result, not per dictionary, so a real merge is needed
// here: each dictionary is asked for its own top-maxWords (which is
// enough — any candidate of the global top-maxWords must also be in its
// own dictionary's top-maxWords, otherwise that same dictionary would
// have maxWords candidates at least as good), after which all candidates
// are re-sorted and truncated to maxWords. maxWords <= 0 means exact
// lookup, same as Dictionary.FuzzyTop.
func (m *MultiDictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch {
	if maxWords <= 0 {
		return m.Fuzzy(word, 0)
	}

	results := make([][]FuzzyMatch, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			matches := d.FuzzyTop(word, maxWords)
			for j := range matches {
				matches[j].Dict = i
			}
			results[i] = matches
		}(i, d)
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
	if len(out) > maxWords {
		out = out[:maxWords]
	}
	return out
}

// Close closes every dictionary in the set (Dictionary.Close is a no-op
// for dictionaries not opened via Open), aggregating all errors via
// errors.Join. The set must not be used after Close, nor its dictionaries.
func (m *MultiDictionary) Close() error {
	var errs []error
	for _, d := range m.dicts {
		if err := d.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
