package index_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/index"
)

func TestTableIDCollection_Less(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		t    index.TableIDCollection
		args args
		want bool
	}{
		{"i_is_less_than_j", index.TableIDCollection{10, 12}, args{0, 1}, true},
		{"i_not_less_j", index.TableIDCollection{10, 10}, args{0, 1}, false},
		{"i_is_greater_than_j", index.TableIDCollection{12, 10}, args{0, 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.t.Less(tt.args.i, tt.args.j))
		})
	}
}

func TestTableIDCollection_Swap(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		t    index.TableIDCollection
		args args
	}{
		{"swap_0_1", index.TableIDCollection{10, 12}, args{0, 1}},
		{"swap_1_2", index.TableIDCollection{10, 12, 13}, args{1, 2}},
		{"swap_0_2", index.TableIDCollection{10, 12, 13}, args{0, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wasI := tt.t[tt.args.i]
			wasJ := tt.t[tt.args.j]
			tt.t.Swap(tt.args.i, tt.args.j)
			require.Equal(t, wasI, tt.t[tt.args.j])
			require.Equal(t, wasJ, tt.t[tt.args.i])
		})
	}
}
