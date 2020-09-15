package grammemes

// GrammemeIndexer задаёт интерфейс для типов, возвращающих индекс граммем
type GrammemeIndexer interface {
	GrammemeIndex() *Index
}
