package internal

import "fmt"

// Paradigm — шаблон склонения/спряжения. Плоский массив uint16:
//
//	[suffix_0..suffix_N-1 | tag_0..tag_N-1 | prefix_0..prefix_N-1]
//
// Форма i парадигмы описывается тройкой (Suffix(i), Tag(i), Prefix(i)).
type Paradigm struct {
	data []uint16
}

// NewParadigm собирает Paradigm из трёх частей одинаковой длины.
func NewParadigm(suffixes, tags, prefixes []uint16) Paradigm {
	if len(suffixes) != len(tags) || len(tags) != len(prefixes) {
		panic("internal: paradigm parts must have equal length")
	}
	data := make([]uint16, 0, len(suffixes)*3)
	data = append(data, suffixes...)
	data = append(data, tags...)
	data = append(data, prefixes...)
	return Paradigm{data: data}
}

// NewParadigmFromData создаёт Paradigm из плоских данных в формате pymorphy2
// paradigms.array: [N суффиксов | N тегов | N префиксов]. Длина обязана
// делиться на 3. Данные не копируются.
func NewParadigmFromData(data []uint16) (Paradigm, error) {
	if len(data)%3 != 0 {
		return Paradigm{}, fmt.Errorf("internal: paradigm length %d is not divisible by 3", len(data))
	}
	return Paradigm{data: data}, nil
}

// Len — число форм в парадигме.
func (p Paradigm) Len() int {
	return len(p.data) / 3
}

// Suffix — id суффикса формы i.
func (p Paradigm) Suffix(i int) uint16 {
	return p.data[i]
}

// Tag — id тега формы i.
func (p Paradigm) Tag(i int) uint16 {
	return p.data[p.Len()+i]
}

// Prefix — id префикса формы i.
func (p Paradigm) Prefix(i int) uint16 {
	return p.data[2*p.Len()+i]
}

// Data возвращает плоские данные парадигмы (для сериализации).
func (p Paradigm) Data() []uint16 {
	return p.data
}
