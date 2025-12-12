package words

import (
	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/node"
)

type dagSize interface {
	uint16 | uint32
}

type wordsSize interface {
	uint8 | uint16 | uint32
}

func NewIndex[D dagSize, W wordsSize]() indexer.IndexOf[W, node.ID[D], *node.ID[D]] {
	return indexer.IndexOf[W, node.ID[D], *node.ID[D]]{}
}
