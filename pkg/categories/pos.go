package categories

import (
	"strings"
)

// Части речи.
// Категория слов языка, определяемая морфологическими и синтаксическими признаками.
// https://ru.wikipedia.org/wiki/Часть_речи.
type POS string

// Наименование грамматической категории падежа.
const CategoryNamePos CategoryName = "pos"

// Словарь всех определяемых частей речи.
type PosDictionary []POS

// Словарь частей речи возвращает тип грамматической категории.
func (c PosDictionary) CategoryName() CategoryName {
	return CategoryNamePos
}

// Словарь частей речи может быть представлен в виде не типизированного словаря.
func (c PosDictionary) Slice() []interface{} {
	res := make([]interface{}, 0)
	for _, item := range c {
		res = append(res, item)
	}

	return res
}

// Получить константное значение по строковому представлению.
// Возвращает nil если значение не найдено.
func (c PosDictionary) ByString(text string) *POS {
	for _, pos := range c {
		if strings.EqualFold(text, string(pos)) {
			return &pos
		}
	}

	return nil
}

// Константные значения частей речи для использования в коде.
// https://ru.wikipedia.org/wiki/Части_речи_в_русском_языке,
// Значения констант совпадают с OpenCorpora
// Названия переменных английские с префиксом Pos-.
const (
	PosNoun            POS = "NOUN" // имя существительное
	PosAdjectiveFull   POS = "ADJF" // имя прилагательное (полное)	хороший
	PosAdjectiveShort  POS = "ADJS" // имя прилагательное (краткое)	хорош
	PosComparative     POS = "COMP" // компаратив лучше, получше, выше
	PosVerb            POS = "VERB" // глагол (личная форма)	говорю, говорит, говорил
	PosInfinitive      POS = "INFN" // глагол (инфинитив)	говорить, сказать
	PosParticipleFull  POS = "PRTF" // причастие (полное)	прочитавший, прочитанная
	PosParticipleShort POS = "PRTS" // причастие (краткое)	прочитана
	PosGerund          POS = "GRND" // деепричастие	прочитав, рассказывая
	PosNumeric         POS = "NUMR" // числительное	три, пятьдесят
	PosAdverb          POS = "ADVB" // наречие	круто
	PosProNoun         POS = "NPRO" // местоимение-существительное	он
	PosPredicate       POS = "PRED" // предикатив	некогда
	PosPreposition     POS = "PREP" // предлог	в
	PosConjunction     POS = "CONJ" // союз	и
	PosParticle        POS = "PRCL" // частица	бы, же, лишь
	PosInterjection    POS = "INTJ" // междометие	ой
)

// Список определяемых частей речи содержит все константные значения.
var KnownPoses = PosDictionary{
	PosNoun,
	PosAdjectiveFull,
	PosAdjectiveShort,
	PosComparative,
	PosVerb,
	PosInfinitive,
	PosParticipleFull,
	PosParticipleShort,
	PosGerund,
	PosNumeric,
	PosAdverb,
	PosProNoun,
	PosPredicate,
	PosPreposition,
	PosConjunction,
	PosParticle,
	PosInterjection,
}

// Сущности, являющиеся частями речи
// предоставляют и сохраняют значение части речи с помощью методов POS() и SetPOS() соответственно.
type PartOfSpeechProvider interface {
	POS() POS
	SetPOS(POS)
}
