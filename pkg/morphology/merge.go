package morphology

import (
	"fmt"
	"slices"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// mergeTagSetName is the TagSet name for dictionaries produced by Merge.
const mergeTagSetName = "merge"

// MergeMode controls how Merge combines a base dictionary with overlays.
type MergeMode int

const (
	// MergeAdd appends overlay readings only to words that are absent
	// from the base dictionary: a word present in the base keeps all of
	// its base readings, its union with an overlay contributes nothing.
	// Words unique to an overlay are added with all of their readings.
	MergeAdd MergeMode = iota

	// MergeReplace replaces the base readings of any word the overlays
	// provide: a word present in both keeps the overlay's readings
	// instead of the base's, while words present only in the base (or
	// only in an overlay) carry over unchanged. When several overlays
	// cover the same word, the first overlay's readings win and later
	// overlays append their readings for that word.
	MergeReplace
)

// Merge combines the base dictionary with the given overlays into a new,
// fully functional *Dictionary (Parse, Lemma, Fuzzy, prediction), dense
// and Savable like any Builder-built dictionary. The inputs are read-only
// and never mutated.
//
// Merge is deterministic: given the same inputs and mode, it produces the
// same output, byte for byte, run to run. Words are emitted in ascending
// lexicographic order; a word's readings keep the order in which they were
// accumulated — the base dictionary's shards first, then the overlays in
// the order they are supplied.
//
// The output dictionary inherits the base dictionary's language and
// CharPolicy, and carries BuildInfo{Source: "merge"} with the base
// dictionary's SourceVersion and Description, when present.
//
// Merge returns an error (wrapping ErrNoEntries) if the combined entry set
// is empty — which is only reachable with a degenerate empty base
// dictionary and no overlays — and rejects nil inputs and unknown modes.
func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error) {
	if base == nil {
		return nil, fmt.Errorf("morphology: merge: base dictionary is nil")
	}
	for i, overlay := range overlays {
		if overlay == nil {
			return nil, fmt.Errorf("morphology: merge: overlay %d is nil", i)
		}
	}
	if mode != MergeAdd && mode != MergeReplace {
		return nil, fmt.Errorf("morphology: merge: unknown mode %d", int(mode))
	}

	baseEntries, err := collectEntries(base)
	if err != nil {
		return nil, fmt.Errorf("morphology: merge: base: %w", err)
	}
	baseSet := make(map[string]bool, len(baseEntries))
	for word := range baseEntries {
		baseSet[word] = true
	}

	merged := baseEntries
	replaced := make(map[string]bool)
	for _, overlay := range overlays {
		overlayEntries, err := collectEntries(overlay)
		if err != nil {
			return nil, fmt.Errorf("morphology: merge: overlay: %w", err)
		}
		for word, readings := range overlayEntries {
			switch mode {
			case MergeAdd:
				if baseSet[word] {
					continue
				}
			case MergeReplace:
				if baseSet[word] && !replaced[word] {
					merged[word] = readings
					replaced[word] = true
					continue
				}
			}
			merged[word] = append(merged[word], readings...)
		}
	}

	words := make([]string, 0, len(merged))
	for word := range merged {
		words = append(words, word)
	}
	slices.Sort(words)

	var entries []internal.BuildEntry
	for _, word := range words {
		entries = append(entries, merged[word]...)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("morphology: merge: %w", ErrNoEntries)
	}

	out, err := buildFromEntries(BuilderOptions{
		Language: base.d.Language,
		Source:   "merge",
	}, entries, mergeTagSetName)
	if err != nil {
		return nil, fmt.Errorf("morphology: merge: %w", err)
	}

	// buildFromEntries cannot express a CharPolicy override or a richer
	// BuildInfo, so carry both over here: the output behaves exactly like
	// the base dictionary except for its "merge" provenance.
	out.d.CharPolicy = base.d.CharPolicy
	info := &internal.BuildInfo{Source: "merge"}
	if base.d.Info != nil {
		info.SourceVersion = base.d.Info.SourceVersion
		info.Description = base.d.Info.Description
	}
	out.d.Info = info

	return out, nil
}

// collectEntries enumerates every wordform reading of one dictionary as
// raw (Word, Lemma, Tag) build entries, keyed by the wordform text. A
// word's readings may span several shards (each homonym reading landed in
// the shard that held its suffix/tag ids at build time); shards are walked
// in order, so readings accumulate in a deterministic order.
func collectEntries(d *Dictionary) (map[string][]internal.BuildEntry, error) {
	if d == nil || d.d == nil {
		return nil, fmt.Errorf("dictionary is nil")
	}
	entries := make(map[string][]internal.BuildEntry)
	for shard := range d.d.Words {
		if d.d.Words[shard] == nil {
			continue
		}
		var walkErr error
		d.d.Words[shard].Walk(func(key string, values [][]byte) {
			word := key
			if d.d.Alphabet != nil {
				decoded, err := d.d.Alphabet.Decode([]byte(key))
				if err != nil {
					if walkErr == nil {
						walkErr = fmt.Errorf("decode DAWG key: %w", err)
					}
					return
				}
				word = decoded
			}
			for _, value := range values {
				r, ok := d.reading(shard, word, value)
				if !ok {
					continue
				}
				entries[word] = append(entries[word], internal.BuildEntry{Word: r.Word, Lemma: r.Normal, Tag: r.Tag})
			}
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return entries, nil
}
