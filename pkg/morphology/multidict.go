// MultiDictionary aggregates Parse/Lemma/Close across an arbitrary set of
// already-open dictionaries. See
// docs/superpowers/specs/2026-09-16-multi-dict-design.md.
package morphology

import (
	"errors"
	"sort"
	"sync"
)

// MultiDictionary — набор независимо открытых словарей, опрашиваемых как
// единое целое. Каждый *Dictionary в наборе сохраняет свой собственный
// жизненный цикл (mmap и т.п.) — MultiDictionary не открывает и не
// импортирует ничего сама, только агрегирует Parse/Lemma и владеет
// закрытием всего набора разом.
type MultiDictionary struct {
	dicts []*Dictionary
}

// NewMultiDictionary оборачивает уже открытые словари в единый набор.
// Порядок dicts фиксирует индексацию Reading.Dict/LemmaRef.Dict и порядок
// склейки результатов Parse/Lemma — оба всегда в порядке регистрации, не
// пересортировываются.
func NewMultiDictionary(dicts ...*Dictionary) *MultiDictionary {
	return &MultiDictionary{dicts: dicts}
}

// Len возвращает число словарей в наборе.
func (m *MultiDictionary) Len() int { return len(m.dicts) }

// DictInfo возвращает диагностические метаданные словаря с индексом i (тот
// же индекс, что несёт Reading.Dict/LemmaRef.Dict), или nil — если индекс
// вне диапазона, или у этого словаря нет секции info (см. Dictionary.Info).
func (m *MultiDictionary) DictInfo(i int) *BuildInfo {
	if i < 0 || i >= len(m.dicts) {
		return nil
	}
	return m.dicts[i].Info()
}

// Parse разбирает word во всех словарях набора параллельно (по горутине на
// словарь — тот же паттерн, что Dictionary.exact уже использует для
// шардов внутри одного словаря). Результат — конкатенация Parse каждого
// словаря в порядке регистрации набора, с проставленным Reading.Dict; без
// какой-либо сортировки или дедупликации между словарями сверх того, что
// каждый Dictionary.Parse уже делает сам внутри себя. nil, если ни один
// словарь не дал чтений.
func (m *MultiDictionary) Parse(word string) []Reading {
	results := make([][]Reading, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			readings := d.Parse(word)
			for j := range readings {
				readings[j].Dict = i
			}
			results[i] = readings
		}(i, d)
	}
	wg.Wait()

	var out []Reading
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Lemma разбирает word во всех словарях набора и возвращает начальные
// формы. Та же конкатенация-в-порядке-регистрации семантика, что и Parse.
func (m *MultiDictionary) Lemma(word string) []LemmaRef {
	results := make([][]LemmaRef, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			refs := d.Lemma(word)
			for j := range refs {
				refs[j].Dict = i
			}
			results[i] = refs
		}(i, d)
	}
	wg.Wait()

	var out []LemmaRef
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Fuzzy ищет слова в пределах расстояния Левенштейна maxDist от word во
// всех словарях набора параллельно. Результат — конкатенация Fuzzy
// каждого словаря в порядке регистрации набора, с проставленным
// FuzzyMatch.Dict; без сортировки/дедупликации между словарями сверх
// того, что каждый Dictionary.Fuzzy уже делает сам внутри себя (та же
// политика, что у Parse/Lemma) — Fuzzy не ограничивает число результатов,
// поэтому объединение "как есть" корректно.
func (m *MultiDictionary) Fuzzy(word string, maxDist int) []FuzzyMatch {
	results := make([][]FuzzyMatch, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			matches := d.Fuzzy(word, maxDist)
			for j := range matches {
				matches[j].Dict = i
			}
			results[i] = matches
		}(i, d)
	}
	wg.Wait()

	var out []FuzzyMatch
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// FuzzyTop возвращает до maxWords ближайших слов по всему набору,
// упорядоченных по (расстояние, слово). В отличие от Parse/Lemma/Fuzzy,
// maxWords — потолок на общий результат, не на словарь, поэтому здесь
// нужно настоящее слияние: у каждого словаря запрашивается его
// собственный top-maxWords (этого достаточно — любой кандидат глобального
// top-maxWords обязан входить и в top-maxWords своего словаря, иначе в
// этом же словаре нашлось бы maxWords кандидатов не хуже него), после
// чего все кандидаты сортируются заново и обрезаются до maxWords.
// maxWords <= 0 — точный поиск, как и у Dictionary.FuzzyTop.
func (m *MultiDictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch {
	if maxWords <= 0 {
		return m.Fuzzy(word, 0)
	}

	results := make([][]FuzzyMatch, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			matches := d.FuzzyTop(word, maxWords)
			for j := range matches {
				matches[j].Dict = i
			}
			results[i] = matches
		}(i, d)
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
	if len(out) > maxWords {
		out = out[:maxWords]
	}
	return out
}

// Close закрывает каждый словарь набора (Dictionary.Close — no-op для
// словарей, открытых не через Open), агрегируя все ошибки через
// errors.Join. После Close набор использовать нельзя, как и его словари.
func (m *MultiDictionary) Close() error {
	var errs []error
	for _, d := range m.dicts {
		if err := d.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
