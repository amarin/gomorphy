// Package intern provides allocation-free string interning backed by stringsx.Arena.
package intern

import (
	"bytes"
	"math"

	"github.com/zeebo/xxh3"

	"github.com/amarin/gomorphy/internal/stringsx"
)

const (
	loadFactorNumerator   = 7
	loadFactorDenominator = 10
	minSlots              = 8
)

const emptyID = math.MaxUint32

type slot struct {
	hash uint64
	id   uint32
}

type Table struct {
	arena  *stringsx.Arena
	slots  []slot
	mask   uint64
	count  int
	growAt int
}

func New(arena *stringsx.Arena, expectedItems int) *Table {
	if expectedItems < minSlots {
		expectedItems = minSlots
	}

	size := pow2Ceil(expectedItems * loadFactorDenominator / loadFactorNumerator)

	t := &Table{
		arena:  arena,
		slots:  make([]slot, size),
		mask:   uint64(size - 1),
		growAt: int(float64(size) * loadFactor()),
	}

	for i := range t.slots {
		t.slots[i].id = emptyID
	}

	return t
}

func loadFactor() float64 {
	return float64(loadFactorNumerator) / float64(loadFactorDenominator)
}

func pow2Ceil(n int) int {
	size := minSlots
	for size < n {
		size <<= 1
	}

	return size
}

func (t *Table) Intern(b []byte) (id uint32, existed bool) {
	h := xxh3.Hash(b)
	pos := h & t.mask

	for {
		s := &t.slots[pos]

		if s.id == emptyID {
			newID := t.arena.Append(b)
			s.hash = h
			s.id = newID
			t.count++

			if t.count > t.growAt {
				t.rehash()
			}

			return newID, false
		}

		if s.hash == h && bytes.Equal(t.arena.Get(s.id), b) {
			return s.id, true
		}

		pos = (pos + 1) & t.mask
	}
}

func (t *Table) Get(id uint32) []byte {
	return t.arena.Get(id)
}

func (t *Table) Len() int {
	return t.count
}

func (t *Table) rehash() {
	newSize := len(t.slots) * 2
	old := t.slots

	t.slots = make([]slot, newSize)
	t.mask = uint64(newSize - 1)
	t.growAt = int(float64(newSize) * loadFactor())

	for i := range t.slots {
		t.slots[i].id = emptyID
	}

	for _, s := range old {
		if s.id == emptyID {
			continue
		}

		pos := s.hash & t.mask
		for t.slots[pos].id != emptyID {
			pos = (pos + 1) & t.mask
		}

		t.slots[pos] = s
	}
}
