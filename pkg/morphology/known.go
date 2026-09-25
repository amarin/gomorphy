package morphology

import "strings"

// IsKnown reports whether word has at least one dictionary reading. It runs
// the same lookup Parse runs first — the input is lower-cased, the
// dictionary's CharPolicy is applied (е→ё for Russian), every shard is
// searched — but never falls back to prediction: IsKnown(w) is true exactly
// when Parse(w) returns readings with Predicted == false. It does not build
// Reading values.
func (x *Dictionary) IsKnown(word string) bool {
	if x == nil || x.d == nil {
		return false
	}
	word = strings.ToLower(word)
	for _, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		for _, it := range dawg.SimilarItems(word, x.d.CharPolicy, x.d.Alphabet) {
			for _, v := range it.Values {
				if len(v) >= 4 { // the guard Dictionary.reading applies
					return true
				}
			}
		}
	}
	return false
}

// IsKnown reports whether any dictionary in the set knows word (see
// Dictionary.IsKnown). Dictionaries are checked in registration order and
// the search stops at the first hit.
func (m *MultiDictionary) IsKnown(word string) bool {
	for _, d := range m.dicts {
		if d.IsKnown(word) {
			return true
		}
	}
	return false
}
