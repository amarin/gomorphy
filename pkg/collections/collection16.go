package collections // nolint:dupl

import (
	"github.com/amarin/gomorphy/pkg/ids"
	"github.com/amarin/gomorphy/pkg/sets"
	"github.com/amarin/gomorphy/pkg/tables"
)

// Collection16 stores unique ids.Set16 items organized in Table16 storages where each storage
// keeps sets of same sizes.
// Each Collection16 item is addressable by unique ids.ID32
// which consists of Table16 ID16 item index in collection and ID16 element index in Table16 item.
type Collection16 []tables.Table16

// Find returns 0-based index of ids.Set16 item in Collection16 array. If no such set found returns -1.
func (collection8x8 Collection16) Find(item sets.Set16) (foundIdx ids.ID32, found bool) {
	var itemIdx ids.ID16

	columnIdx := len(item) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if itemIdx, found = collection8x8[columnIdx].Find(item); !found {
		return 0, false
	}

	return ids.Combine16(ids.ID16(columnIdx), itemIdx), true
}

// Index returns 0-based index of set in Collection8x8 array.
func (collection8x8 *Collection16) Index(item sets.Set16) (indexedIdx ids.ID32) {
	if len(item) == 0 {
		panic("empty set")
	}

	columnIdx := len(item) - 1
	if columnIdx >= len(*collection8x8) { // no enough columns extend inplace
		currentLen := len(*collection8x8)

		for i := currentLen; i <= columnIdx; i++ {
			*collection8x8 = append(*collection8x8, make(tables.Table16, 0))
		}
	}

	itemIdx := (*collection8x8)[columnIdx].Index(item)

	return ids.Combine16(ids.ID16(columnIdx), itemIdx)
}

// Get returns set by index.
func (collection8x8 Collection16) Get(itemIdx ids.ID32) (sets.Set16, bool) {
	if int(itemIdx.Upper16()) >= len(collection8x8) {
		return nil, false
	}

	return collection8x8[itemIdx.Upper16()].Get(itemIdx.Lower16())
}
