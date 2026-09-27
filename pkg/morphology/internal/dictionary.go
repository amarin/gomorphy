// Package internal holds gomorphy's internal dictionary representation and
// on-disk format: TagSet, Paradigm, the DAWG engine and builder, and the
// sectioned GMOR container format. Not part of the public API — external
// code should use pkg/morphology instead.
package internal

// Dictionary is an immutable dictionary snapshot.
//
// Suffixes, Paradigms, and Words each hold one element per shard; index 0
// is the only shard for unsharded dictionaries (there's no separate
// "unsharded" representation). Shards exist because a suffix id is
// addressed as a uint16: a dictionary with more suffixes than fit in one
// uint16 range is split into several shards with independent id spaces
// (see docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md).
// TagSet and Prefixes stay shared across all shards.
type Dictionary struct {
	Language   string
	TagSet     *TagSet
	Suffixes   [][]string
	Prefixes   []string
	Paradigms  [][]Paradigm
	Words      []*DAWG
	Prediction []*DAWG
	// PredictionSharded is the value format of Prediction: true for 8-byte
	// count|para|form|shard values (dictionaries with more than one shard,
	// saved as prediction-sharded-N), false for 6-byte count|para|form
	// values resolved against shard 0 (saved as prediction-N).
	PredictionSharded bool
	Probability       *DAWG
	CharPolicy        *CharPolicy
	Alphabet          Alphabet // nil = raw UTF-8 keys (today's behavior, unchanged)
	Info              *BuildInfo
}

// NewDictionary assembles a Dictionary from its components. suffixes,
// paradigms, and words must have equal length (the number of shards) — it
// panics otherwise, the same way NewParadigm panics on mismatched part
// lengths. Prediction, Probability, and Info stay nil; Prediction/
// Probability are filled in by importers when reading prediction files,
// Info by an importer (usually just Source) or by SaveTo
// (BuiltAt/LibraryVersion — on every save).
func NewDictionary(
	language string,
	tagSet *TagSet,
	suffixes [][]string,
	prefixes []string,
	paradigms [][]Paradigm,
	words []*DAWG,
	charPolicy *CharPolicy,
) *Dictionary {
	if len(suffixes) != len(paradigms) || len(paradigms) != len(words) {
		panic("internal: NewDictionary shard slices (suffixes, paradigms, words) must have equal length")
	}
	return &Dictionary{
		Language:   language,
		TagSet:     tagSet,
		Suffixes:   suffixes,
		Prefixes:   prefixes,
		Paradigms:  paradigms,
		Words:      words,
		CharPolicy: charPolicy,
	}
}
