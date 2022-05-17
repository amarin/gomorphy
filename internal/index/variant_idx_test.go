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
	exampleIndex := index.VariantsIndex{
		index.VariantsTable{{10}, {11}, {12}},
		index.VariantsTable{{20, 21}, {22, 23}},
		index.VariantsTable{{30, 31, 32}, {33, 34, 35}, {36, 37, 38}},
	}
	tests := []struct {
		name       string
		storageIdx index.VariantID
		want       index.TagSetIDCollection
	}{
		{"get_0x00000", 0, index.TagSetIDCollection{}},
		{"get_0x10000", 0x10000, index.TagSetIDCollection{10}},
		{"get_0x20001", 0x20001, index.TagSetIDCollection{22, 23}},
		{"get_0x30002", 0x30002, index.TagSetIDCollection{36, 37, 38}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exampleIndex.Get(tt.storageIdx)
			require.Equalf(t, tt.want, got,
				"idx: %v\nid: %v,\ntable: %v\nitem: %v",
				exampleIndex, tt.storageIdx, tt.storageIdx.TableNum(), tt.storageIdx.CollectionTableID())
		})
	}
}
