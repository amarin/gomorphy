package internal

// slotAllocator picks free base slots for the double-array layout using
// an intrusive doubly-linked free list (a technique from dawgdic/cedar/
// Darts, Aoe 1989): free slots are linked via next/prev, and the search
// only walks that list — already-used slots are never rescanned, unlike
// a linear bitset scan starting from zero for every node.
//
// Slot 0 is reserved for the DAWG root (compileImpl always places the
// root at index 0) and doubles as the sentinel value "no neighbor" for
// the free list: it is never free, so 0 unambiguously means "end/start
// of the list".
type slotAllocator struct {
	used []bool
	next []uint32 // next[i]: the next free slot after i; 0 = no end
	prev []uint32 // prev[i]: the previous free slot before i; 0 = no start
	head uint32   // the first free slot; 0 if the list is empty
	tail uint32   // the last free slot; 0 if the list is empty
	hint [256]uint32
}

// maxSlotAllocatorCap is a safety ceiling: real dictionaries never reach
// it (hundreds of millions of nodes) — it's only a replacement for the
// old errDAWGBuild path in case of a pathological key set.
const maxSlotAllocatorCap = 1 << 30

func newSlotAllocator() *slotAllocator {
	return &slotAllocator{
		used: []bool{true}, // slot 0 is used from the start (the root)
		next: []uint32{0},
		prev: []uint32{0},
	}
}

func (a *slotAllocator) cap() uint32 { return uint32(len(a.used)) }

// grow expands capacity to at least n slots, appending the new slots to
// the tail of the free list in ascending index order.
func (a *slotAllocator) grow(n uint32) {
	old := a.cap()
	if n <= old {
		return
	}
	a.used = append(a.used, make([]bool, n-old)...)
	a.next = append(a.next, make([]uint32, n-old)...)
	a.prev = append(a.prev, make([]uint32, n-old)...)
	for i := old; i < n; i++ {
		a.linkTail(i)
	}
}

func (a *slotAllocator) linkTail(i uint32) {
	if a.head == 0 {
		a.head = i
		a.tail = i
		return
	}
	a.next[a.tail] = i
	a.prev[i] = a.tail
	a.tail = i
}

func (a *slotAllocator) unlink(i uint32) {
	p, n := a.prev[i], a.next[i]
	if p != 0 {
		a.next[p] = n
	} else {
		a.head = n
	}
	if n != 0 {
		a.prev[n] = p
	} else {
		a.tail = p
	}
	a.prev[i], a.next[i] = 0, 0
}

// markUsed reserves slot i, growing capacity if needed. No-op if the
// slot is already used.
func (a *slotAllocator) markUsed(i uint32) {
	if i >= a.cap() {
		a.grow(i + 1)
	}
	if a.used[i] {
		return
	}
	a.used[i] = true
	a.unlink(i)
}

// isUsed reports whether slot i is used (slots beyond the current
// capacity count as free — nothing has needed them yet).
func (a *slotAllocator) isUsed(i uint32) bool {
	return i < a.cap() && a.used[i]
}

// fits checks that base itself is free and that every base^label for
// labels is also free — i.e. the node can be placed at this base
// without collisions.
func (a *slotAllocator) fits(base uint32, labels []byte) bool {
	if base == 0 || a.isUsed(base) {
		return false
	}
	for _, l := range labels {
		if a.isUsed(base ^ uint32(l)) {
			return false
		}
	}
	return true
}

// commit reserves base and every child slot base^label.
func (a *slotAllocator) commit(base uint32, labels []byte) {
	a.markUsed(base)
	for _, l := range labels {
		a.markUsed(base ^ uint32(l))
	}
}

// alloc finds and reserves a base for the node at position index: base
// itself and every base^label for labels must be free, and index^base
// must be representable in the dictionary unit's offset field
// (encodable). The search walks the free-slot list starting from the
// hint for the first label (if any and still free), otherwise from the
// head of the list. If the list is exhausted without success, capacity
// is doubled and a full pass is retried; this only happens at growth
// boundaries, so the total cost of repeated full passes is bounded by
// O(n log n), not O(n) per node.
func (a *slotAllocator) alloc(index uint32, labels []byte) (uint32, bool) {
	var startLabel byte
	if len(labels) > 0 {
		startLabel = labels[0]
	}

	base := a.head
	if h := a.hint[startLabel]; h != 0 && !a.isUsed(h) {
		base = h
	}

	for {
		for ; base != 0; base = a.next[base] {
			if encodable(index^base) && a.fits(base, labels) {
				a.commit(base, labels)
				a.hint[startLabel] = base
				return base, true
			}
		}
		if a.cap() >= maxSlotAllocatorCap {
			return 0, false
		}
		newCap := a.cap() * 2
		if newCap > maxSlotAllocatorCap {
			newCap = maxSlotAllocatorCap
		}
		a.grow(newCap)
		base = a.head
	}
}
