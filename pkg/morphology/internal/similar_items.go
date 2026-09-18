package internal

import "unicode/utf8"

// Item is a search result: the found key and its payload values.
type Item struct {
	Key    string
	Values [][]byte
}

// SimilarItems looks up a key while accounting for CharPolicy character
// substitutions. For 'е→ё', it finds wordforms differing only in е/ё,
// e.g. "ежик" and "ёжик". alphabet encodes each rune before following an
// edge in the DAWG (nil = raw UTF-8, same behavior as the previous
// version); the returned Item.Key is always the original human-readable
// text and never needs decoding regardless of alphabet.
func (d *DAWG) SimilarItems(key string, pol *CharPolicy, alphabet Alphabet) []Item {
	return d.similarItemsRecursive("", []rune(key), 0, pol, alphabet)
}

// followRuneVia follows one rune r from index, encoding it via alphabet.
// alphabet == nil preserves the old behavior (FollowRune, raw UTF-8).
// Returns 0 if alphabet cannot encode r (a rune outside the corpus the
// alphabet was built from) — the same "no edge" contract as
// FollowByte/FollowRune.
func (d *DAWG) followRuneVia(alphabet Alphabet, r rune, index uint32) uint32 {
	if alphabet == nil {
		return d.FollowRune(r, index)
	}
	code, err := alphabet.Encode(string(r))
	if err != nil {
		return 0
	}
	return d.followBytes(code, index)
}

func (d *DAWG) similarItemsRecursive(prefix string, key []rune, index uint32, pol *CharPolicy, alphabet Alphabet) []Item {
	var items []Item

	startPos := utf8.RuneCountInString(prefix)
	endPos := len(key)
	wordPos := startPos

	for wordPos < endPos {
		r := key[wordPos]
		if pol != nil {
			for _, sub := range pol.Substitutions {
				if r == sub.From {
					if next := d.followRuneVia(alphabet, sub.To, index); next != 0 {
						newPrefix := prefix + string(key[startPos:wordPos]) + string(sub.To)
						items = append(items, d.similarItemsRecursive(newPrefix, key, next, pol, alphabet)...)
					}
				}
			}
		}
		if index = d.followRuneVia(alphabet, r, index); index == 0 {
			break
		}
		wordPos++
	}

	if wordPos == endPos {
		if sepIndex := d.FollowByte(PayloadSeparator, index); sepIndex != 0 {
			foundKey := prefix + string(key[startPos:])
			items = append([]Item{{Key: foundKey, Values: d.ValuesForIndex(sepIndex)}}, items...)
		}
	}

	return items
}
