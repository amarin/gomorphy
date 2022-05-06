package opencorpora_test

import (
	"testing"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestF_getTagsFromSet(t *testing.T) {
	// for _, tt := range []struct {
	// 	name          string
	// 	grammarList   string
	// 	searchList    []string
	// 	expectedCount int
	// }{
	// 	{
	// 		"ok_find_nothing",
	// 		"one,two",
	// 		[]string{"three", "four"},
	// 		0,
	// 	},
	// } {
	// 	tt := tt // pin tt
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		tt := tt // pin tt
	// 		x := &WordForm{Form: "", G: make([]*Category, 0)}
	// 		for _, g := range strings.Split(tt.grammarList, ",") {
	// 			x.G = append(x.G, &Category{VAttr: grammeme2.TagName(g)})
	// 		}
	// 		if got := x.GetTagsFromSet(tt.searchList); len(got) != tt.expectedCount {
	// 			t.Errorf(
	// 				"%v.getTagsFromSet(%v) = %v (%v), want %v",
	// 				x, tt.searchList, got, len(got), tt.expectedCount,
	// 			)
	// 		}
	// 	})
	// }
}
