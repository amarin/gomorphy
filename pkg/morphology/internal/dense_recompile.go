package internal

import (
	"encoding/binary"
	"fmt"
)

// RecompileDense rebuilds every shard of d.Words under one dense 1-byte
// alphabet shared across the whole dictionary — not a separate alphabet
// per shard, see docs/en/implementation/pymorphy2-dense-alphabet.md's
// "Agreed design decisions", item 1 — built from the union of every
// shard's wordforms. Paradigms/Suffixes/Prefixes/Prediction/Probability
// are left untouched: none of them depend on words.dawg's key encoding
// (see the same doc). d is mutated in place (Words and Alphabet); on
// error d is left unmodified.
//
// Source-agnostic: used by both the pymorphy2 importer (always exactly
// one shard) and the OpenCorpora importer (one or more shards, per
// docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md).
func RecompileDense(d *Dictionary) error {
	type shardData struct {
		keys   []string
		values []uint32
	}
	shards := make([]shardData, len(d.Words))
	var allWords []string

	for i, words := range d.Words {
		var sd shardData
		words.Walk(func(word string, vals [][]byte) {
			allWords = append(allWords, word)
			for _, v := range vals {
				if len(v) < 4 {
					continue // matches Dictionary.reading's own guard, parse.go
				}
				sd.keys = append(sd.keys, word)
				sd.values = append(sd.values, binary.BigEndian.Uint32(v[:4]))
			}
		})
		shards[i] = sd
	}

	alphabet, err := NewDenseAlphabet(1, allWords)
	if err != nil {
		return fmt.Errorf("internal: recompile dense: build alphabet: %w", err)
	}

	newWords := make([]*DAWG, len(d.Words))
	for i, sd := range shards {
		encodedKeys := make([]string, len(sd.keys))
		for j, k := range sd.keys {
			enc, err := alphabet.Encode(k)
			if err != nil {
				return fmt.Errorf("internal: recompile dense: shard %d: encode %q: %w", i, k, err)
			}
			encodedKeys[j] = string(enc)
		}
		dense, err := BuildDAWGWithValues(encodedKeys, sd.values)
		if err != nil {
			return fmt.Errorf("internal: recompile dense: shard %d: build DAWG: %w", i, err)
		}
		newWords[i] = dense
	}

	d.Words = newWords
	d.Alphabet = alphabet
	return nil
}
