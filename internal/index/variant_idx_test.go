package index_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"
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

func TestVariantsIndex_BinaryWriteTo_BinaryReadFrom(t *testing.T) {
	tests := []struct {
		name       string
		addToIndex index.TagSetIDCollection
	}{
		{"index_1st_level", index.TagSetIDCollection{11}},
		{"index_2nd_level", index.TagSetIDCollection{11, 13}},
		{"index_3rd_level", index.TagSetIDCollection{11, 13, 17}},
		{"index_4rd_level", index.TagSetIDCollection{11, 13, 17, 19}},
		{"index_5th_level", index.TagSetIDCollection{11, 13, 17, 19, 23}},
		{"index_6th_level", index.TagSetIDCollection{11, 13, 17, 19, 23, 29}},
		{"index_7th_level", index.TagSetIDCollection{11, 13, 17, 19, 23, 29, 31}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				originalVariantID index.VariantID
				loadedVariantID   index.VariantID
				found             bool
			)
			idxOriginal := make(index.VariantsIndex, 0)
			originalVariantID = idxOriginal.Index(tt.addToIndex)
			loadedVariantID, found = idxOriginal.Find(tt.addToIndex)
			require.Truef(t, found,
				"variant not found after index:\nLooking for variant: %v (0x%x)\nindex: %v",
				originalVariantID, originalVariantID, idxOriginal.Tables(),
			)
			require.Equalf(t, originalVariantID, loadedVariantID,
				"variant get after index:\nLooking for variant: %v\nindex: %v\n",
				originalVariantID, loadedVariantID, idxOriginal.Tables(),
			)

			buffer := new(bytes.Buffer)

			require.NoError(t, idxOriginal.BinaryWriteTo(binutils.NewBinaryWriter(buffer)))
			writtenBytes := buffer.Bytes()
			idxLoaded := make(index.VariantsIndex, 0)
			require.NoError(t, idxLoaded.BinaryReadFrom(binutils.NewBinaryReader(buffer)))

			loadedVariantID, found = idxLoaded.Find(tt.addToIndex)
			require.Truef(t, found,
				"variant not found after save-load:\nLooking for variant: %v (%0X)\n index binary: %v\nindex: %v\nloaded: %v",
				originalVariantID, originalVariantID, hex.EncodeToString(writtenBytes), idxOriginal.Tables(), idxLoaded.Tables(),
			)
			require.Equalf(t, originalVariantID, loadedVariantID,
				"variant changed after save-load:\nOriginal for variant: %v\nReceived variant: %v\nindex binary: %v\nindex: %v\nloaded: %v",
				originalVariantID, loadedVariantID, hex.EncodeToString(writtenBytes), idxOriginal.Tables(), idxLoaded.Tables(),
			)
		})
	}
}
