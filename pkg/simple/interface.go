package simple

import "github.com/amarin/gomorphy/pkg/tag"

type Interface interface {
	WordsCount() int
	NodesCount() int
	Optimize()
	Add(word string, tags ...any) (int, error)
	RegisterTag(tag.Tag) (int, error)
}
