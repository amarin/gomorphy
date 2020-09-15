package categories

import (
	"strings"
)

// Род в грамматике — категория, представляющая распределение слов и форм по классам,
// традиционно соотносимым с признаками пола или их отсутствием.
// Род характеризует различные части речи, являясь для них словоизменительной категорией.
// https://ru.wikipedia.org/wiki/Род_(лингвистика)
type Gender string

// Наименование грамматической категории рода.
const CategoryNameGender CategoryName = "gender"

// Словарь всех определяемых родов.
type GenderDictionary []Gender

// Словарь определяемых родов возвращает тип грамматической категории.
func (g GenderDictionary) CategoryName() CategoryName {
	return CategoryNameGender
}

// Словарь определяемых родов может быть представлен в виде не типизированного словаря.
func (g GenderDictionary) Slice() []interface{} {
	res := make([]interface{}, 0)
	for _, item := range g {
		res = append(res, item)
	}

	return res
}

// Получить константное значение по строковому представлению.
// Возвращает nil если значение не найдено.
func (g GenderDictionary) ByString(text string) *Gender {
	for _, gender := range g {
		if strings.ToUpper(text) == strings.ToUpper(string(gender)) {
			return &gender
		}
	}

	return nil
}

// Константные значения падежей для использования в коде.
// https://ru.wikipedia.org/wiki/Род_(лингвистика)
// Значения совпадают с OpenCorpora
const (
	// мужской род
	GenderMasculine Gender = "masc"
	// женский род
	GenderFeminine Gender = "femn"
	// средний род
	GenderNeutral Gender = "neut"
	// В русском языке существует понятие “общего рода”;
	// некоторые слова могут употребляться применительно к людям мужского или женского пола:
	// “он бедный сирота”, “она бедная сирота”.
	// Таким словам проставлена пометка Ms-f
	GenderCommon Gender = "Ms-f"
	// Существуют также существительные, у которых род не выражен; им проставлена пометка GNdr
	GenderAbsent Gender = "GNdr"
)

// Список определяемых родов содержит все константные значения
var KnownGenders = GenderDictionary{GenderMasculine, GenderFeminine, GenderNeutral, GenderCommon, GenderAbsent}

// Расширенная информация о роде
type GenderExtend string

// Части речи, обладающие родом, реализуют стандартный интерфейс
type GenderProvider interface {
	// Получить род
	Gender() Gender
	// Установить род
	SetGender(Gender)
}
