package internal

import "unicode/utf8"

// Item — результат поиска: найденный ключ и его payload-значения.
type Item struct {
	Key    string
	Values [][]byte
}

// SimilarItems ищет ключ с учётом подмен символов CharPolicy. Для 'е→ё'
// находит словоформы, различающиеся е/ё, например "ежик" и "ёжик".
func (d *DAWG) SimilarItems(key string, pol *CharPolicy) []Item {
	return d.similarItemsRecursive("", []rune(key), 0, pol)
}

func (d *DAWG) similarItemsRecursive(prefix string, key []rune, index uint32, pol *CharPolicy) []Item {
	var items []Item

	startPos := utf8.RuneCountInString(prefix)
	endPos := len(key)
	wordPos := startPos

	for wordPos < endPos {
		r := key[wordPos]
		if pol != nil {
			for _, sub := range pol.Substitutions {
				if r == sub.From {
					if next := d.FollowRune(sub.To, index); next != 0 {
						newPrefix := prefix + string(key[startPos:wordPos]) + string(sub.To)
						items = append(items, d.similarItemsRecursive(newPrefix, key, next, pol)...)
					}
				}
			}
		}
		if index = d.FollowRune(r, index); index == 0 {
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
