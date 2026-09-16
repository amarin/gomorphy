package morphology

import (
	"sort"
	"sync"
	"unicode/utf8"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// FuzzyMatch — найденное по нечёткому поиску слово и расстояние до запроса.
type FuzzyMatch struct {
	Word     string
	Distance int
}

// Fuzzy возвращает слова словаря в пределах расстояния Левенштейна maxDist
// от word (метрика по рунам: вставка/удаление/замена одной руны = 1,
// «ё/е» — одна замена). Результат отсортирован по (расстояние, слово),
// дубликаты слова (несколько чтений, в том числе из разных шардов)
// схлопнуты. Отрицательное maxDist трактуется как 0 (точный поиск).
// Пустой результат — слов нет.
//
// Словари с плотным алфавитом (Dictionary.Alphabet != nil, например
// открытые через OpenPyMorphyDense) пока не поддерживаются: внутренний
// обход декодирует байты DAWG как raw UTF-8, что для плотного кода даёт
// не ошибку, а тихий мусор. Для такого словаря Fuzzy возвращает nil.
func (x *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch {
	if x.d.Alphabet != nil {
		return nil
	}
	if maxDist < 0 {
		maxDist = 0
	}
	return dedupeFuzzy(x.fuzzyWalk(word, maxDist))
}

// FuzzyTop возвращает до maxWords ближайших слов, упорядоченных по
// (расстояние, слово). Расстояние расширяется итеративно от 0 до верхней
// границы (len(query)+наибольшая длина слова в рунах среди всех шардов),
// пока не набраны maxWords слов или не пройден весь словарь. maxWords ≤ 0
// — точный поиск (слово само по себе, либо пусто).
//
// Как и Fuzzy, не поддерживает словари с плотным алфавитом
// (Dictionary.Alphabet != nil) — возвращает nil.
func (x *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch {
	if x.d.Alphabet != nil {
		return nil
	}
	if maxWords <= 0 {
		return x.Fuzzy(word, 0)
	}

	maxRunes := x.maxWordRunes()
	if maxRunes == 0 {
		return nil
	}

	bound := utf8.RuneCountInString(word) + maxRunes
	seen := make(map[string]bool, maxWords)
	var out []FuzzyMatch
	for dist := 0; dist <= bound && len(out) < maxWords; dist++ {
		for _, m := range x.fuzzyWalk(word, dist) {
			if seen[m.Word] {
				continue
			}
			seen[m.Word] = true
			out = append(out, m)
			if len(out) >= maxWords {
				break
			}
		}
	}
	return out
}

func dedupeFuzzy(matches []FuzzyMatch) []FuzzyMatch {
	if len(matches) == 0 {
		return nil
	}
	out := make([]FuzzyMatch, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		if seen[m.Word] {
			continue
		}
		seen[m.Word] = true
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// fuzzyWalk обходит каждый шард параллельно (по горутине на шард) и
// склеивает результаты. Шарды — независимые DAWG, поэтому обходы не
// делят изменяемое состояние.
func (x *Dictionary) fuzzyWalk(word string, k int) []FuzzyMatch {
	results := make([][]FuzzyMatch, len(x.d.Words))

	var wg sync.WaitGroup
	for shard, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		wg.Add(1)
		go func(shard int, dawg *internal.DAWG) {
			defer wg.Done()
			results[shard] = fuzzyWalkShard(dawg, word, k)
		}(shard, dawg)
	}
	wg.Wait()

	var out []FuzzyMatch
	for _, r := range results {
		out = append(out, r...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// fuzzyWalkShard — один проход совместного обхода одного шарда DAWG и
// banded DP Левенштейна.
func fuzzyWalkShard(words *internal.DAWG, word string, k int) []FuzzyMatch {
	f := &fuzzySearch{
		words: words,
		q:     []rune(word),
		k:     k,
		path:  make([]byte, 0, 32),
	}

	row := f.rowFor(0)
	for j := range row {
		row[j] = j
	}
	f.visit(0, 0, row)
	return f.out
}

// fuzzySearch переносит состояние одного обхода: rows[depth] — DP-строка
// после depth рун пути, path — байты текущего пути DAWG.
type fuzzySearch struct {
	words *internal.DAWG
	q     []rune
	k     int
	rows  [][]int
	path  []byte
	out   []FuzzyMatch
}

func (f *fuzzySearch) visit(state uint32, depth int, row []int) {
	if f.words.HasPayloadChild(state) {
		if dist := row[len(f.q)]; dist <= f.k {
			f.out = append(f.out, FuzzyMatch{Word: string(f.path), Distance: dist})
		}
	}

	f.words.ForEachChild(state, func(label byte, next uint32) {
		if label == internal.PayloadSeparator {
			return
		}
		start := len(f.path)
		f.path = append(f.path, label)
		f.explore(next, depth, row, start)
		f.path = f.path[:start]
	})
}

// explore завершает текущую руну (путь из байтов с позиции start) и
// применяет DP-переход; продолжает рекурсию, если строка ещё в пределах k.
func (f *fuzzySearch) explore(state uint32, depth int, row []int, start int) {
	tail := f.path[start:]

	if !utf8.FullRune(tail) {
		f.words.ForEachChild(state, func(label byte, next uint32) {
			if label == internal.PayloadSeparator {
				return
			}
			f.path = append(f.path, label)
			f.explore(next, depth, row, start)
			f.path = f.path[:len(f.path)-1]
		})
		return
	}

	r, _ := utf8.DecodeRune(tail)
	if r == utf8.RuneError {
		return
	}

	nrow := f.nextRow(depth, row, r)
	if minRow(nrow) <= f.k {
		f.visit(state, depth+1, nrow)
	}
}

func (f *fuzzySearch) nextRow(depth int, row []int, r rune) []int {
	nrow := f.rowFor(depth + 1)
	nrow[0] = row[0] + 1

	for j := 1; j <= len(f.q); j++ {
		cost := 1
		if f.q[j-1] == r {
			cost = 0
		}
		nrow[j] = min3(row[j]+1, nrow[j-1]+1, row[j-1]+cost)
	}
	return nrow
}

func (f *fuzzySearch) rowFor(depth int) []int {
	for len(f.rows) <= depth {
		f.rows = append(f.rows, make([]int, len(f.q)+1))
	}
	return f.rows[depth]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func minRow(row []int) int {
	m := row[0]
	for _, v := range row[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// maxWordRunes возвращает наибольшую длину словоформы словаря в рунах,
// среди всех шардов.
func (x *Dictionary) maxWordRunes() int {
	best := 0
	for _, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		if m := maxWordRunesInShard(dawg); m > best {
			best = m
		}
	}
	return best
}

// maxWordRunesInShard обходит DAG одного шарда. Обход дедуплицирует узлы
// по наибольшей достигнутой глубине — иначе разделяемые суффиксы
// считались бы экспоненциально. Продолжающие байты UTF-8 (0x80–0xBF)
// руну не добавляют.
func maxWordRunesInShard(words *internal.DAWG) int {
	seen := make(map[uint32]int)
	var rec func(state uint32, runes int)
	rec = func(state uint32, runes int) {
		if prev, ok := seen[state]; ok && prev >= runes {
			return
		}
		seen[state] = runes
		words.ForEachChild(state, func(label byte, next uint32) {
			if label == internal.PayloadSeparator {
				return
			}
			step := 1
			if label >= 0x80 && label <= 0xBF {
				step = 0
			}
			rec(next, runes+step)
		})
	}
	rec(0, 0)

	max := 0
	for _, depth := range seen {
		if depth > max {
			max = depth
		}
	}
	return max
}
