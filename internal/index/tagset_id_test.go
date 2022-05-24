package index

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagSetTableNumber_Add(t *testing.T) {
	tests := []struct {
		name              string
		tagSetTableNumber TagSetTableNumber
		increment         int
		want              TagSetTableNumber
	}{
		{"increment_0_by_1", 0, 1, 1},
		{"increment_1_by_1", 1, 1, 2},
		{"increment_13_by_11", 13, 11, 24},
		{"decrement_10_by_7", 10, -7, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tagSetTableNumber.Add(tt.increment))
		})
	}
}

func TestTagSetTableNumber_TagSetID(t *testing.T) {
	tests := []struct {
		name              string
		tagSetTableNumber TagSetTableNumber
		subID             TagSetSubID
		want              TagSetID
	}{
		{"0_in_0", 0, 0, 0},
		{"1_in_0", 0, 1, 0x1},
		{"1_in_1", 1, 1, 0x10001},
		{"10_in_10", 10, 10, 0xa000a},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tagSetTableNumber.TagSetID(tt.subID))
		})
	}
}

func TestTagSetID_TagSetSubID(t *testing.T) {
	tests := []struct {
		name     string
		tagSetID TagSetID
		want     TagSetSubID
	}{
		{"0_of_0", 0, 0},
		{"10_of_0xa", 0xa, 10},
		{"10_of_0x1000a", 0x1000a, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tagSetID.TagSetSubID())
		})
	}
}

func TestTagSetID_TagSetTableNumber(t *testing.T) {
	tests := []struct {
		name     string
		tagSetID TagSetID
		want     TagSetTableNumber
	}{
		{"0_of_0x1", 0x1, 0},
		{"1_of_0x10001", 0x10001, 1},
		{"1_of_0xc0001", 0xc0001, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tagSetID.TagSetTableNumber())
		})
	}
}
