package storage_test

import (
	"bytes"
	"encoding/hex"
	"github.com/amarin/gomorphy/pkg/storage"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSet8_Swap(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set8
		i            int
		j            int
	}{
		{"swap_0_1", storage.Set8{6, 7, 8, 9}, 0, 1},
		{"swap_1_2", storage.Set8{1, 2, 3, 4}, 1, 2},
		{"swap_2_3", storage.Set8{5, 6, 7, 8}, 2, 3},
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

func TestSet8_Less(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set8
		i            int
		j            int
		wantLess     bool
	}{
		{"less_0_0_false", storage.Set8{6, 7, 8, 9}, 0, 0, false},
		{"less_0_1_true", storage.Set8{6, 7, 8, 9}, 0, 1, true},
		{"less_0_2_true", storage.Set8{6, 7, 8, 9}, 0, 2, true},
		{"less_0_3_true", storage.Set8{6, 7, 8, 9}, 0, 3, true},
		{"less_1_0_false", storage.Set8{1, 2, 3, 4}, 1, 0, false},
		{"less_1_1_false", storage.Set8{6, 7, 8, 9}, 1, 1, false},
		{"less_1_2_true", storage.Set8{6, 7, 8, 9}, 1, 2, true},
		{"less_1_3_true", storage.Set8{6, 7, 8, 9}, 1, 3, true},
		{"less_2_0_false", storage.Set8{5, 6, 7, 8}, 2, 0, false},
		{"less_2_1_false", storage.Set8{5, 6, 7, 8}, 2, 1, false},
		{"less_2_2_false", storage.Set8{5, 6, 7, 8}, 2, 2, false},
		{"less_2_3_true", storage.Set8{5, 6, 7, 8}, 2, 3, true},
		{"less_3_0_false", storage.Set8{5, 6, 7, 8}, 3, 0, false},
		{"less_3_1_false", storage.Set8{5, 6, 7, 8}, 3, 1, false},
		{"less_3_2_false", storage.Set8{5, 6, 7, 8}, 3, 2, false},
		{"less_3_3_false", storage.Set8{5, 6, 7, 8}, 3, 3, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.wantLess, tt.grammemesSet.Less(tt.i, tt.j))
		})
	}
}

func TestSet8_Sort(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name      string
		origin    storage.Set8
		sortedSet storage.Set8
	}{
		{"sort_0", storage.Set8{}, storage.Set8{}},
		{"sort_1", storage.Set8{99}, storage.Set8{99}},
		{"sort_2", storage.Set8{22, 11}, storage.Set8{11, 22}},
		{"sort_3", storage.Set8{22, 33, 11}, storage.Set8{11, 22, 33}},
		{"sort_4", storage.Set8{44, 22, 33, 11}, storage.Set8{11, 22, 33, 44}},
		{"sort_5", storage.Set8{44, 22, 33, 55, 11}, storage.Set8{11, 22, 33, 44, 55}},
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

func TestSet8_EqualTo(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set8
		another      storage.Set8
		want         bool
	}{
		{"empty_are_equal", storage.Set8{}, storage.Set8{}, true},
		{"equal_1_equal", storage.Set8{1}, storage.Set8{1}, true},
		{"different_1_is_not", storage.Set8{1}, storage.Set8{2}, false},
		{"equal_5_equals", storage.Set8{1, 3, 5, 7, 9}, storage.Set8{1, 3, 5, 7, 9}, true},
		{"different_5_equals", storage.Set8{1, 3, 5, 7, 9}, storage.Set8{1, 3, 5, 7}, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			require.Equal(t, tt.want, tt.grammemesSet.EqualTo(tt.another))
			require.Equal(t, tt.want, tt.another.EqualTo(tt.grammemesSet))
		})
	}
}

func TestSet8_WriteTo(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set8
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", storage.Set8{}, "00", 1, false},
		{"len_1", storage.Set8{1}, "0101", 2, false},
		{"len_2", storage.Set8{11, 22}, "020b16", 3, false},
		{"len_3", storage.Set8{11, 22, 33}, "030b1621", 4, false},
		{"len_5", storage.Set8{11, 33, 55, 77, 99}, "050b21374d63", 6, false},
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

func TestSet8_ReadFrom(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Set8
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", storage.Set8{}, "00", 1, false},
		{"len_1", storage.Set8{255}, "01ff", 2, false},
		{"len_2", storage.Set8{255, 253}, "02fffd", 3, false},
		{"len_3", storage.Set8{11, 22, 33}, "030b1621", 4, false},
		{"len_5", storage.Set8{11, 33, 55, 77, 99}, "050b21374d63", 6, false},
		{"err_1_byte_missed", storage.Set8{}, "050b21374d", 6, true},
		{"err_2_bytes_missed", storage.Set8{}, "050b2137", 6, true},
		{"err_3_bytes_missed", storage.Set8{}, "050b21", 6, true},
		{"err_4_bytes_missed", storage.Set8{}, "050b", 6, true},
		{"err_all_bytes_missed", storage.Set8{}, "10", 16, true},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			wantData, err := hex.DecodeString(tt.wantW)
			require.NoError(t, err)

			buf := bytes.NewBuffer(wantData)
			newSet := make(storage.Set8, 0)
			gotN, writeErr := newSet.ReadFrom(buf)
			require.Equal(t, tt.wantErr, writeErr != nil)
			if writeErr == nil {
				require.Equal(t, tt.wantN, gotN)
				require.Equal(t, len(wantData)-1, newSet.Len())
				require.Equal(t, tt.grammemesSet, newSet)
			}
		})
	}
}
