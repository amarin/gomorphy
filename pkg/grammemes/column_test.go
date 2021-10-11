package grammemes_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/grammemes"
	"github.com/stretchr/testify/require"
)

func TestSetColumn_Find(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name  string
		idx   grammemes.Column
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
			grammemes.Column{[]uint8{1, 2, 4}, []uint8{2, 3, 4}, []uint8{0, 2}},
			[]uint8{1, 2, 3},
			0,
			false,
		},
		{
			"found_among_other",
			grammemes.Column{[]uint8{2, 3, 4}, []uint8{0, 2}, []uint8{1, 2, 3}},
			[]uint8{1, 2, 3},
			2,
			true,
		},
		{
			"found_alone",
			grammemes.Column{[]uint8{1, 2, 3}},
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
		idx  grammemes.Column
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
			grammemes.Column{[]uint8{1, 2, 4}, []uint8{2, 3, 4}, []uint8{0, 2}},
			[]uint8{1, 2, 3},
			3,
		},
		{
			"found_bottom",
			grammemes.Column{[]uint8{2, 3, 4}, []uint8{0, 2}, []uint8{1, 2, 3}},
			[]uint8{1, 2, 3},
			2,
		},
		{
			"found_middle",
			grammemes.Column{[]uint8{2, 3, 4}, []uint8{1, 2, 3}, []uint8{0, 2}},
			[]uint8{1, 2, 3},
			1,
		},
		{
			"found_top",
			grammemes.Column{[]uint8{1, 2, 3}},
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
