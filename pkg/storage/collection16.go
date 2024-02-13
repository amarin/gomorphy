package storage //nolint:dupl

// Collection16 stores unique ids.Set16 items organized in Table16 storages where each storage
// keeps sets of same sizes.
// Each Collection16 item is addressable by unique ids.ID32
// which consists of Table16 ID16 item index in collection and ID16 element index in Table16 item.
type Collection16 []Table16

// Find returns 0-based index of ids.Set16 item in Collection16 array. If no such set found returns -1.
func (collection8x8 *Collection16) Find(item Set16) (foundIdx ID32, found bool) {
	var itemIdx ID16

	columnIdx := len(item) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if itemIdx, found = (*collection8x8)[columnIdx].Find(item); !found {
		return 0, false
	}

	return Combine16(ID16(columnIdx), itemIdx), true
}

// Index returns 0-based index of set in Collection8x8 array.
func (collection8x8 *Collection16) Index(item Set16) (indexedIdx ID32) {
	if len(item) == 0 {
		panic("empty set")
	}

	columnIdx := len(item) - 1
	if columnIdx >= len(*collection8x8) { // no enough columns extend inplace
		currentLen := len(*collection8x8)

		for i := currentLen; i <= columnIdx; i++ {
			*collection8x8 = append(*collection8x8, make(Table16, 0))
		}
	}

	itemIdx := (*collection8x8)[columnIdx].Index(item)

	return Combine16(ID16(columnIdx), itemIdx)
}

// Get returns set by index.
func (collection8x8 *Collection16) Get(itemIdx ID32) (Set16, bool) {
	if int(itemIdx.Upper16()) >= len(*collection8x8) {
		return nil, false
	}

	return (*collection8x8)[itemIdx.Upper16()].Get(itemIdx.Lower16())
}
