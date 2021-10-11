package grammemes_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/amarin/gomorphy/pkg/grammemes"
	"github.com/stretchr/testify/require"
)

func TestSet_Swap(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name         string
		grammemesSet grammemes.Set
		i            int
		j            int
	}{
		{"swap_0_1", grammemes.Set{6, 7, 8, 9}, 0, 1},
		{"swap_1_2", grammemes.Set{1, 2, 3, 4}, 1, 2},
		{"swap_2_3", grammemes.Set{5, 6, 7, 8}, 2, 3},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			wantI := tt.grammemesSet[tt.i]
			wantJ := tt.grammemesSet[tt.j]
			tt.grammemesSet.Swap(tt.i, tt.j)
			require.Equal(t, tt.grammemesSet[tt.i], wantJ)
			require.Equal(t, tt.grammemesSet[tt.j], wantI)
		})
	}
}

func TestSet_Less(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name         string
		grammemesSet grammemes.Set
		i            int
		j            int
		wantLess     bool
	}{
		{"less_0_0_false", grammemes.Set{6, 7, 8, 9}, 0, 0, false},
		{"less_0_1_true", grammemes.Set{6, 7, 8, 9}, 0, 1, true},
		{"less_0_2_true", grammemes.Set{6, 7, 8, 9}, 0, 2, true},
		{"less_0_3_true", grammemes.Set{6, 7, 8, 9}, 0, 3, true},
		{"less_1_0_false", grammemes.Set{1, 2, 3, 4}, 1, 0, false},
		{"less_1_1_false", grammemes.Set{6, 7, 8, 9}, 1, 1, false},
		{"less_1_2_true", grammemes.Set{6, 7, 8, 9}, 1, 2, true},
		{"less_1_3_true", grammemes.Set{6, 7, 8, 9}, 1, 3, true},
		{"less_2_0_false", grammemes.Set{5, 6, 7, 8}, 2, 0, false},
		{"less_2_1_false", grammemes.Set{5, 6, 7, 8}, 2, 1, false},
		{"less_2_2_false", grammemes.Set{5, 6, 7, 8}, 2, 2, false},
		{"less_2_3_true", grammemes.Set{5, 6, 7, 8}, 2, 3, true},
		{"less_3_0_false", grammemes.Set{5, 6, 7, 8}, 3, 0, false},
		{"less_3_1_false", grammemes.Set{5, 6, 7, 8}, 3, 1, false},
		{"less_3_2_false", grammemes.Set{5, 6, 7, 8}, 3, 2, false},
		{"less_3_3_false", grammemes.Set{5, 6, 7, 8}, 3, 3, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.wantLess, tt.grammemesSet.Less(tt.i, tt.j))
		})
	}
}

func TestSet_Sort(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		origin    grammemes.Set
		sortedSet grammemes.Set
	}{
		{"sort_0", grammemes.Set{}, grammemes.Set{}},
		{"sort_1", grammemes.Set{99}, grammemes.Set{99}},
		{"sort_2", grammemes.Set{22, 11}, grammemes.Set{11, 22}},
		{"sort_3", grammemes.Set{22, 33, 11}, grammemes.Set{11, 22, 33}},
		{"sort_4", grammemes.Set{44, 22, 33, 11}, grammemes.Set{11, 22, 33, 44}},
		{"sort_5", grammemes.Set{44, 22, 33, 55, 11}, grammemes.Set{11, 22, 33, 44, 55}},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			tt.origin.Sort()
			require.Equal(t, tt.sortedSet.Len(), tt.origin.Len())
			require.Equal(t, tt.sortedSet, tt.origin)
		})
	}
}

func TestSet_EqualTo(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name         string
		grammemesSet grammemes.Set
		another      grammemes.Set
		want         bool
	}{
		{"empty_are_equal", grammemes.Set{}, grammemes.Set{}, true},
		{"equal_1_equal", grammemes.Set{1}, grammemes.Set{1}, true},
		{"different_1_is_not", grammemes.Set{1}, grammemes.Set{2}, false},
		{"equal_5_equals", grammemes.Set{1, 3, 5, 7, 9}, grammemes.Set{1, 3, 5, 7, 9}, true},
		{"different_5_equals", grammemes.Set{1, 3, 5, 7, 9}, grammemes.Set{1, 3, 5, 7}, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			require.Equal(t, tt.want, tt.grammemesSet.EqualTo(tt.another))
			require.Equal(t, tt.want, tt.another.EqualTo(tt.grammemesSet))
		})
	}
}

func TestSet_WriteTo(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name         string
		grammemesSet grammemes.Set
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", grammemes.Set{}, "00", 1, false},
		{"len_1", grammemes.Set{1}, "0101", 2, false},
		{"len_2", grammemes.Set{11, 22}, "020b16", 3, false},
		{"len_3", grammemes.Set{11, 22, 33}, "030b1621", 4, false},
		{"len_5", grammemes.Set{11, 33, 55, 77, 99}, "050b21374d63", 6, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

func TestSet_ReadFrom(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name         string
		grammemesSet grammemes.Set
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", grammemes.Set{}, "00", 1, false},
		{"len_1", grammemes.Set{255}, "01ff", 2, false},
		{"len_2", grammemes.Set{255, 253}, "02fffd", 3, false},
		{"len_3", grammemes.Set{11, 22, 33}, "030b1621", 4, false},
		{"len_5", grammemes.Set{11, 33, 55, 77, 99}, "050b21374d63", 6, false},
		{"err_1_byte_missed", grammemes.Set{}, "050b21374d", 6, true},
		{"err_2_bytes_missed", grammemes.Set{}, "050b2137", 6, true},
		{"err_3_bytes_missed", grammemes.Set{}, "050b21", 6, true},
		{"err_4_bytes_missed", grammemes.Set{}, "050b", 6, true},
		{"err_all_bytes_missed", grammemes.Set{}, "10", 16, true},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt := tt
			wantData, err := hex.DecodeString(tt.wantW)
			require.NoError(t, err)

			buf := bytes.NewBuffer(wantData)
			newSet := make(grammemes.Set, 0)
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
