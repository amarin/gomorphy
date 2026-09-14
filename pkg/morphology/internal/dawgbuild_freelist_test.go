package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlotAllocatorFirstAllocAvoidsRoot(t *testing.T) {
	a := newSlotAllocator()
	base, ok := a.alloc(0, []byte{0x41})
	require.True(t, ok)
	assert.NotZero(t, base, "base must never be slot 0 (reserved for DAWG root)")
	assert.True(t, encodable(0^base))
}

func TestSlotAllocatorRejectsCollidingBase(t *testing.T) {
	a := newSlotAllocator()
	// Force slot 5 and 5^0x10=21 to be used, so a later request for the
	// same label must not return base=5.
	base1, ok := a.alloc(0, []byte{0x10})
	require.True(t, ok)

	// Directly probe: base1 and base1^0x10 must now be reported used.
	base2, ok := a.alloc(1, []byte{0x10})
	require.True(t, ok)
	assert.NotEqual(t, base1, base2, "second alloc must not reuse base1's now-occupied slots")
	// The two children (base^0x10) must also be distinct and not equal to
	// either base, otherwise the double-array would alias two nodes.
	child1 := base1 ^ 0x10
	child2 := base2 ^ 0x10
	assert.NotEqual(t, child1, base2)
	assert.NotEqual(t, child2, base1)
}

func TestSlotAllocatorNoChildrenStillReservesBase(t *testing.T) {
	a := newSlotAllocator()
	base, ok := a.alloc(0, nil)
	require.True(t, ok)
	assert.NotZero(t, base)

	// A second alloc for the same index/no-children case must not reuse it.
	base2, ok := a.alloc(0, nil)
	require.True(t, ok)
	assert.NotEqual(t, base, base2)
}

func TestSlotAllocatorGrowsCapacityOnDemand(t *testing.T) {
	a := newSlotAllocator()
	assert.Less(t, a.cap(), uint32(1<<20), "must start small, not pre-sized to 1<<30")

	// Exhaust small capacities repeatedly; allocator must keep succeeding
	// by growing, never returning ok=false for a satisfiable request.
	seen := make(map[uint32]bool)
	for i := 0; i < 5000; i++ {
		base, ok := a.alloc(uint32(i), []byte{byte(i % 7), byte(i % 251)})
		require.True(t, ok)
		require.False(t, seen[base], "base %d reused", base)
		seen[base] = true
	}
	assert.Greater(t, a.cap(), uint32(5000))
}

func TestSlotAllocatorHintStartsSearchNearLastSuccess(t *testing.T) {
	a := newSlotAllocator()
	base1, ok := a.alloc(0, []byte{0x22, 0x30})
	require.True(t, ok)

	// A later alloc with the same first label should land at or after
	// base1 (the hint should skip the region already known to be occupied
	// near the head), not necessarily back at the very first free slot.
	base2, ok := a.alloc(1, []byte{0x22})
	require.True(t, ok)
	assert.GreaterOrEqual(t, base2, base1)
}
