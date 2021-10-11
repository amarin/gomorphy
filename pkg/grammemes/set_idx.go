package grammemes

type SetColumn []Set

// Find returns 0-based index of set in GrammemesSet array. If no such set found returns -1.
// It expected GrammemesSet to find to be Sorted before.
func (idx SetColumn) Find(setItem Set) (uint8, bool) {
	for id, existedSet := range idx {
		if existedSet.EqualTo(setItem) {
			return uint8(id), true
		}
	}

	return 0, false
}

// GrammemeIdx returns 0-based index of set in GrammemesSet array.
// Returns index of existed or appended item.
func (idx *SetColumn) Index(setItem Set) (id uint8) {
	var found bool

	if len(setItem) == 0 {
		panic("empty set")
	}

	if id, found = idx.Find(setItem); found {
		return id
	}

	id = uint8(len(*idx))

	*idx = append(*idx, setItem)

	return id
}

// Get returns set by index.
func (idx SetColumn) Get(itemIdx uint8) (Set, bool) {
	if int(itemIdx) >= len(idx) {
		return nil, false
	}

	return idx[itemIdx], true
}

// SetIdx organizes sets into SetColumn list where each column stores sets with same sizes.
// GrammemeIdx in SetIdx consists of 2 components: uint8 column determinant of uint8 and uint8 GrammemesSet index in specified column.
type SetIdx []SetColumn

// Find returns 0-based index of set in SetIdx array. If no such set found returns -1.
// NOTE: result is int32 to represent -1 in not found case but actual index is uint16
// which occupies positive part of int32 range.
func (idx SetIdx) Find(setItem Set) (foundIdx uint16, found bool) {
	var itemIdx uint8

	columnIdx := len(setItem) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if itemIdx, found = idx[columnIdx].Find(setItem); !found {
		return 0, false
	}

	return uint16(columnIdx)<<8 | uint16(itemIdx), true
}

// GrammemeIdx returns 0-based index of set in SetIdx array. If no such set found returns -1.
// NOTE: result is int32 to represent -1 in not found case but actual index is uint16
// which occupies positive part of int32 range.
func (idx *SetIdx) Index(setItem Set) (indexedIdx uint16) {
	if len(setItem) == 0 {
		panic("empty set")
	}

	columnIdx := len(setItem) - 1
	if columnIdx >= len(*idx) { // no enough columns extend inplace
		currentLen := len(*idx)

		for i := currentLen; i <= columnIdx; i++ {
			*idx = append(*idx, make(SetColumn, 0))
		}
	}

	itemIdx := (*idx)[columnIdx].Index(setItem)

	return uint16(columnIdx)<<8 | uint16(itemIdx)
}

// Get returns set by index.
func (idx SetIdx) Get(itemIdx uint16) (Set, bool) {
	columnIdx := itemIdx >> 8 //nolint:gomnd
	if int(columnIdx) >= len(idx) {
		return nil, false
	}

	return idx[columnIdx].Get(uint8(itemIdx & 255)) //nolint:gomnd
}
