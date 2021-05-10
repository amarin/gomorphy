package grammemes

import (
	"github.com/amarin/gomorphy/internal/grammeme"
)

// GrammemeIndexer задаёт интерфейс для типов, возвращающих индекс граммем
type GrammemeIndexer interface {
	GrammemeIndex() *grammeme.Index
}
