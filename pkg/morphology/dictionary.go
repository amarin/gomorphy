// Package morphology is the public API for morphological analysis over
// dictionaries in a single internal format. Dictionaries come from three
// bundled importers — pymorphy2, OpenCorpora and UniMorph (see
// docs/en/unimorph.md) — or are built from scratch: Builder accumulates
// wordform entries, ImportTSV loads a wordform TSV, Merge combines
// existing dictionaries, and the CompileFrom* helpers wrap an importer
// in a single call.
package morphology

import (
	"github.com/amarin/gomorphy/internal/mmapx"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Dictionary is an immutable dictionary, either loaded by an importer or
// opened from a GMOR file (Open). Dictionaries opened via Open use mmap
// and must be closed with Close when no longer needed.
type Dictionary struct {
	d  *internal.Dictionary
	mm *mmapx.Region
}
