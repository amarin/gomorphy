package size

type (
	// Morphemes задаёт максимальную ёмкость DAG словаря с учётом всех возможных словоформ
	Morphemes interface {
		uint16 | uint32
	}
)

// TagsSize задаёт размерность справочника тегов
type TagsSize interface {
	uint8 | uint16
}

// TagSetSize задаёт размерность справочника наборов тегов
type TagSetSize interface {
	uint16
}
