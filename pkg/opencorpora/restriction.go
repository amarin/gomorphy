package opencorpora

// Грамматическая категория как составная часть ограничения.
// Может принимать значения:
// - lemma: слово не может принимать данную категорию при выполнении каких-либо условий (см. Требование использования)
// - form: слово не может принимать данную форму категории при выполнении каких-либо условий (см. Требование использования)
type RestrictionItem struct {
	TypeAttr string `xml:"type,attr"`
}

// Требование использования граммем описывают соотношения между грамматическими категориями,
// например, необходимость указания категории падежа для существительных
// или возможность использования общего рода для существительных же.
type Restriction struct {
	// Тип аттрибута задаёт тип ограничения совместного использования категорий
	//  - obligatory: обязательная категория left для категории right
	//  - maybe: возможная категория left для категории right
	//  - forbidden: запретная категория left для категории right
	TypeAttr string `xml:"type,attr"`
	// Тип настройки ограничения:
	// - 0: ручная настройка
	// - 1: автоматическая настройка
	AutoAttr int `xml:"auto,attr"`
	// Категория, с которой должна, может или не может использоваться категория Right
	Left *RestrictionItem `xml:"left"`
	// Категория, которая должна, может или не может использоваться с категорией Left
	Right *RestrictionItem `xml:"right"`
}

// Restrictions Список ограничений задаёт все известные ограничения и требования совместного использования категорий.
type Restrictions struct {
	// Список ограничений
	Restr []*Restriction `xml:"restr"`
}

// RestrType ...
// type RestrType string
