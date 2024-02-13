package storage_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/pkg/storage"
	"github.com/stretchr/testify/require"
)

func TestSet16_Swap(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set16
		i            int
		j            int
	}{
		{"swap_0_1", storage.Set16{100, 200, 300, 400}, 0, 1},
		{"swap_1_2", storage.Set16{1, 2, 3, 4}, 1, 2},
		{"swap_2_3", storage.Set16{5, 6, 7, 8}, 2, 3},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			wantI := tt.grammemesSet[tt.i]
			wantJ := tt.grammemesSet[tt.j]
			tt.grammemesSet.Swap(tt.i, tt.j)
			require.Equal(t, tt.grammemesSet[tt.i], wantJ)
			require.Equal(t, tt.grammemesSet[tt.j], wantI)
		})
	}
}

func TestSet16_Less(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set16
		i            int
		j            int
		wantLess     bool
	}{
		{"less_0_0_false", storage.Set16{6, 7, 8, 9}, 0, 0, false},
		{"less_0_1_true", storage.Set16{6, 7, 8, 9}, 0, 1, true},
		{"less_0_2_true", storage.Set16{6, 7, 8, 9}, 0, 2, true},
		{"less_0_3_true", storage.Set16{6, 7, 8, 9}, 0, 3, true},
		{"less_1_0_false", storage.Set16{1, 2, 3, 4}, 1, 0, false},
		{"less_1_1_false", storage.Set16{6, 7, 8, 9}, 1, 1, false},
		{"less_1_2_true", storage.Set16{6, 7, 8, 9}, 1, 2, true},
		{"less_1_3_true", storage.Set16{6, 7, 8, 9}, 1, 3, true},
		{"less_2_0_false", storage.Set16{5, 6, 7, 8}, 2, 0, false},
		{"less_2_1_false", storage.Set16{5, 6, 7, 8}, 2, 1, false},
		{"less_2_2_false", storage.Set16{5, 6, 7, 8}, 2, 2, false},
		{"less_2_3_true", storage.Set16{5, 6, 7, 8}, 2, 3, true},
		{"less_3_0_false", storage.Set16{5, 6, 7, 8}, 3, 0, false},
		{"less_3_1_false", storage.Set16{5, 6, 7, 8}, 3, 1, false},
		{"less_3_2_false", storage.Set16{5, 6, 7, 8}, 3, 2, false},
		{"less_3_3_false", storage.Set16{5, 6, 7, 8}, 3, 3, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantLess, tt.grammemesSet.Less(tt.i, tt.j))
		})
	}
}

func TestSet16_Sort(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name      string
		origin    storage.Set16
		sortedSet storage.Set16
	}{
		{"sort_0", storage.Set16{}, storage.Set16{}},
		{"sort_1", storage.Set16{99}, storage.Set16{99}},
		{"sort_2", storage.Set16{22, 11}, storage.Set16{11, 22}},
		{"sort_3", storage.Set16{22, 33, 11}, storage.Set16{11, 22, 33}},
		{"sort_4", storage.Set16{44, 22, 33, 11}, storage.Set16{11, 22, 33, 44}},
		{"sort_5", storage.Set16{44, 22, 33, 55, 11}, storage.Set16{11, 22, 33, 44, 55}},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			tt.origin.Sort()
			require.Equal(t, tt.sortedSet.Len(), tt.origin.Len())
			require.Equal(t, tt.sortedSet, tt.origin)
		})
	}
}

func TestSet16_EqualTo(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set16
		another      storage.Set16
		want         bool
	}{
		{"empty_are_equal", storage.Set16{}, storage.Set16{}, true},
		{"equal_1_equal", storage.Set16{1}, storage.Set16{1}, true},
		{"different_1_is_not", storage.Set16{1}, storage.Set16{2}, false},
		{"equal_5_equals", storage.Set16{1, 3, 5, 7, 9}, storage.Set16{1, 3, 5, 7, 9}, true},
		{"different_5_equals", storage.Set16{1, 3, 5, 7, 9}, storage.Set16{1, 3, 5, 7}, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			require.Equal(t, tt.want, tt.grammemesSet.EqualTo(tt.another))
			require.Equal(t, tt.want, tt.another.EqualTo(tt.grammemesSet))
		})
	}
}

func TestSet16_WriteTo(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set16
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", storage.Set16{}, "0000", 2, false},
		{"len_1", storage.Set16{1}, "00010001", 4, false},
		{"len_2", storage.Set16{11, 22}, "0002000b0016", 6, false},
		{"len_3", storage.Set16{11, 22, 33}, "0003000b00160021", 8, false},
		{"len_5", storage.Set16{11, 33, 55, 77, 99}, "0005000b00210037004d0063", 12, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			wantData, err := hex.DecodeString(tt.wantW)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			gotN, writeErr := tt.grammemesSet.WriteTo(buf)
			require.Equal(t, tt.wantErr, writeErr != nil)
			if writeErr == nil {
				require.Equal(t, tt.wantN, gotN)
				require.Equal(t, buf.Bytes(), wantData)
			}
		})
	}
}

func TestSet16_ReadFrom(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set16
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", storage.Set16{}, "0000", 2, false},
		{"len_1", storage.Set16{1}, "00010001", 4, false},
		{"len_2", storage.Set16{11, 22}, "0002000b0016", 6, false},
		{"len_3", storage.Set16{11, 22, 33}, "0003000b00160021", 8, false},
		{"len_5", storage.Set16{11, 33, 55, 77, 99}, "0005000b00210037004d0063", 12, false},
		{"err_1_item_missed", storage.Set16{}, "0005000b00210037004d", 6, true},
		{"err_all_bytes_missed", storage.Set16{}, "0010", 16, true},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			wantData, err := hex.DecodeString(tt.wantW)
			require.NoError(t, err)

			buf := bytes.NewBuffer(wantData)
			newSet := make(storage.Set16, 0)
			gotN, readErr := newSet.ReadFrom(buf)
			require.Equalf(t, tt.wantErr, readErr != nil, "want error %v got %v", tt.wantErr, readErr)
			if readErr == nil {
				require.Equal(t, tt.wantN, gotN)
				require.Equal(t, len(wantData)/binutils.Uint16size-1, newSet.Len())
				require.True(t, tt.grammemesSet.EqualTo(newSet))
			}
		})
	}
}
