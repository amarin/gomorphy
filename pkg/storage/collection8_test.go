package storage_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/storage"

	"github.com/stretchr/testify/require"
)

func TestSetIdx_Find(t *testing.T) { //nolint:paralleltest
	setIdxToTest := storage.Collection8x8{
		storage.Table8{storage.Set8{0}, storage.Set8{5}, storage.Set8{11}},
		storage.Table8{storage.Set8{0, 1}, storage.Set8{1, 2}, storage.Set8{2, 3}},
		storage.Table8{storage.Set8{1, 2, 3}, storage.Set8{1, 3, 4}, storage.Set8{2, 7, 10}},
		storage.Table8{storage.Set8{1, 2, 3, 4}, storage.Set8{1, 3, 4, 5}, storage.Set8{6, 7, 9, 11}},
		storage.Table8{},
	}

	for _, tt := range []struct { //nolint:paralleltest
		name  string
		find  storage.Set8
		want  storage.ID16
		found bool
	}{
		{"not_found_in_empty_column", storage.Set8{1, 2, 3, 4, 5}, 0, false},
		{"not_found_in_filled_column", storage.Set8{1, 2, 5}, 0, false},
		{"found_among_other_in_column0", storage.Set8{0}, 0, true},
		{"found_among_other_in_column1", storage.Set8{1, 2}, 257, true},
		{"found_among_other_in_column2", storage.Set8{1, 3, 4}, 513, true},
		{"found_among_other_in_column3", storage.Set8{1, 3, 4, 5}, 769, true},
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
		find storage.Set8
		want storage.ID16
	}{
		{"index_to_column0", storage.Set8{0}, 0},
		{"index_to_column1", storage.Set8{1, 2}, 256},
		{"index_to_column2", storage.Set8{1, 2, 5}, 512},
		{"index_to_column3", storage.Set8{1, 2, 3, 4}, 768},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			idx := make(storage.Collection8x8, 0)
			require.Equal(t, tt.want, idx.Index(tt.find))
		})
	}
}

func TestSetIdx_Get(t *testing.T) { //nolint:paralleltest
	setIdxToTest := storage.Collection8x8{
		storage.Table8{storage.Set8{0}, storage.Set8{5}, storage.Set8{11}},
		storage.Table8{storage.Set8{0, 1}, storage.Set8{1, 2}, storage.Set8{2, 3}},
		storage.Table8{storage.Set8{1, 2, 3}, storage.Set8{1, 3, 4}, storage.Set8{2, 7, 10}},
		storage.Table8{storage.Set8{1, 2, 3, 4}, storage.Set8{1, 3, 4, 5}, storage.Set8{6, 7, 9, 11}},
		storage.Table8{},
	}

	for _, tt := range []struct { //nolint:paralleltest
		name  string
		find  storage.Set8
		want  storage.ID16
		found bool
	}{
		{"not_found_in_empty_column", nil, 4*256 + 1, false},
		{"not_found_in_filled_column", nil, 3, false},
		{"found_in_column0", storage.Set8{0}, 0, true},
		{"found_in_column1", storage.Set8{1, 2}, 257, true},
		{"found_in_column2", storage.Set8{1, 3, 4}, 513, true},
		{"found_in_column3", storage.Set8{1, 3, 4, 5}, 769, true},
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
