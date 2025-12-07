package opencorpora

import (
	"github.com/amarin/gomorphy/pkg/tag"
)

// Category represents OpenCorpora grammar category as a set of grammar Name's
type Category struct {
	VAttr tag.Name `xml:"v,attr"`
}

// String returns string representation of category. Implements fmt.Stringer.
func (x Category) String() string {
	return x.VAttr.String()
}
