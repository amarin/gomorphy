package index_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"
	"github.com/stretchr/testify/require"

	. "github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
)

func TestTagSetIndex_Find(t *testing.T) {
	tests := []struct {
		name           string
		tagSetIndex    TagSetIndex
		search         TagSet
		wantStorageIdx TagSetID
		wantFound      bool
	}{
		{"missed_in_empty_idx",
			TagSetIndex{},
			TagSet{1},
			0,
			false},
		{"found_at_1st_level",
			TagSetIndex{{{1}}},
			TagSet{1},
			0,
			true},
		{"found_at_2nd_level_first",
			TagSetIndex{{{1}}, {{2, 3}}},
			TagSet{2, 3},
			0x10000,
			true},
		{"found_at_3rd_level_not_first",
			TagSetIndex{{{1}}, {{2, 3}}, {{4, 5, 6}, {7, 8, 9}, {11, 12, 13}}},
			TagSet{11, 12, 13},
			0x20002,
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStorageIdx, gotFound := tt.tagSetIndex.Find(tt.search)
			require.Equal(t, tt.wantFound, gotFound)
			if !gotFound {
				return
			}
			require.Equal(t, tt.wantStorageIdx, gotStorageIdx)
		})
	}
}

func TestTagSetIndex_BinaryReadFrom(t *testing.T) {
	t.Run("nil_reader_raises", func(t *testing.T) {
		idx := make(TagSetIndex, 0)
		err := idx.BinaryReadFrom(nil)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNilReader)
	})

	tests := []struct {
		name       string
		addTagSets []TagSet
		expectSize int
		expectHex  string
	}{
		{"empty_tagSet_index", make([]TagSet, 0), 0,
			"00000000"},
		{"set_of_1", []TagSet{[]dag.TagID{1}}, 1,
			"0000000100010101"},
		{"set_of_2", []TagSet{[]dag.TagID{2, 3}}, 1,
			"0000000200000001020203"},
		{"set_of_1_and_2", []TagSet{[]dag.TagID{1}, []dag.TagID{2, 3}}, 2,
			"00000002000101010001020203"},
		{"set_of_5", []TagSet{[]dag.TagID{5, 6, 7, 8, 9}}, 1,
			"0000000500000000000000000001050506070809"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := hex.DecodeString(tt.expectHex)
			require.NoError(t, err)
			buffer := bytes.NewBuffer(data)
			reader := binutils.NewBinaryReader(buffer)
			idx := make(TagSetIndex, 0)
			require.Equal(t, 0, idx.Size())
			require.NoError(t, idx.BinaryReadFrom(reader))
			require.Equal(t, tt.expectSize, idx.Size())
			for _, ts := range tt.addTagSets {
				id, found := idx.Find(ts)
				require.True(
					t, found,
					"Missed: %v\nID    : %v\nData  : %v\nIdx   : %v", ts, id, tt.expectHex, idx,
				)
				loadedTS, loaded := idx.Get(id)
				require.True(t, loaded)
				require.Truef(
					t, loadedTS.EqualTo(ts),
					"ID    : %v\nSaved : %v\nLoaded: %v\nIdx   : %v",
					id, ts, loadedTS, idx,
				)
			}
		})
	}
}

func TestTagSetIndex_BinaryWriteTo(t *testing.T) {
	t.Run("nil_writer_raises", func(t *testing.T) {
		idx := TagSetIndex{[]TagSet{[]dag.TagID{1}}}
		err := idx.BinaryWriteTo(nil)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNilWriter)
	})

	buffer := new(bytes.Buffer)
	writer := binutils.NewBinaryWriter(buffer)
	reader := binutils.NewBinaryReader(buffer)
	tests := []struct {
		name       string
		addTagSets []TagSet
		expectSize int
		expectHex  string
	}{
		{"empty_tagSet_index", make([]TagSet, 0), 0,
			"00000000"},
		{"set_of_1", []TagSet{[]dag.TagID{1}}, 1,
			"0000000100010101"},
		{"set_of_2", []TagSet{[]dag.TagID{2, 3}}, 1,
			"0000000200000001020203"},
		{"set_of_1_and_2", []TagSet{[]dag.TagID{1}, []dag.TagID{2, 3}}, 2,
			"00000002000101010001020203"},
		{"set_of_5", []TagSet{[]dag.TagID{5, 6, 7, 8, 9}}, 1,
			"0000000500000000000000000001050506070809"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := make(TagSetIndex, 0)
			require.Equal(t, 0, idx.Size())
			buffer.Reset()
			for _, ts := range tt.addTagSets {
				idx.Index(ts)
			}
			require.Equal(t, tt.expectSize, idx.Size())
			require.NoError(t, idx.BinaryWriteTo(writer))
			hex, err := reader.ReadHex(buffer.Len())
			require.NoError(t, err)
			require.Equal(t, tt.expectHex, hex)
		})
	}
}

func TestTagSetIndex_Size(t *testing.T) {
	tests := []struct {
		name        string
		want        int
		addElements []TagSet
	}{
		{"empty_index_zero_len", 0, []TagSet{}},
		{"len_1_with_single_elem", 1, []TagSet{{0}}},
		{"len_2_with_couple_different_sizes", 2, []TagSet{{0}, {1, 2}}},
		{"len_3_with_same_sizes", 3, []TagSet{{0}, {1}, {2}}},
		{"len_4_with_gaps", 4, []TagSet{{0}, {1, 2}, {3, 4, 5, 6}, {7, 8, 9, 10, 11, 12, 13}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := make(TagSetIndex, 0)
			for _, ts := range tt.addElements {
				_ = idx.Index(ts)
			}
			require.Equal(t, tt.want, idx.Size())
		})
	}
}
