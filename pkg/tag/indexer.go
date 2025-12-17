package tag

import (
	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/size"
)

func NewIndexer[S size.TagsSize]() *indexer.IndexOf[S, Name, *Name] {
	return indexer.New[S, Name]("tags")
}
