package index_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/index"
)

func TestVariantsTable_Index(t *testing.T) {
	tests := []struct {
		name       string
		existed    index.VariantsTable
		item       index.TagSetIDCollection
		expectedID index.VariantSubID
	}{
		{"push_to_empty_1st_level", make(index.VariantsTable, 0), index.TagSetIDCollection{10}, 0},
		{"push_to_empty_2nd_level", make(index.VariantsTable, 0), index.TagSetIDCollection{10, 11}, 0},
		{"push_2nd", index.VariantsTable{{10}}, index.TagSetIDCollection{11}, 1},
		{"push_3rd", index.VariantsTable{{10}, {11}}, index.TagSetIDCollection{12}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexedNo := tt.existed.Index(tt.item)
			require.Equal(t, tt.expectedID, indexedNo)
			require.True(t, tt.existed.Get(indexedNo).EqualTo(tt.item))
		})
	}
}
