package internal

import "unicode/utf8"

// Item — результат поиска: найденный ключ и его payload-значения.
type Item struct {
	Key    string
	Values [][]byte
}

// SimilarItems ищет ключ с учётом подмен символов CharPolicy. Для 'е→ё'
// находит словоформы, различающиеся е/ё, например "ежик" и "ёжик".
// alphabet кодирует каждую руну перед переходом по DAWG (nil — сырой
// UTF-8, поведение не отличается от предыдущей версии); возвращаемый
// Item.Key — всегда исходный человекочитаемый текст и не требует
// декодирования независимо от alphabet.
func (d *DAWG) SimilarItems(key string, pol *CharPolicy, alphabet Alphabet) []Item {
	return d.similarItemsRecursive("", []rune(key), 0, pol, alphabet)
}

// followRuneVia переходит по одной руне r из index, кодируя её через
// alphabet. alphabet == nil сохраняет старое поведение (FollowRune, сырой
// UTF-8). Возвращает 0, если alphabet не может закодировать r (руна вне
// корпуса, на котором был построен алфавит) — тот же контракт "нет
// перехода", что и у FollowByte/FollowRune.
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
