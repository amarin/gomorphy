package indexing_test

import (
	"fmt"
	"testing"

	"github.com/amarin/gomorphy/pkg/indexing"
	"github.com/stretchr/testify/require"
)

var uint32tests = []struct {
	id32 indexing.ID32
	hi   indexing.ID16
	lo   indexing.ID16
}{
	{0xffffffff, 0xffff, 0xffff},
	{0x11112222, 0x1111, 0x2222},
	{0x33332222, 0x3333, 0x2222},
}

func TestCombine16(t *testing.T) {
	t.Parallel()

	for _, tt := range uint32tests {
		tt := tt
		t.Run(fmt.Sprintf("uint32_%v_hi_%v_lo_%v", tt.id32, tt.hi, tt.lo), func(t *testing.T) {
			t.Parallel()
			tt := tt
			require.Equal(t, tt.id32, indexing.Combine16(tt.hi, tt.lo))
		})
	}
}

func TestID32_Upper16(t *testing.T) {
	t.Parallel()

	for _, tt := range uint32tests {
		tt := tt
		t.Run(fmt.Sprintf("uint32_%v_hi_%v", tt.id32, tt.hi), func(t *testing.T) {
			t.Parallel()
			tt := tt
			require.Equal(t, tt.hi, tt.id32.Upper16())
		})
	}
}

func TestID32_Lower16(t *testing.T) {
	t.Parallel()

	for _, tt := range uint32tests {
		tt := tt
		t.Run(fmt.Sprintf("uint32_%v_lo_%v", tt.id32, tt.lo), func(t *testing.T) {
			t.Parallel()
			tt := tt
			require.Equal(t, tt.lo, tt.id32.Lower16())
		})
	}
}
