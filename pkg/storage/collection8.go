package storage

// Collection8x8 stores unique ids.Set8 items in Table8 items where each Table8 stores sets of same sizes.
// Each Collection8x8 item is addressable by unique ID16
// which consists of Table8 ID8 item index in collection and ID8 element index in Table8 item.
type Collection8x8 []Table8

// Find returns 0-based index of ids.Set8 item in Collection8x8 array. If no such set found returns -1.
func (collection8x8 Collection8x8) Find(setItem Set8) (foundIdx ID16, found bool) {
	var itemIdx ID8

	columnIdx := len(setItem) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if itemIdx, found = collection8x8[columnIdx].Find(setItem); !found {
		return 0, false
	}

	return Combine8(ID8(columnIdx), itemIdx), true
}

// Index returns 0-based index of set in Collection8x8 array.
func (collection8x8 *Collection8x8) Index(setItem Set8) (indexedIdx ID16) {
	if len(setItem) == 0 {
		panic("empty set")
	}

	columnIdx := len(setItem) - 1
	if columnIdx >= len(*collection8x8) { // no enough columns extend inplace
		currentLen := len(*collection8x8)

		for i := currentLen; i <= columnIdx; i++ {
			*collection8x8 = append(*collection8x8, make(Table8, 0))
		}
	}

	itemIdx := (*collection8x8)[columnIdx].Index(setItem)

	return Combine8(ID8(columnIdx), itemIdx)
}

// Get returns set by index.
func (collection8x8 Collection8x8) Get(itemIdx ID16) (Set8, bool) {
	if int(itemIdx.Upper()) >= len(collection8x8) {
		return nil, false
	}

	return collection8x8[itemIdx.Upper()].Get(itemIdx.Lower())
}
