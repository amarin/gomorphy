package dag

import (
	"strings"

	"github.com/amarin/gomorphy/pkg/tag"
)

// TagSet represents a list of tags attached to words.
type TagSet []tag.Tag

// String returns string representation of TagSet. Implements fmt.Stringer.
func (tagSet TagSet) String() string {
	tagStrings := make([]string, len(tagSet))
	for idx, tag := range tagSet {
		tagStrings[idx] = string(tag.Name)
	}

	return strings.Join(tagStrings, ",")
}
