package morphology

import (
	"errors"
	"fmt"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
)

// MergeMode is the word-level conflict policy Merge applies to every
// overlay. Overlays are applied in order, as a fold:
// Merge(b, [o1, o2], m) equals Merge(Merge(b, [o1], m), [o2], m). A word
// is identified by its exact stored form (no е/ё substitution).
type MergeMode int

const (
	// MergeAdd takes an overlay word only if it is absent from the base
	// and from every earlier overlay; words already present keep their
	// readings untouched.
	MergeAdd MergeMode = iota

	// MergeReplace lets an overlay word's readings fully replace whatever
	// the word had so far — the base's or an earlier overlay's. The last
	// overlay that has the word wins.
	MergeReplace
)

// ErrIncompatibleDictionaries is returned by Merge when an overlay's
// language differs from the base's, or when base and overlay use two
// different tag vocabularies that pkg/morphology/tagmap both knows
// (e.g. "opencorpora-int" and "unimorph") and so cannot share one TagSet.
var ErrIncompatibleDictionaries = errors.New("incompatible dictionaries")

// ErrPredictionSharded is returned by MergeWithOptions when
// RebuildPrediction is set but the merged dictionary has more than one
// shard.
var ErrPredictionSharded = internal.ErrPredictionSharded

// MergeOptions configures MergeWithOptions.
type MergeOptions struct {
	// Mode is the conflict policy (default MergeAdd).
	Mode MergeMode

	// RebuildPrediction replaces the base's prediction with one rebuilt
	// from every word of the merged dictionary, overlays included. By
	// default the base's prediction is carried over unchanged (overlay
	// words don't feed it) — the right choice for a large base such as
	// pymorphy2. Requires a single-shard result (ErrPredictionSharded).
	RebuildPrediction bool
}

// Merge is MergeWithOptions(base, overlays, MergeOptions{Mode: mode}).
func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error) {
	return MergeWithOptions(base, overlays, MergeOptions{Mode: mode})
}

// MergeWithOptions merges overlays into base and returns a new, dense,
// Savable dictionary. The merge is structural: the base's tag set,
// paradigms and suffixes keep their ids, overlay paradigms are remapped
// into them, and only the parts that changed are rebuilt. So the result
// keeps everything the base had:
//
//   - reading order and probabilities (p(tag|word)) of untouched words,
//     plus the probabilities an overlay carries for the words it adds;
//   - prediction for out-of-dictionary words (see RebuildPrediction);
//   - the base's TagSet name, so pkg/morphology/tagmap keeps working.
//     Overlay tags are appended verbatim.
//
// A replaced word loses all of its base readings and base probabilities.
// Inputs are never mutated. DAWGs and arrays are deep-copied so the
// result stays valid after the inputs are closed; immutable values
// (alphabet, CharPolicy) may be shared. The result reports
// BuildInfo{Source: "merge"} with the base's SourceVersion and
// Description, and the base's language and CharPolicy.
//
// Errors: ErrIncompatibleDictionaries (language or tag vocabulary
// mismatch), ErrPredictionSharded, ErrNoEntries (the result has no
// words), and nil-input / unknown-mode errors.
func MergeWithOptions(base *Dictionary, overlays []*Dictionary, opts MergeOptions) (*Dictionary, error) {
	if base == nil || base.d == nil {
		return nil, fmt.Errorf("morphology: merge: base dictionary is nil")
	}
	if opts.Mode != MergeAdd && opts.Mode != MergeReplace {
		return nil, fmt.Errorf("morphology: merge: unknown mode %d", int(opts.Mode))
	}
	ins := make([]*internal.Dictionary, len(overlays))
	for i, o := range overlays {
		if o == nil || o.d == nil {
			return nil, fmt.Errorf("morphology: merge: overlay %d is nil", i)
		}
		if err := checkMergeCompatible(base.d, o.d); err != nil {
			return nil, fmt.Errorf("morphology: merge: overlay %d: %w", i, err)
		}
		ins[i] = o.d
	}

	d, err := internal.MergeDictionaries(base.d, ins, internal.MergeOptions{
		Mode:              internal.MergeMode(opts.Mode),
		RebuildPrediction: opts.RebuildPrediction,
		Productive:        productive,
	})
	if err != nil {
		return nil, fmt.Errorf("morphology: merge: %w", err)
	}
	if dictionaryEmpty(d) {
		return nil, fmt.Errorf("morphology: merge: %w", ErrNoEntries)
	}

	info := &internal.BuildInfo{Source: "merge"}
	if base.d.Info != nil {
		info.SourceVersion = base.d.Info.SourceVersion
		info.Description = base.d.Info.Description
	}
	d.Info = info
	return &Dictionary{d: d}, nil
}

// checkMergeCompatible rejects overlays that can't share the base's
// language or tag vocabulary.
func checkMergeCompatible(base, overlay *internal.Dictionary) error {
	if base.Language != overlay.Language {
		return fmt.Errorf("%w: language %q differs from the base's %q", ErrIncompatibleDictionaries, overlay.Language, base.Language)
	}
	bn, on := tagSetNameOf(base), tagSetNameOf(overlay)
	if bn != on && tagmap.Known(bn) && tagmap.Known(on) {
		return fmt.Errorf("%w: tag set %q cannot be mixed into the base's %q", ErrIncompatibleDictionaries, on, bn)
	}
	return nil
}

func tagSetNameOf(d *internal.Dictionary) string {
	if d.TagSet == nil {
		return ""
	}
	return d.TagSet.Name
}

func dictionaryEmpty(d *internal.Dictionary) bool {
	for _, w := range d.Words {
		if !w.Empty() {
			return false
		}
	}
	return true
}
