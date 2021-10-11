package grammemes_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/grammemes"
	"github.com/stretchr/testify/require"
)

func TestSetColumn_Find(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name  string
		idx   grammemes.SetColumn
		find  grammemes.Set
		want  uint8
		found bool
	}{
		{
			"not_found_in_empty_column",
			nil,
			[]uint8{1, 2, 3},
			0,
			false,
		},
		{
			"not_found_in_filled_column",
			grammemes.SetColumn{[]uint8{1, 2, 4}, []uint8{2, 3, 4}, []uint8{0, 2}},
			[]uint8{1, 2, 3},
			0,
			false,
		},
		{
			"found_among_other",
			grammemes.SetColumn{[]uint8{2, 3, 4}, []uint8{0, 2}, []uint8{1, 2, 3}},
			[]uint8{1, 2, 3},
			2,
			true,
		},
		{
			"found_alone",
			grammemes.SetColumn{[]uint8{1, 2, 3}},
			[]uint8{1, 2, 3},
			0,
			true,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			got, found := tt.idx.Find(tt.find)
			require.Equal(t, tt.found, found)
			if !found {
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSetColumn_Index(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name string
		idx  grammemes.SetColumn
		find grammemes.Set
		want uint8
	}{
		{
			"indexed_first",
			nil,
			[]uint8{1, 2, 3},
			0,
		},
		{
			"indexed_end",
			grammemes.SetColumn{[]uint8{1, 2, 4}, []uint8{2, 3, 4}, []uint8{0, 2}},
			[]uint8{1, 2, 3},
			3,
		},
		{
			"found_bottom",
			grammemes.SetColumn{[]uint8{2, 3, 4}, []uint8{0, 2}, []uint8{1, 2, 3}},
			[]uint8{1, 2, 3},
			2,
		},
		{
			"found_middle",
			grammemes.SetColumn{[]uint8{2, 3, 4}, []uint8{1, 2, 3}, []uint8{0, 2}},
			[]uint8{1, 2, 3},
			1,
		},
		{
			"found_top",
			grammemes.SetColumn{[]uint8{1, 2, 3}},
			[]uint8{1, 2, 3},
			0,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			require.Equal(t, tt.want, tt.idx.Index(tt.find))
			got, found := tt.idx.Find(tt.find)
			require.True(t, found)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSetIdx_Find(t *testing.T) { //nolint:paralleltest
	setIdxToTest := grammemes.SetIdx{
		grammemes.SetColumn{[]uint8{0}, []uint8{5}, []uint8{11}},
		grammemes.SetColumn{[]uint8{0, 1}, []uint8{1, 2}, []uint8{2, 3}},
		grammemes.SetColumn{[]uint8{1, 2, 3}, []uint8{1, 3, 4}, []uint8{2, 7, 10}},
		grammemes.SetColumn{[]uint8{1, 2, 3, 4}, []uint8{1, 3, 4, 5}, []uint8{6, 7, 9, 11}},
		grammemes.SetColumn{},
	}

	for _, tt := range []struct { //nolint:paralleltest
		name  string
		find  grammemes.Set
		want  uint16
		found bool
	}{
		{"not_found_in_empty_column", []uint8{1, 2, 3, 4, 5}, 0, false},
		{"not_found_in_filled_column", []uint8{1, 2, 5}, 0, false},
		{"found_among_other_in_column0", []uint8{0}, 0, true},
		{"found_among_other_in_column1", []uint8{1, 2}, 257, true},
		{"found_among_other_in_column2", []uint8{1, 3, 4}, 513, true},
		{"found_among_other_in_column3", []uint8{1, 3, 4, 5}, 769, true},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			got, found := setIdxToTest.Find(tt.find)
			require.Equal(t, tt.found, found)
			if !found {
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSetIdx_Index(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name string
		find grammemes.Set
		want uint16
	}{
		{"index_to_column0", []uint8{0}, 0},
		{"index_to_column1", []uint8{1, 2}, 256},
		{"index_to_column2", []uint8{1, 2, 5}, 512},
		{"index_to_column3", []uint8{1, 2, 3, 4}, 768},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			idx := make(grammemes.SetIdx, 0)
			require.Equal(t, tt.want, idx.Index(tt.find))
		})
	}
}

func TestSetIdx_Get(t *testing.T) { //nolint:paralleltest
	setIdxToTest := grammemes.SetIdx{
		grammemes.SetColumn{[]uint8{0}, []uint8{5}, []uint8{11}},
		grammemes.SetColumn{[]uint8{0, 1}, []uint8{1, 2}, []uint8{2, 3}},
		grammemes.SetColumn{[]uint8{1, 2, 3}, []uint8{1, 3, 4}, []uint8{2, 7, 10}},
		grammemes.SetColumn{[]uint8{1, 2, 3, 4}, []uint8{1, 3, 4, 5}, []uint8{6, 7, 9, 11}},
		grammemes.SetColumn{},
	}

	for _, tt := range []struct { //nolint:paralleltest
		name  string
		find  grammemes.Set
		want  uint16
		found bool
	}{
		{"not_found_in_empty_column", nil, 4*256 + 1, false},
		{"not_found_in_filled_column", nil, 3, false},
		{"found_in_column0", []uint8{0}, 0, true},
		{"found_in_column1", []uint8{1, 2}, 257, true},
		{"found_in_column2", []uint8{1, 3, 4}, 513, true},
		{"found_in_column3", []uint8{1, 3, 4, 5}, 769, true},
		{"missed_in_not_existed_column", nil, 2560, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			got, found := setIdxToTest.Get(tt.want)
			require.Equal(
				t, tt.found, found,
				"Get(%v)=(%v,%v), expected (%v,%v)",
				tt.want, got, found, tt.want, tt.found,
			)
			if !found {
				return
			}
			require.Equal(t, tt.find, got)
		})
	}
}
