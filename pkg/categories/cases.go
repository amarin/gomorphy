package categories

import (
	"strings"
)

// Падеж это словоизменительная грамматическая категория именных и местоимённых частей речи
// (существительных, прилагательных, числительных) и близких к ним гибридных частей речи
// (причастий, герундиев, инфинитивов и проч.),
// выражающая их синтаксическую и/или семантическую роль в предложении.
// https://ru.wikipedia.org/wiki/Падеж
type Case string

// Наименование грамматической категории падежа.
const CategoryNameCase CategoryName = "case"

// Словарь всех определяемых падежей.
type CaseDictionary []Case

// Словарь падежей возвращает тип грамматической категории.
func (c CaseDictionary) CategoryName() CategoryName {
	return CategoryNameCase
}

// Словарь падежей может быть представлен в виде не типизированного словаря.
func (c CaseDictionary) Slice() []interface{} {
	res := make([]interface{}, 0)
	for _, item := range c {
		res = append(res, item)
	}

	return res
}

// Получить константное значение по строковому представлению.
// Возвращает nil если значение не найдено.
func (c CaseDictionary) ByString(text string) *Case {
	for _, pos := range c {
		if strings.ToUpper(text) == strings.ToUpper(string(pos)) {
			return &pos
		}
	}
	return nil
}

// Константные значения падежей для использования в коде.
// Названия переменных на латыни согласно https://ru.wikipedia.org/wiki/Падеж,
// Значения совпадают с OpenCorpora
const (
	NominativusCase      Case = "nomn" // Номинатив (именительный). Кто? Что? хомяк ест
	GenitivusCase        Case = "gent" // Генитив (родительный). Кого? Чего?	у нас нет хомяка
	DativusCase          Case = "datv" // Датив (дательный). Кому? Чему?	сказать хомяку спасибо
	AccusativusCase      Case = "accs" // Аккузатив (винительный). Кого? Что? хомяк читает книгу
	InstrumentalisCase   Case = "ablt" // Инструментатив (творительный). Кем? Чем? зерно съедено хомяком
	PraepositionalisCase Case = "loct" // Препозитив (предложный). О ком? О чём? и т.п. хомяка несут в корзинке
	VocativusCase        Case = "voct" // Вокатив (звательный). Его формы используются при обращении к человеку. Саш, пойдем в кино.
	PartitivusCase       Case = "gen2" // Партитив (количественно-отделительный, второй родительный, частичный). Ложка сахару (GenitivusCase - производство сахара); стакан яду (GenitivusCase - нет яда)
	TranslativusCase     Case = "acc2" // Транслатив (превратительный, включительный падеж, второй винительный). Записался в солдаты избрать в президенты
	LocativusCase        Case = "loc2" // Локатив (местный падеж, второй предложный). Я у него в долгу (PraepositionalisCase - напоминать о долге); висит в шкафу (PraepositionalisCase - монолог о шкафе); весь в снегу (PraepositionalisCase - писать о снеге)
)

// Список определяемых падежей содержит все константные значения
var KnownCases = CaseDictionary{
	NominativusCase,
	GenitivusCase,
	DativusCase,
	AccusativusCase,
	InstrumentalisCase,
	PraepositionalisCase,
	VocativusCase,
	PartitivusCase,
	TranslativusCase,
	LocativusCase,
}

// Сущности, определяющие части речи, обладающие падежом,
// предоставляют и сохраняют значение падежа с помощью методов Case() и SetCase() соответственно
type CaseProvider interface {
	// Получить значение падежа
	Case() Case
	// Установить значение падежа
	SetCase(Case)
}

var MapStringToCase map[string]Case
var CaseStrings []string

func init() {
	CaseStrings = make([]string, 0)
	MapStringToCase = make(map[string]Case)
	for _, c := range KnownCases {
		CaseStrings = append(CaseStrings, string(c))
		MapStringToCase[string(c)] = c
	}
}
