package categories

import (
	"strings"
)

// Число́ (в грамматике) — грамматическая категория, выражающая количественную характеристику объектов,
// их наличие в одном или большем количестве экземпляров.
type Number string

// Наименование грамматической категории числа.
const CategoryNameNumber CategoryName = "number"

// Словарь определяемых категорий числа.
type NumbersDictionary []Number

// Словарь определяемых категорий числа возвращает тип грамматической категории.
func (n NumbersDictionary) CategoryName() CategoryName {
	return CategoryNameNumber
}

// Словарь определяемых родов может быть представлен в виде не типизированного словаря.
func (n NumbersDictionary) Slice() []interface{} {
	res := make([]interface{}, 0)
	for _, item := range n {
		res = append(res, item)
	}

	return res
}

// Получить константное значение по строковому представлению.
// Возвращает nil если значение не найдено.
func (n NumbersDictionary) ByString(text string) *Number {
	for _, number := range n {
		if strings.EqualFold(text, string(number)) {
			return &number
		}
	}

	return nil
}

// Части речи, обладающие числом.
type NumberProvider interface {
	GetNumber() Number
	SetNumber(Number)
}

// Константные значения категорий числа для использования в коде.
// https://ru.wikipedia.org/wiki/Грамматическое_число
// Значения совпадают с OpenCorpora.
const (
	// единственное число. хомяк, говорит
	NumberSingular Number = "sing"
	// множественное число. хомяки, говорят
	NumberPlural Number = "plur"
)

// KnownNumbers задаёт список определяемых категорий числа.
var KnownNumbers = NumbersDictionary{NumberSingular, NumberPlural}

// type NumberExtra string
//
// // Дополнительная информация о числе.
// type NumberExtraProvider interface {
// 	GetNumberExtra() NumberExtra
// 	SetNumberExtra(NumberExtra)
// }
//
// const (
// 	// Нет специальных отметок
// 	NoNumberExtra NumberExtra = ""
// 	// Некоторые имена существительные употребляются только во множественном числе;
// 	// им проставлена пометка Pltm (“Pluralia tantum”)
// 	PluraliaTantum NumberExtra = "Pltm plur"
// 	// Существуют также существительные, употребляемые только в единственном числе;
// 	// им проставлена пометка Sgtm (“Singularia tantum”)
// 	SingulariaTantum NumberExtra = "Sgtm sing"
// )

var NumberStrings []string
var MapStringToNumber map[string]Number

func init() {
	NumberStrings = make([]string, 0)
	MapStringToNumber = make(map[string]Number)

	for _, n := range KnownNumbers {
		NumberStrings = append(NumberStrings, string(n))
		MapStringToNumber[string(n)] = n
	}
}
