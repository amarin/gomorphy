package storage_test

import (
	"fmt"
	"github.com/amarin/gomorphy/pkg/storage"
	"testing"

	"github.com/stretchr/testify/require"
)

var uint16tests = []struct {
	id16 storage.ID16
	hi   storage.ID8
	lo   storage.ID8
}{
	{0x0000, 0x00, 0x00},
	{0x1234, 0x12, 0x34},
	{0x5678, 0x56, 0x78},
	{0x9abc, 0x9a, 0xbc},
	{0xdeef, 0xde, 0xef},
}

func TestCombine8(t *testing.T) {
	t.Parallel()

	for _, tt := range uint16tests {
		tt := tt
		t.Run(fmt.Sprintf("uint16_%v_hi_%v_lo_%v", tt.id16, tt.hi, tt.lo), func(t *testing.T) {
			t.Parallel()
			tt := tt
			require.Equal(t, tt.id16, storage.Combine8(tt.hi, tt.lo))
		})
	}
}

func TestID16_Upper8(t *testing.T) {
	t.Parallel()

	for _, tt := range uint16tests {
		tt := tt
		t.Run(fmt.Sprintf("uint16_%v_hi_%v", tt.id16, tt.hi), func(t *testing.T) {
			t.Parallel()
			tt := tt
			require.Equal(t, tt.hi, tt.id16.Upper())
		})
	}
}

func TestID16_Lower8(t *testing.T) {
	t.Parallel()

	for _, tt := range uint16tests {
		tt := tt
		t.Run(fmt.Sprintf("uint16_%v_lo_%v", tt.id16, tt.lo), func(t *testing.T) {
			t.Parallel()
			tt := tt
			require.Equal(t, tt.lo, tt.id16.Lower())
		})
	}
}
