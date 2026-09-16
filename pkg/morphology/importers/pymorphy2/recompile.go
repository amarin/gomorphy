// Package pymorphy2's RecompileDense rebuilds an imported dictionary's
// words.dawg under a dense 1-byte alphabet. See
// docs/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md.
package pymorphy2

import (
	"encoding/binary"
	"fmt"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// RecompileDense imports dir like ImportFromDir, then rebuilds Words[0]
// (and only Words[0] — Paradigms/Suffixes/Prefixes/Prediction/Probability
// are copied through unchanged, since they don't depend on words.dawg's
// key encoding) under a dense 1-byte alphabet built from the dictionary's
// own wordforms. The result's Parse() must return identical readings to
// ImportFromDir's, for any word the source dictionary itself resolves.
func RecompileDense(dir string) (*internal.Dictionary, error) {
	d, err := ImportFromDir(dir)
	if err != nil {
		return nil, err
	}

	var words []string
	var keys []string
	var values []uint32
	d.Words[0].Walk(func(word string, vals [][]byte) {
		words = append(words, word)
		for _, v := range vals {
			if len(v) < 4 {
				continue // matches Dictionary.reading's own guard, parse.go
			}
			keys = append(keys, word)
			values = append(values, binary.BigEndian.Uint32(v[:4]))
		}
	})

	alphabet, err := internal.NewDenseAlphabet(1, words)
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: recompile: build alphabet: %w", err)
	}

	encodedKeys := make([]string, len(keys))
	for i, k := range keys {
		enc, err := alphabet.Encode(k)
		if err != nil {
			return nil, fmt.Errorf("pymorphy2: recompile: encode %q: %w", k, err)
		}
		encodedKeys[i] = string(enc)
	}

	dense, err := internal.BuildDAWGWithValues(encodedKeys, values)
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: recompile: build DAWG: %w", err)
	}

	d.Words[0] = dense
	d.Alphabet = alphabet
	return d, nil
}
