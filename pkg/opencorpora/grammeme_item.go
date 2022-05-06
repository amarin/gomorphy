package opencorpora

import (
	"github.com/amarin/gomorphy/pkg/dag"
)

// Grammeme provides grammar category definition structure as provided by OpenCorpora
type Grammeme struct {
	// ParentAttr provides parent category TagName.
	ParentAttr dag.TagName `xml:"parent,attr"`
	// Наименование категории. Аббревиатура от латинского или англоязычного наименования. Всегда 4 символа
	Name dag.TagName `xml:"name"`
	// Кириллическая аббревиатура наименования.
	Alias string `xml:"alias"`
	// Полное наименование на русском языке
	Description string `xml:"description"`
}

func (g Grammeme) String() string {
	return "Tag(" + string(g.Name) + "/" + string(g.ParentAttr) + "," + g.Alias + "," + g.Description + ")"
}

// Tag provides grammar tag from OpenCorpora grammeme definition.
func (g Grammeme) Tag() dag.Tag {
	return *dag.NewTag(g.ParentAttr, g.Name)
}
