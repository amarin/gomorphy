package opencorpora

import (
	"strings"

	"github.com/amarin/gomorphy/pkg/dag"
)

// WordForm provides word form and categories list.
type WordForm struct {
	Form string       `xml:"t,attr"`
	G    CategoryList `xml:"grammemes"`
}

func newWordForm() *WordForm {
	return &WordForm{
		Form: "",
		G:    make(CategoryList, 0),
	}
}

// String returns string representation of WordForm. Implements fmt.Stringer.
func (x WordForm) String() string {
	str := make([]string, 0)
	for _, item := range x.G {
		str = append(str, item.String())
	}
	return "WordForm(" + x.Form + "," + strings.Join(str, ",") + ")"
}

// GetTagsFromSet takes TagName's list from word form.
func (x WordForm) GetTagsFromSet() []dag.TagName {
	var resultTags []dag.TagName
	for _, g := range x.G {
		if g != nil {
		}
		resultTags = append(resultTags, g.VAttr)
	}
	return resultTags
}
