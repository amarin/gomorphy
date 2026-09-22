package internal

import (
	"encoding/binary"
	"fmt"
)

// MergeMode is MergeDictionaries' word-level conflict policy; values
// mirror the public morphology.MergeMode one for one.
type MergeMode int

const (
	// MergeAdd takes an overlay word only if neither the base nor an
	// earlier overlay has it.
	MergeAdd MergeMode = iota
	// MergeReplace lets an overlay word's readings replace whatever the
	// word had so far (base or earlier overlay) — the last overlay wins.
	MergeReplace
)

// overlayReading is one reading of an overlay wordform, in the overlay's
// own id space.
type overlayReading struct {
	shard      int
	para, form uint16
}

// winner is the overlay whose readings a word gets in the output.
type winner struct {
	overlay  int
	readings []overlayReading
}

// decideOverlays folds overlays over the base in order and returns, for
// every word an overlay supplies to the output, the winning overlay and
// its readings. Merge(b,[o1,o2]) == Merge(Merge(b,[o1]),[o2]).
func decideOverlays(base *Dictionary, overlays []*Dictionary, mode MergeMode) (map[string]*winner, error) {
	winners := make(map[string]*winner)
	for oi, o := range overlays {
		words, err := wordReadings(o)
		if err != nil {
			return nil, fmt.Errorf("overlay %d: %w", oi, err)
		}
		for word, rs := range words {
			if mode == MergeAdd {
				if _, taken := winners[word]; taken || hasWord(base, word) {
					continue
				}
			}
			winners[word] = &winner{overlay: oi, readings: rs}
		}
	}
	return winners, nil
}

// wordReadings enumerates every wordform of d (decoded to plain text)
// with its readings, across all shards.
func wordReadings(d *Dictionary) (map[string][]overlayReading, error) {
	out := make(map[string][]overlayReading)
	for s, w := range d.Words {
		if w == nil {
			continue
		}
		var walkErr error
		w.Walk(func(key string, vals [][]byte) {
			if walkErr != nil {
				return
			}
			word, err := decodeKey(d.Alphabet, key)
			if err != nil {
				walkErr = err
				return
			}
			for _, v := range vals {
				if len(v) < 4 {
					continue
				}
				out[word] = append(out[word], overlayReading{
					shard: s,
					para:  binary.BigEndian.Uint16(v[:2]),
					form:  binary.BigEndian.Uint16(v[2:4]),
				})
			}
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return out, nil
}

// decodeKey turns a stored words.dawg key into plain text.
func decodeKey(a Alphabet, key string) (string, error) {
	if a == nil {
		return key, nil
	}
	word, err := a.Decode([]byte(key))
	if err != nil {
		return "", fmt.Errorf("decode DAWG key: %w", err)
	}
	return word, nil
}

// encodeKey turns plain text into a words.dawg key; ok is false when the
// alphabet can't encode it.
func encodeKey(a Alphabet, word string) (string, bool) {
	if a == nil {
		return word, true
	}
	b, err := a.Encode(word)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// hasWord reports whether word is an exact key of any of d's shards (no
// CharPolicy substitution).
func hasWord(d *Dictionary, word string) bool {
	for _, w := range d.Words {
		if shardHasWord(w, d.Alphabet, word) {
			return true
		}
	}
	return false
}

// shardHasWord reports whether word is an exact key of one words DAWG.
func shardHasWord(w *DAWG, a Alphabet, word string) bool {
	if w == nil || word == "" {
		return false
	}
	key, ok := encodeKey(a, word)
	if !ok {
		return false
	}
	idx := w.Follow(key, 0)
	return idx != 0 && w.HasPayloadChild(idx)
}
