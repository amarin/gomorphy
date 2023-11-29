package storage_test //nolint:dupl

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/storage"

	"github.com/stretchr/testify/require"
)

func TestTable16_Find(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name  string
		idx   storage.Table16
		find  storage.Set16
		want  storage.ID16
		found bool
	}{
		{
			"not_found_in_empty_column",
			nil,
			storage.Set16{1, 2, 3},
			0,
			false,
		},
		{
			"not_found_in_filled_column",
			storage.Table16{storage.Set16{1, 2, 4}, storage.Set16{2, 3, 4}, storage.Set16{0, 2}},
			storage.Set16{1, 2, 3},
			0,
			false,
		},
		{
			"found_among_other",
			storage.Table16{storage.Set16{2, 3, 4}, storage.Set16{0, 2}, storage.Set16{1, 2, 3}},
			storage.Set16{1, 2, 3},
			2,
			true,
		},
		{
			"found_alone",
			storage.Table16{storage.Set16{1, 2, 3}},
			storage.Set16{1, 2, 3},
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

func TestTable16_Index(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name string
		idx  storage.Table16
		find storage.Set16
		want storage.ID16
	}{
		{
			"indexed_first",
			nil,
			storage.Set16{1, 2, 3},
			0,
		},
		{
			"indexed_end",
			storage.Table16{storage.Set16{1, 2, 4}, storage.Set16{2, 3, 4}, storage.Set16{0, 2}},
			storage.Set16{1, 2, 3},
			3,
		},
		{
			"found_bottom",
			storage.Table16{storage.Set16{2, 3, 4}, storage.Set16{0, 2}, storage.Set16{1, 2, 3}},
			storage.Set16{1, 2, 3},
			2,
		},
		{
			"found_middle",
			storage.Table16{storage.Set16{2, 3, 4}, storage.Set16{1, 2, 3}, storage.Set16{0, 2}},
			storage.Set16{1, 2, 3},
			1,
		},
		{
			"found_top",
			storage.Table16{storage.Set16{1, 2, 3}},
			storage.Set16{1, 2, 3},
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
