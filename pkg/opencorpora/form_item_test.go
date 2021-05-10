package opencorpora_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/grammeme"
	"github.com/amarin/gomorphy/pkg/categories"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestF_getTagsFromSet(t *testing.T) {
	type testStruct struct {
		name          string
		grammarList   string
		searchList    []string
		expectedCount int
	}
	var tests []testStruct
	tests = append(
		tests,
		testStruct{
			"ok_find_nothing",
			"one,two",
			[]string{"three", "four"},
			0,
		},
	)
	for _, c := range categories.CaseStrings {
		tests = append(
			tests,
			testStruct{
				fmt.Sprintf("ok_find_%v", c),
				fmt.Sprintf("%v,unknown", c),
				categories.CaseStrings,
				1,
			},
		)
	}
	for _, c := range categories.NumberStrings {
		tests = append(
			tests,
			testStruct{
				fmt.Sprintf("ok_find_%v", c),
				fmt.Sprintf("%v,unknown", c),
				categories.NumberStrings,
				1,
			},
		)
	}
	for _, tt := range tests {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			x := &WordForm{Form: "", G: make([]*Category, 0)}
			for _, g := range strings.Split(tt.grammarList, ",") {
				x.G = append(x.G, &Category{VAttr: grammeme.Name(g)})
			}
			if got := x.GetTagsFromSet(tt.searchList); len(got) != tt.expectedCount {
				t.Errorf(
					"%v.getTagsFromSet(%v) = %v (%v), want %v",
					x, tt.searchList, got, len(got), tt.expectedCount,
				)
			}
		})
	}
}
