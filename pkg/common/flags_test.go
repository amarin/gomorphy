package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/amarin/gomorphy/pkg/common"
)

func TestFindOutFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"absent", []string{"compile"}, ""},
		{"space form", []string{"compile", "-o", "out.dat"}, "out.dat"},
		{"equals form", []string{"compile", "-o=out.dat"}, "out.dat"},
		{"dangling flag", []string{"compile", "-o"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, common.FindOutFlag(tc.args))
		})
	}
}

func TestStripOutFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"absent", []string{"compile"}, []string{"compile"}},
		{"space form", []string{"compile", "-o", "out.dat"}, []string{"compile"}},
		{"equals form", []string{"compile", "-o=out.dat"}, []string{"compile"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, common.StripOutFlag(tc.args))
		})
	}
}
