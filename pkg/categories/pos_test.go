package categories_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/amarin/gomorphy/pkg/categories"
)

func TestPOS_ByString(t *testing.T) {
	type testStruct struct {
		name       string
		testString string
		want       *POS
	}
	var tests = make([]testStruct, 0)

	for _, pos := range KnownPoses {
		pos := pos
		tests = append(
			tests,
			testStruct{
				fmt.Sprintf("ok_%v", strings.ToUpper(string(pos))),
				strings.ToUpper(string(pos)),
				&pos,
			},
		)
		tests = append(
			tests,
			testStruct{
				fmt.Sprintf("ok_%v", strings.ToLower(string(pos))),
				strings.ToLower(string(pos)),
				&pos,
			},
		)
	}

	for _, tt := range tests {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			if got := KnownPoses.ByString(tt.testString); tt.want != nil && got == nil {
				t.Errorf("MatchPartOfString() = nil, want %v", tt.want)
			} else if tt.want != nil && got != nil && *tt.want != *got {
				t.Errorf("MatchPartOfString() = %v, want %v", *got, *tt.want)
			}
		})
	}
}
