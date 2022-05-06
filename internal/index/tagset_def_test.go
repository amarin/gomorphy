package index

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagSet_EqualTo(t *testing.T) {
	tests := []struct {
		name    string
		tagSet  TagSet
		another TagSet
		want    bool
	}{
		{"empty_sets_are_equal", make(TagSet, 0), make(TagSet, 0), true},
		{"single_equal_elements", TagSet{10}, TagSet{10}, true},
		{"some_equal_elements", TagSet{10, 12}, TagSet{10, 12}, true},
		{"non_sorted_are_not_equal", TagSet{12, 10}, TagSet{10, 12}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tagSet.EqualTo(tt.another))
		})
	}
}
