package build

import (
	"github.com/zeebo/xxh3"
)

// HashEntry is one slot of the open-addressing exact-match hash.
type HashEntry struct {
	Hash  uint64
	State uint32
}

const hashEmptyState = ^uint32(0)

const (
	hashLoadNum = 7
	hashLoadDen = 10
)

func pow2Ceil(n int) int {
	size := 8
	for size < n {
		size <<= 1
	}

	return size
}

// buildExactHash maps every terminal text node into a flat probe table.
// nodeState must return the compiled trie state for text id.
func buildExactHash(textCount int, nodeState func(textID uint32) (uint32, bool), text func(textID uint32) []byte) ([]HashEntry, uint64) {
	capacity := pow2Ceil(textCount * hashLoadDen / hashLoadNum)
	if capacity < 8 {
		capacity = 8
	}

	slots := make([]HashEntry, capacity)
	for i := range slots {
		slots[i].State = hashEmptyState
	}

	mask := uint64(capacity - 1)

	for id := 0; id < textCount; id++ {
		state, ok := nodeState(uint32(id))
		if !ok {
			continue
		}

		h := xxh3.Hash(text(uint32(id)))
		pos := h & mask

		for {
			if slots[pos].State == hashEmptyState {
				slots[pos] = HashEntry{Hash: h, State: state}

				break
			}

			pos = (pos + 1) & mask
		}
	}

	return slots, mask
}
