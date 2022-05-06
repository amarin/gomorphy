package opencorpora

import (
	"github.com/amarin/gomorphy/pkg/dag"
)

// CategoryList provides a category list.
type CategoryList []*Category

// GrammemeNames возвращает список имён граммем, заданных в списке категорий.
func (c CategoryList) GrammemeNames() []dag.TagName {
	res := make([]dag.TagName, 0)
	for _, item := range c {
		res = append(res, item.VAttr)
	}

	return res
}
