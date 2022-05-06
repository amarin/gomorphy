package opencorpora

import (
	"fmt"
	"strings"

	"github.com/amarin/gomorphy/pkg/dag"
)

// Lemma provides grouping of word base form and its variations tagged by grammar categories.
type Lemma struct {
	// Уникальный идентификатор слова
	IdAttr int `xml:"id,attr"`
	// Номер ревизии
	RevAttr int `xml:"rev,attr"`
	// Базовая форма
	L WordForm `xml:"l"`
	// Другие формы этой-же части речи
	F WordFormList `xml:"f"`
}

func newLemma() *Lemma {
	return &Lemma{
		IdAttr:  0,
		RevAttr: 0,
		L:       *newWordForm(),
		F:       make(WordFormList, 0),
	}
}

// Строковое представление леммы. Реализует интерфейс Stringer()
func (lemma Lemma) String() string {
	str := make([]string, 0)
	for _, item := range lemma.F {
		str = append(str, item.String())
	}
	return "Lemma(" + fmt.Sprintf("%v", lemma.IdAttr) + "," + fmt.Sprintf("%v", lemma.RevAttr) + "," + lemma.L.String() + ", " + strings.Join(str, ",")
}

func (lemma Lemma) pushToIndex(idx dag.Index) error {
	return fmt.Errorf("not implemented")
}
