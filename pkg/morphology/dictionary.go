// Package morphology is the public API for morphological analysis over
// dictionaries in a single internal format.
//
// # Quick start
//
// Install the gomorphy command (go install github.com/amarin/gomorphy/cmd/gomorphy),
// fetch and compile a ready-made dictionary in one step, then look words up:
//
//	gomorphy update unimorph
//	gomorphy lookup -d .data/unimorph/ru/unimorph.dat кота
//
// Self-contained runnable programs for every entry point of this package
// live in the examples/ directory of the repository.
//
// # Creating a dictionary in code
//
// Dictionaries come from three bundled importers — pymorphy2, OpenCorpora
// and UniMorph (the CompileFrom* helpers wrap one in a single call, see
// also docs/en/unimorph.md) — or are built from scratch with [Builder]
// (accumulates wordforms programmatically, see ExampleNewBuilder) and
// [ImportTSV] (loads a lemma<TAB>wordform[<TAB>tags] stream, see
// ExampleImportTSV). The CLI mirrors these as `gomorphy build` for the
// importers and `gomorphy import tsv words.tsv -o out.dat` for TSV.
//
// # Merging dictionaries
//
// [Merge] adds overlay dictionaries into a base one without rebuilding it:
// the base keeps its tag set, probabilities and prediction. [MergeAdd]
// keeps the base reading for words present in both, [MergeReplace] swaps
// in the overlay's (the last overlay wins); see ExampleMerge and
// [MergeWithOptions]. The CLI equivalent is
// `gomorphy merge --mode add -o merged.dat base.dat overlay.dat`.
//
// # Dictionary words and guesses
//
// [Dictionary.Parse] and [Dictionary.Lemma] lower-case their input and fall
// back to ending-based prediction for words the dictionary does not have;
// such results carry [Reading].Predicted / [LemmaRef].Predicted.
// [Dictionary.IsKnown] checks membership without predicting — the building
// block for dictionary-based named-entity lookup.
//
// # Character policy
//
// A dictionary stores a [CharPolicy]: one-way rune substitutions applied to
// every lookup, including [Dictionary.Fuzzy]. The default is е→ё («елка»
// finds «ёлка») for Russian and none for other languages; set
// [BuilderOptions].CharPolicy to [NoCharPolicy], [RussianCharPolicy] or
// [NewCharPolicy] to change it.
//
// # Opening and lifecycle
//
// [Open] maps a GMOR file with mmap (not supported on Windows); [OpenBytes]
// opens one already in memory, typically embedded with //go:embed, and
// works everywhere. Query methods are safe for concurrent use;
// [Dictionary.Close] must not race with them. [MultiDictionary] queries
// several dictionaries as one, and [Dictionary.ContentHash] identifies a
// dictionary's content independently of when and how it was saved. Package
// tagmap compares tags across sources.
//
// Usage scenarios with runnable examples: docs/en/scenarios.md in the
// repository.
package morphology

import (
	"sync"

	"github.com/amarin/gomorphy/internal/mmapx"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Dictionary is an immutable dictionary: loaded by an importer, built with
// Builder/ImportTSV/Merge, opened from a GMOR file (Open, mmap-backed) or
// from memory (OpenBytes). Dictionaries opened via Open must be closed with
// Close when no longer needed; see Close for when that is safe.
// All query methods (Parse, Lemma, IsKnown, Fuzzy, FuzzyTop, Info,
// ContentHash, …) are safe for concurrent use by multiple goroutines;
// Close is not.
type Dictionary struct {
	d  *internal.Dictionary
	mm *mmapx.Region

	hashOnce sync.Once // guards hash; see ContentHash
	hash     string
}
