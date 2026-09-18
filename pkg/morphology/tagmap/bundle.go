// Package tagmap normalizes a dictionary's native grammatical tag string
// into a universal feature bundle (the UniMorph Schema), so tags from
// different dictionary sources (OpenCorpora, pymorphy2) can be compared
// for the same grammatical meaning. See
// docs/en/superpowers/specs/2026-09-17-tag-mapping-design.md.
package tagmap

import "sort"

// Dimension is one of the UniMorph Schema's dimensions of meaning
// (Sylak-Glassman 2016), restricted to the ones a mapping table in this
// package actually uses. Declaration order fixes the canonical sort
// order for Bundle.Features — this is what makes bundles built from
// different sources comparable regardless of the input tag's own token
// order. New dimensions are appended here only when a mapping table
// needs them; existing ones must never be reordered (it would silently
// change every existing Bundle's canonical order).
type Dimension uint8

const (
	DimPartOfSpeech Dimension = iota
	DimAnimacy
	DimCase
	DimNumber
	DimGender
	DimTense
	DimAspect
	DimMood
	DimVoice
	DimPerson
)

// Feature is a single UniMorph feature value within a Dimension, e.g.
// {DimCase, "NOM"}.
type Feature struct {
	Dim   Dimension
	Value string
}

// Bundle is a native tag normalized into UniMorph terms. Features is
// sorted by Dimension so two Bundles expressing the same grammatical
// meaning from different sources compare equal via reflect.DeepEqual/
// assert.Equal, independent of source token order. Unmapped holds,
// verbatim and in encounter order, every input token that had no entry
// in the source's mapping table — never merged into Features, never
// dropped silently. If two tokens both map to the same Dimension, the
// later token's Feature silently replaces the earlier one (later wins).
type Bundle struct {
	Features []Feature
	Unmapped []string
}

// buildBundle maps each token through table, merging hits by Dimension
// (a later token whose Feature shares a Dimension with an earlier one
// overwrites it) and collecting misses into Unmapped verbatim. The
// result's Features is sorted by Dimension's declaration order.
func buildBundle(tokens []string, table map[string]Feature) Bundle {
	var (
		byDim    map[Dimension]Feature
		order    []Dimension
		unmapped []string
	)

	for _, tok := range tokens {
		f, ok := table[tok]
		if !ok {
			unmapped = append(unmapped, tok)
			continue
		}
		if byDim == nil {
			byDim = make(map[Dimension]Feature)
		}
		if _, exists := byDim[f.Dim]; !exists {
			order = append(order, f.Dim)
		}
		byDim[f.Dim] = f
	}

	if len(order) == 0 {
		return Bundle{Unmapped: unmapped}
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	features := make([]Feature, 0, len(order))
	for _, d := range order {
		features = append(features, byDim[d])
	}
	return Bundle{Features: features, Unmapped: unmapped}
}
