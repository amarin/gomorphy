// MultiDictionary aggregates Parse/Lemma/Close across an arbitrary set of
// already-open dictionaries. See
// docs/superpowers/specs/2026-09-16-multi-dict-design.md.
package morphology

import (
	"errors"
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
