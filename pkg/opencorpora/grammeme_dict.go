package opencorpora

import (
	"strings"
)

// Grammemes provides a list of known grammar categories&
type Grammemes struct {
	Grammeme []*Grammeme `xml:"grammeme"`
}

func (x Grammemes) String() string {
	var stringItems []string
	for _, g := range x.Grammeme {
		stringItems = append(stringItems, g.String())
	}
	return "Grammemes(" + strings.Join(stringItems, ",") + ")"
}
