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
type Dictionary struct {
	d  *internal.Dictionary
	mm *mmapx.Region

	hashOnce sync.Once // guards hash; see ContentHash
	hash     string
}
