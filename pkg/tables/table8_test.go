package tables_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/ids"
	"github.com/amarin/gomorphy/pkg/sets"
	"github.com/amarin/gomorphy/pkg/tables"
	"github.com/stretchr/testify/require"
)

func TestTable8x8_Find(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name  string
		idx   tables.Table8
		find  sets.Set8
		want  ids.ID8
		found bool
	}{
		{
			"not_found_in_empty_column",
			nil,
			sets.Set8{1, 2, 3},
			0,
			false,
		},
		{
			"not_found_in_filled_column",
			tables.Table8{sets.Set8{1, 2, 4}, sets.Set8{2, 3, 4}, sets.Set8{0, 2}},
			sets.Set8{1, 2, 3},
			0,
			false,
		},
		{
			"found_among_other",
			tables.Table8{sets.Set8{2, 3, 4}, sets.Set8{0, 2}, sets.Set8{1, 2, 3}},
			sets.Set8{1, 2, 3},
			2,
			true,
		},
		{
			"found_alone",
			tables.Table8{sets.Set8{1, 2, 3}},
			sets.Set8{1, 2, 3},
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

func TestTable8x8_Index(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name string
		idx  tables.Table8
		find sets.Set8
		want ids.ID8
	}{
		{
			"indexed_first",
			nil,
			sets.Set8{1, 2, 3},
			0,
		},
		{
			"indexed_end",
			tables.Table8{sets.Set8{1, 2, 4}, sets.Set8{2, 3, 4}, sets.Set8{0, 2}},
			sets.Set8{1, 2, 3},
			3,
		},
		{
			"found_bottom",
			tables.Table8{sets.Set8{2, 3, 4}, sets.Set8{0, 2}, sets.Set8{1, 2, 3}},
			sets.Set8{1, 2, 3},
			2,
		},
		{
			"found_middle",
			tables.Table8{sets.Set8{2, 3, 4}, sets.Set8{1, 2, 3}, sets.Set8{0, 2}},
			sets.Set8{1, 2, 3},
			1,
		},
		{
			"found_top",
			tables.Table8{sets.Set8{1, 2, 3}},
			sets.Set8{1, 2, 3},
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