package index_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/index"
)

func TestTableIDCollection_Less(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		t    index.TagSetIDCollection
		args args
		want bool
	}{
		{"i_is_less_than_j", index.TagSetIDCollection{10, 12}, args{0, 1}, true},
		{"i_not_less_j", index.TagSetIDCollection{10, 10}, args{0, 1}, false},
		{"i_is_greater_than_j", index.TagSetIDCollection{12, 10}, args{0, 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.t.Less(tt.args.i, tt.args.j))
		})
	}
}

func TestTableIDCollection_Swap(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		t    index.TagSetIDCollection
		args args
	}{
		{"swap_0_1", index.TagSetIDCollection{10, 12}, args{0, 1}},
		{"swap_1_2", index.TagSetIDCollection{10, 12, 13}, args{1, 2}},
		{"swap_0_2", index.TagSetIDCollection{10, 12, 13}, args{0, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wasI := tt.t[tt.args.i]
			wasJ := tt.t[tt.args.j]
			tt.t.Swap(tt.args.i, tt.args.j)
			require.Equal(t, wasI, tt.t[tt.args.j])
			require.Equal(t, wasJ, tt.t[tt.args.i])
		})
	}
}

func TestTagSetIDCollection_EqualTo(t *testing.T) {
	tests := []struct {
		name    string
		t       index.TagSetIDCollection
		another index.TagSetIDCollection
		want    bool
	}{
		{"empties_are_equal",
			index.TagSetIDCollection{},
			index.TagSetIDCollection{},
			true},
		{"long_equal",
			index.TagSetIDCollection{1, 3, 5, 7, 11, 13, 17, 23, 29},
			index.TagSetIDCollection{1, 3, 5, 7, 11, 13, 17, 23, 29},
			true},
		{"looking_similar_but_len_differs",
			index.TagSetIDCollection{1, 3, 5, 7, 11, 13, 17, 23, 29},
			index.TagSetIDCollection{1, 3, 5, 7, 11, 13, 17, 23},
			false},
		{"same_elements_but_unsorted_are_not_equal",
			index.TagSetIDCollection{1, 3, 5, 7, 11, 13, 17, 23, 29},
			index.TagSetIDCollection{1, 3, 5, 7, 11, 13, 17, 29, 23},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.t.EqualTo(tt.another))
		})
	}
}

func TestTagSetIDCollection_Has(t *testing.T) {
	tests := []struct {
		name           string
		t              index.TagSetIDCollection
		searchTagSetID index.TagSetID
		want           bool
	}{
		{"empty_contains_nothing", index.TagSetIDCollection{}, 0, false},
		{"non_existed_non_found", index.TagSetIDCollection{1, 3, 5, 7}, 2, false},
		{"existed_is_found", index.TagSetIDCollection{1, 3, 5, 7}, 7, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.Has(tt.searchTagSetID); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagSetIDCollection_Add(t *testing.T) {
	tests := []struct {
		name               string
		t                  index.TagSetIDCollection
		additionalTagSetID index.TagSetID
		want               index.TagSetIDCollection
	}{
		{
			"add_some_to_empty",
			index.TagSetIDCollection{},
			1,
			index.TagSetIDCollection{1},
		},
		{
			"add_existed_tag_set_id_produces_a_copy",
			index.TagSetIDCollection{1, 3, 7},
			1,
			index.TagSetIDCollection{1, 3, 7},
		},
		{
			"adding_produces_sorted_result",
			index.TagSetIDCollection{11, 7, 5, 3, 2},
			1,
			index.TagSetIDCollection{1, 2, 3, 5, 7, 11},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.t.Add(tt.additionalTagSetID)
			require.Equal(t, tt.want, res)        // check result is expected by reflected values
			require.True(t, tt.want.EqualTo(res)) // check result is expected by EqualTo
			require.NotSame(t, tt.want, res)      // check result is different even if a copy

			// check if result is sorted
			possiblyUnsorted := make(index.TagSetIDCollection, len(res))
			copy(possiblyUnsorted, res)
			sort.Sort(possiblyUnsorted)
			require.Equal(t, possiblyUnsorted, res) // ensure possibly unsorted result and its sorted copy are equal
		})
	}
}
