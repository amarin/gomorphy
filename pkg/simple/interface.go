package simple

import "github.com/amarin/gomorphy/pkg/tag"

type Interface interface {
	WordsCount() int
	NodesCount() int
	Optimize()
	Add(word string, tags ...tag.Name) (int, error)
	RegisterTag(tag.Name) (int, error)
	TagsCount() int
}
