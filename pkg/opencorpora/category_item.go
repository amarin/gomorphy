package opencorpora

import (
	"github.com/amarin/gomorphy/pkg/dag"
)

// Category represents OpenCorpora grammar category as a set of grammar TagName's
type Category struct {
	VAttr dag.TagName `xml:"v,attr"`
}

// String returns string representation of category. Implements fmt.Stringer.
func (x Category) String() string {
	return x.VAttr.String()
}
