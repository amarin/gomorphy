package morphology

import (
	"fmt"
	"time"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// SaveTo writes the dictionary to a GMOR file — the single on-disk format.
// Sections: meta, info, tagset, prefixes, suffixes-N, paradigms-N,
// words.dawg-N (one set per shard, N starting at 0), prediction-N,
// probability (if present).
func (x *Dictionary) SaveTo(path string) error {
	if x == nil || x.d == nil {
		return fmt.Errorf("morphology: nil dictionary")
	}
	if x.d.Alphabet != nil {
		return fmt.Errorf("morphology: SaveTo: dictionaries with a non-nil Alphabet " +
			"(e.g. from OpenPyMorphyDense) cannot yet be serialized to .dat — the Alphabet " +
			"codec has no on-disk representation, so a saved-and-reopened dense dictionary " +
			"would silently mis-decode; see docs/en/superpowers/specs/" +
			"2026-09-16-pymorphy2-dense-recompile-design.md's non-goals")
	}

	// info is a copy of x.d.Info (if an importer populated it, e.g.
	// Source), with BuiltAt/LibraryVersion set anew on every save; x.d
	// itself is not mutated (Dictionary is immutable).
	info := internal.BuildInfo{}
	if x.d.Info != nil {
		info = *x.d.Info
	}
	info.BuiltAt = time.Now().UTC()
	info.LibraryVersion = Version

	// Compression is not yet implemented (see docs/en/todo.md, "Stage 17")
	// — all sections are written as-is. words.dawg-N will always stay
	// CompressionNone: it is aliased from mmap without copying, while a
	// compressed section would require full decompression into memory on
	// load.
	const noCompression = internal.CompressionNone
	sections := []internal.Section{
		{Name: "meta", Data: internal.EncodeMeta(x.d.Language, x.d.CharPolicy), Flags: noCompression},
		{Name: "info", Data: internal.EncodeBuildInfo(&info), Flags: noCompression},
		{Name: "tagset", Data: internal.EncodeTagSet(x.d.TagSet), Flags: noCompression},
		{Name: "prefixes", Data: internal.EncodeStrings(x.d.Prefixes), Flags: noCompression},
	}
	for i, suffixes := range x.d.Suffixes {
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("suffixes-%d", i), Data: internal.EncodeStrings(suffixes), Flags: noCompression,
		})
	}
	for i, paradigms := range x.d.Paradigms {
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("paradigms-%d", i), Data: internal.EncodeParadigms(paradigms), Flags: noCompression,
		})
	}
	for i, words := range x.d.Words {
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("words.dawg-%d", i), Data: words.Bytes(), Flags: internal.CompressionNone,
		})
	}
	for i, pred := range x.d.Prediction {
		if pred == nil {
			continue
		}
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("prediction-%d", i), Data: pred.Bytes(), Flags: noCompression,
		})
	}
	if x.d.Probability != nil {
		sections = append(sections, internal.Section{
			Name: "probability", Data: x.d.Probability.Bytes(), Flags: noCompression,
		})
	}
	return internal.SaveContainer(path, sections)
}
