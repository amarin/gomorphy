package storage_test //nolint:dupl

import (
	"github.com/amarin/gomorphy/pkg/storage"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTable8_Find(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name  string
		idx   storage.Table8
		find  storage.Set8
		want  storage.ID8
		found bool
	}{
		{
			"not_found_in_empty_column",
			nil,
			storage.Set8{1, 2, 3},
			0,
			false,
		},
		{
			"not_found_in_filled_column",
			storage.Table8{storage.Set8{1, 2, 4}, storage.Set8{2, 3, 4}, storage.Set8{0, 2}},
			storage.Set8{1, 2, 3},
			0,
			false,
		},
		{
			"found_among_other",
			storage.Table8{storage.Set8{2, 3, 4}, storage.Set8{0, 2}, storage.Set8{1, 2, 3}},
			storage.Set8{1, 2, 3},
			2,
			true,
		},
		{
			"found_alone",
			storage.Table8{storage.Set8{1, 2, 3}},
			storage.Set8{1, 2, 3},
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

func TestTable8_Index(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name string
		idx  storage.Table8
		find storage.Set8
		want storage.ID8
	}{
		{
			"indexed_first",
			nil,
			storage.Set8{1, 2, 3},
			0,
		},
		{
			"indexed_end",
			storage.Table8{storage.Set8{1, 2, 4}, storage.Set8{2, 3, 4}, storage.Set8{0, 2}},
			storage.Set8{1, 2, 3},
			3,
		},
		{
			"found_bottom",
			storage.Table8{storage.Set8{2, 3, 4}, storage.Set8{0, 2}, storage.Set8{1, 2, 3}},
			storage.Set8{1, 2, 3},
			2,
		},
		{
			"found_middle",
			storage.Table8{storage.Set8{2, 3, 4}, storage.Set8{1, 2, 3}, storage.Set8{0, 2}},
			storage.Set8{1, 2, 3},
			1,
		},
		{
			"found_top",
			storage.Table8{storage.Set8{1, 2, 3}},
			storage.Set8{1, 2, 3},
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
