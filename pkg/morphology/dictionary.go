// Package morphology is the public API for morphological analysis over
// dictionaries in a single internal format. Two sources are supported
// today: pymorphy2 and OpenCorpora; UniMorph import is planned but not
// yet implemented (see docs/todo.md, Stage 16).
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
