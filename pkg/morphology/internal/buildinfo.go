package internal

import (
	"encoding/json"
	"time"
)

// BuildInfo — диагностические метаданные словаря: секция "info" в файле
// GMOR. В отличие от meta (язык + CharPolicy — обязательны для Parse), ни
// одно поле BuildInfo не требуется для работы словаря: все поля
// опциональны (нулевое значение = «не указано»), секция целиком может
// отсутствовать (файлы, собранные до появления этого поля, или словари,
// собранные вручную через Builder API без вызова SaveTo).
type BuildInfo struct {
	// BuiltAt и LibraryVersion проставляет сама SaveTo при каждом
	// сохранении — значения, заданные импортёром заранее, перезаписываются.
	BuiltAt        time.Time `json:"built_at,omitempty"`
	LibraryVersion string    `json:"library_version,omitempty"`

	// Source — источник данных: "opencorpora", "pymorphy2", "unimorph",
	// "tsv", ... Заполняется импортёром при построении Dictionary.
	Source string `json:"source,omitempty"`

	// SourceVersion — версия/ревизия исходных данных (например, для
	// OpenCorpora dict.xml — атрибуты version/revision корневого тега).
	// Пока не заполняется ни одним импортёром: xmlscan не отдаёт атрибуты
	// корневого тега наружу — см. docs/todo.md.
	SourceVersion string `json:"source_version,omitempty"`

	// Author, Description — свободные поля, актуальны в первую очередь
	// для тематических словарей (см. docs/todo.md, Этап 19).
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`

	// SourceURL — где вручную (или агентом) скачать свежую версию словаря.
	// Задел под будущий стандарт списка версий — см. docs/todo.md.
	SourceURL string `json:"source_url,omitempty"`
}

// EncodeBuildInfo сериализует BuildInfo в JSON для секции "info".
func EncodeBuildInfo(info *BuildInfo) []byte {
	if info == nil {
		info = &BuildInfo{}
	}
	data, _ := json.Marshal(info)
	return data
}

// DecodeBuildInfo читает BuildInfo, записанный EncodeBuildInfo.
func DecodeBuildInfo(data []byte) (*BuildInfo, error) {
	var info BuildInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, wrap(ErrMalformedFile, "info")
	}
	return &info, nil
}
