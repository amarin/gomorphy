package index_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/index"
)

func TestTableIDCollectionIndex_Index(t *testing.T) {
	tests := []struct {
		name       string
		idx        index.VariantsIndex
		item       index.TagSetIDCollection
		expectedID index.VariantID
	}{
		{"pass_to_empty_1st_level", index.VariantsIndex{}, index.TagSetIDCollection{10}, 0x10000},
		{"pass_to_empty_2nd_level", index.VariantsIndex{}, index.TagSetIDCollection{10, 11}, 0x20000},
		{"pass_zero", index.VariantsIndex{}, index.TagSetIDCollection{0}, 0x10000},
		{"pass_empty", index.VariantsIndex{}, index.TagSetIDCollection{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIDx := tt.idx.Index(tt.item)
			require.Equal(t, tt.expectedID, gotIDx)
			require.Truef(t, tt.idx.Get(tt.expectedID).EqualTo(tt.item),
				"expected: %v,\ngot:     %v\nID:       %v\nIDX:      %v",
				tt.item, tt.idx.Get(tt.expectedID), gotIDx, tt.idx)
		})
	}
}

func TestVariantsIndex_Get(t *testing.T) {
	tests := []struct {
		name        string
		tagSetIndex index.VariantsIndex
		storageIdx  index.VariantID
		want        index.TagSetIDCollection
	}{
		{
			"get_0",
			index.VariantsIndex{
				index.VariantsTable{
					{10},
				},
			},
			0,
			index.TagSetIDCollection{},
		},
		{
			"get_0x10000",
			index.VariantsIndex{
				index.VariantsTable{
					{10},
				},
			},
			0,
			index.TagSetIDCollection{10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tagSetIndex.Get(tt.storageIdx)
			require.Equal(t, tt.want, got)
		})
	}
}
