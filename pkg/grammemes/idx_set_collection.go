package grammemes

// SetCollectionIdx provides index if SetID lists to represent different meanings if encountered in words.
type SetCollectionIdx []SetCollectionColumn

// Find returns 0-based index of set in SetIdx array. If no such set found returns -1.
// NOTE: result is int32 to represent -1 in not found case but actual index is uint16
// which occupies positive part of int32 range.
func (idx SetCollectionIdx) Find(setItem SetCollection) (foundIdx SetCollectionID, found bool) {
	var itemIdx SetCollectionID

	columnIdx := len(setItem) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if itemIdx, found = idx[columnIdx].Find(setItem); !found {
		return 0, false
	}

	return SetCollectionID(columnIdx)<<16 | itemIdx, true
}

// Index returns 0-based index of SetCollection in SetIdx array.
// If no such collection present before it silently added.
func (idx *SetCollectionIdx) Index(setItem SetCollection) (indexedIdx SetCollectionID) {
	if len(setItem) == 0 {
		panic("empty set")
	}

	columnIdx := len(setItem) - 1
	if columnIdx >= len(*idx) { // no enough columns extend inplace
		currentLen := len(*idx)

		for i := currentLen; i <= columnIdx; i++ {
			*idx = append(*idx, make(SetCollectionColumn, 0))
		}
	}

	itemIdx := (*idx)[columnIdx].Index(setItem)

	return SetCollectionID(columnIdx)<<16 | itemIdx
}

// Get returns SetCollection by SetCollectionID.
func (idx SetCollectionIdx) Get(itemIdx SetCollectionID) (SetCollection, bool) {
	columnIdx := itemIdx >> 16 //nolint:gomnd
	if int(columnIdx) >= len(idx) {
		return nil, false
	}

	return idx[columnIdx].Get(itemIdx & 0x0000ffff) //nolint:gomnd
}
