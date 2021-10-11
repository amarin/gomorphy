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
		want  grammemes.SetID
		found bool
	}{
		{
			"not_found_in_empty_column",
			nil,
			grammemes.Set{1, 2, 3},
			0,
			false,
		},
		{
			"not_found_in_filled_column",
			grammemes.Column{grammemes.Set{1, 2, 4}, grammemes.Set{2, 3, 4}, grammemes.Set{0, 2}},
			grammemes.Set{1, 2, 3},
			0,
			false,
		},
		{
			"found_among_other",
			grammemes.Column{grammemes.Set{2, 3, 4}, grammemes.Set{0, 2}, grammemes.Set{1, 2, 3}},
			grammemes.Set{1, 2, 3},
			2,
			true,
		},
		{
			"found_alone",
			grammemes.Column{grammemes.Set{1, 2, 3}},
			grammemes.Set{1, 2, 3},
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
		want grammemes.SetID
	}{
		{
			"indexed_first",
			nil,
			grammemes.Set{1, 2, 3},
			0,
		},
		{
			"indexed_end",
			grammemes.Column{grammemes.Set{1, 2, 4}, grammemes.Set{2, 3, 4}, grammemes.Set{0, 2}},
			grammemes.Set{1, 2, 3},
			3,
		},
		{
			"found_bottom",
			grammemes.Column{grammemes.Set{2, 3, 4}, grammemes.Set{0, 2}, grammemes.Set{1, 2, 3}},
			grammemes.Set{1, 2, 3},
			2,
		},
		{
			"found_middle",
			grammemes.Column{grammemes.Set{2, 3, 4}, grammemes.Set{1, 2, 3}, grammemes.Set{0, 2}},
			grammemes.Set{1, 2, 3},
			1,
		},
		{
			"found_top",
			grammemes.Column{grammemes.Set{1, 2, 3}},
			grammemes.Set{1, 2, 3},
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
