package tag

import "github.com/amarin/gomorphy/pkg/indexer"

type Indexer = indexer.Indexer[uint8, Tag]

func NewIndexer() *Indexer {
	return indexer.New[uint8, Tag]()
}
