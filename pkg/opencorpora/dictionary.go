package opencorpora

// Словарь OpenCorpora является корневым контейнером XML-представления словаря.
// Словарь доступен для загрузки по адресу http://opencorpora.org/?page=downloads
// Описание формата http://opencorpora.org/?page=export
// Словарь представляет собой файл XML в кодировке utf-8.
// Сам словарь доступен по одной из ссылок:
// - Упакованный архив BZip2: http://opencorpora.org/files/export/dict/dict.opcorpora.xml.bz2
// - Упакованный архив Zip: http://opencorpora.org/files/export/dict/dict.opcorpora.xml.zip

// Dictionary реализует структуру словаря.
type Dictionary struct {
	// Версия словаря
	VersionAttr float64 `xml:"version,attr"`
	// Номер ревизии на момент импорта
	RevisionAttr int `xml:"revision,attr"`
	// Граммемы (грамматические категории)
	Grammemes *Grammemes `xml:"grammemes"`
	// Ограничения и требования на применение категорий
	Restrictions *Restrictions `xml:"restrictions"`
	// Список лемм
	Lemmata *Lemmata `xml:"lemmata"`
	// Типы связей между леммами
	Linktypes *LinkTypes `xml:"link_types"`
	// Связи между леммами
	Links *Links `xml:"links"`
}
