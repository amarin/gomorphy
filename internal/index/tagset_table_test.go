package index_test

import (
	"bytes"
	"testing"

	"github.com/amarin/binutils"
	"github.com/stretchr/testify/require"

	. "github.com/amarin/gomorphy/internal/index"
)

func TestTagSetTable_Index(t *testing.T) {
	table := make(TagSetTable, 0)
	require.Equal(t, TagSetSubID(0), table.Index(TagSet{0}))
	require.Equal(t, 1, table.Len())
	require.Equal(t, TagSetSubID(1), table.Index(TagSet{1}))
	require.Equal(t, 2, table.Len())
	require.Equal(t, TagSetSubID(2), table.Index(TagSet{2}))
	require.Equal(t, 3, table.Len())
	require.Equal(t, TagSetSubID(3), table.Index(TagSet{3}))
	require.Equal(t, 4, table.Len())
}

func TestTagSetTable_Get(t *testing.T) {
	tests := []struct {
		name        string
		tagSetTable TagSetTable
		tagSetSubID TagSetSubID
		wantTagSet  TagSet
		wantFound   bool
	}{
		{"get_existed_0", TagSetTable{{0}, {1}, {2}}, 0, TagSet{0}, true},
		{"get_existed_1", TagSetTable{{0, 1}, {2, 3}}, 1, TagSet{2, 3}, true},
		{"not_existed_2", TagSetTable{{0, 1}, {2, 3}}, 2, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTargetSet, gotFound := tt.tagSetTable.Get(tt.tagSetSubID)
			require.Equal(t, tt.wantFound, gotFound)
			if !gotFound {
				return
			}
			require.NotNil(t, gotTargetSet)
			require.True(t, gotTargetSet.EqualTo(tt.wantTagSet))
		})
	}
}

func TestTagSetTable_Find(t *testing.T) {
	tests := []struct {
		name        string
		tagSetTable TagSetTable
		tagSetSubID TagSetSubID
		wantTagSet  TagSet
		wantFound   bool
	}{
		{"get_existed_0", TagSetTable{{0}, {1}, {2}}, 0, TagSet{0}, true},
		{"get_existed_1", TagSetTable{{0, 1}, {2, 3}}, 1, TagSet{2, 3}, true},
		{"not_existed_2", TagSetTable{{0, 1}, {2, 3}}, 2, TagSet{1, 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTagSetSubID, gotFound := tt.tagSetTable.Find(tt.wantTagSet)
			require.Equal(t, tt.wantFound, gotFound)
			if !gotFound {
				return
			}
			require.Equal(t, tt.tagSetSubID, gotTagSetSubID)
		})
	}
}

func TestTagSetTable_ReadWrite(t *testing.T) {
	t.Run("nil_writer_cause_error", func(t *testing.T) {
		require.Error(t, (&TagSetTable{}).BinaryWriteTo(nil))
	})
	t.Run("nil_reader_cause_error", func(t *testing.T) {
		require.Error(t, (&TagSetTable{}).BinaryReadFrom(nil))
	})
	tests := []struct {
		name        string
		tagSetTable TagSetTable
		wantErr     bool
	}{
		{"read_write_empty", TagSetTable{}, false},
		{"read_write_3_elems_of_len_1", TagSetTable{{0}, {1}, {2}}, false},
		{"read_write_4_elems_of_len_2", TagSetTable{{0, 1}, {2, 3}, {4, 5}, {6, 7}}, false},
		{"read_write_3_elems_of_len_3", TagSetTable{{0, 1, 2}, {3, 4, 5}, {6, 7, 8}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := new(bytes.Buffer)
			writer := binutils.NewBinaryWriter(buffer)
			reader := binutils.NewBinaryReader(buffer)
			err := tt.tagSetTable.BinaryWriteTo(writer)
			require.Equal(t, tt.wantErr, err != nil)
			if err != nil {
				return
			}

			loaded := make(TagSetTable, 0)
			require.NoError(t, loaded.BinaryReadFrom(reader))
			require.Equal(t, tt.tagSetTable.Len(), loaded.Len())
			for idx := 0; idx < loaded.Len(); idx++ {
				require.True(t, loaded[idx].EqualTo(tt.tagSetTable[idx]))
			}
		})
	}
}
