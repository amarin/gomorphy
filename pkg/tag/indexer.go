package tag

import "github.com/amarin/gomorphy/pkg/indexer"

type Indexer struct {
	*indexer.IndexOf[uint8, Tag, *Tag]
}

func NewIndexer() *Indexer {
	return &Indexer{
		IndexOf: indexer.New[uint8, Tag]("tags"),
	}
}
