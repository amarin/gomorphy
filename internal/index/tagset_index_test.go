package index_test

import (
	"bytes"
	"testing"

	"github.com/amarin/binutils"
	"github.com/stretchr/testify/require"

	. "github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
)

func TestTagSetIndex_BinaryWriteTo(t *testing.T) {
	buffer := new(bytes.Buffer)
	writer := binutils.NewBinaryWriter(buffer)
	reader := binutils.NewBinaryReader(buffer)
	tests := []struct {
		name       string
		addTagSets []TagSet
		expectLen  int
		expectHex  string
	}{
		{"empty_tagSet_index", make([]TagSet, 0), 0,
			"00000000"},
		{"set_of_1", []TagSet{[]dag.TagID{1}}, 1,
			"0000000100010101"},
		{"set_of_2", []TagSet{[]dag.TagID{2, 3}}, 2,
			"0000000200000001020203"},
		{"set_of_1_and_2", []TagSet{[]dag.TagID{1}, []dag.TagID{2, 3}}, 2,
			"00000002000101010001020203"},
		{"set_of_5", []TagSet{[]dag.TagID{5, 6, 7, 8, 9}}, 5,
			"0000000500000000000000000001050506070809"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := make(TagSetIndex, 0)
			require.Equal(t, 0, idx.Len())
			buffer.Reset()
			for _, ts := range tt.addTagSets {
				idx.Index(ts)
			}
			require.Equal(t, tt.expectLen, idx.Len())
			require.NoError(t, idx.BinaryWriteTo(writer))
			hex, err := reader.ReadHex(buffer.Len())
			require.NoError(t, err)
			require.Equal(t, tt.expectHex, hex)
		})
	}
}
