package grammemes

import (
	grammeme2 "github.com/amarin/gomorphy/pkg/dag"
)

// GrammemeIndexer задаёт интерфейс для типов, возвращающих индекс граммем
type GrammemeIndexer interface {
	GrammemeIndex() *grammeme2.Idx
}
