package grammemes

// SetIdx organizes sets into Column list where each column stores sets with same sizes.
// Grammemes Set in SetIdx consists of 2 components:
// uint8 column ID of uint8 type and uint8 Set ID in specified column.
type SetIdx []Column

// SetID represents Grammeme Set ID.
type SetID uint16

// Find returns 0-based index of set in SetIdx array. If no such set found returns -1.
// NOTE: result is int32 to represent -1 in not found case but actual index is uint16
// which occupies positive part of int32 range.
func (idx SetIdx) Find(setItem Set) (foundIdx SetID, found bool) {
	var itemIdx uint8

	columnIdx := len(setItem) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if itemIdx, found = idx[columnIdx].Find(setItem); !found {
		return 0, false
	}

	return SetID(columnIdx)<<8 | SetID(itemIdx), true
}

// Index returns 0-based index of set in SetIdx array.
func (idx *SetIdx) Index(setItem Set) (indexedIdx SetID) {
	if len(setItem) == 0 {
		panic("empty set")
	}

	columnIdx := len(setItem) - 1
	if columnIdx >= len(*idx) { // no enough columns extend inplace
		currentLen := len(*idx)

		for i := currentLen; i <= columnIdx; i++ {
			*idx = append(*idx, make(Column, 0))
		}
	}

	itemIdx := (*idx)[columnIdx].Index(setItem)

	return SetID(columnIdx)<<8 | SetID(itemIdx)
}

// Get returns set by index.
func (idx SetIdx) Get(itemIdx SetID) (Set, bool) {
	columnIdx := itemIdx >> 8 //nolint:gomnd
	if int(columnIdx) >= len(idx) {
		return nil, false
	}

	return idx[columnIdx].Get(uint8(itemIdx & 0xff)) //nolint:gomnd
}
