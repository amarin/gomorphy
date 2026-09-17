// Package internal holds gomorphy's internal dictionary representation and
// on-disk format: TagSet, Paradigm, the DAWG engine and builder, and the
// sectioned GMOR container format. Not part of the public API — external
// code should use pkg/morphology instead.
package internal

// Dictionary — иммутабельный снимок словаря.
//
// Suffixes, Paradigms и Words — по одному элементу на шард; индекс 0 —
// единственный шард для несегментированных словарей (нет отдельного
// "нешардированного" представления). Шарды существуют потому, что
// suffix id адресуется uint16: словарь с суффиксов больше, чем помещается
// в один uint16-диапазон, делится на несколько шардов с независимыми
// id-пространствами (см. docs/superpowers/specs/2026-09-14-suffix-sharding-design.md).
// TagSet и Prefixes остаются общими для всех шардов.
type Dictionary struct {
	Language    string
	TagSet      *TagSet
	Suffixes    [][]string
	Prefixes    []string
	Paradigms   [][]Paradigm
	Words       []*DAWG
	Prediction  []*DAWG
	Probability *DAWG
	CharPolicy  *CharPolicy
	Alphabet    Alphabet // nil = raw UTF-8 keys (today's behavior, unchanged)
	Info        *BuildInfo
}

// NewDictionary собирает Dictionary из компонентов. suffixes, paradigms и
// words обязаны иметь одинаковую длину (число шардов) — паникует иначе,
// как и NewParadigm паникует на несовпадении длин своих частей. Prediction,
// Probability и Info остаются nil; Prediction/Probability заполняются
// импортёрами при чтении prediction-файлов, Info — импортёром (обычно
// только Source) или SaveTo (BuiltAt/LibraryVersion — при каждом
// сохранении).
func NewDictionary(
	language string,
	tagSet *TagSet,
	suffixes [][]string,
	prefixes []string,
	paradigms [][]Paradigm,
	words []*DAWG,
	charPolicy *CharPolicy,
) *Dictionary {
	if len(suffixes) != len(paradigms) || len(paradigms) != len(words) {
		panic("internal: NewDictionary shard slices (suffixes, paradigms, words) must have equal length")
	}
	return &Dictionary{
		Language:   language,
		TagSet:     tagSet,
		Suffixes:   suffixes,
		Prefixes:   prefixes,
		Paradigms:  paradigms,
		Words:      words,
		CharPolicy: charPolicy,
	}
}
