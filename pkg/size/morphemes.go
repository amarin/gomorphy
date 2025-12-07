package size

type (
	// Morphemes задаёт максимальную ёмкость DAG словаря с учётом всех возможных словоформ
	Morphemes interface {
		~uint16 | ~uint32
	}
)
