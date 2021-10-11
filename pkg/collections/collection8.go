package collections

import (
	"github.com/amarin/gomorphy/pkg/ids"
	"github.com/amarin/gomorphy/pkg/sets"
	"github.com/amarin/gomorphy/pkg/tables"
)

// Collection8x8 stores unique ids.Set8 items in Table8 items where each Table8 stores sets of same sizes.
// Each Collection8x8 item is addressable by unique ID16
// which consists of Table8 ID8 item index in collection and ID8 element index in Table8 item.
type Collection8x8 []tables.Table8

// Find returns 0-based index of ids.Set8 item in Collection8x8 array. If no such set found returns -1.
func (collection8x8 Collection8x8) Find(setItem sets.Set8) (foundIdx ids.ID16, found bool) {
	var itemIdx ids.ID8

	columnIdx := len(setItem) - 1

	if columnIdx < 0 {
		return 0, false // always return not found for empty sets
	}

	if itemIdx, found = collection8x8[columnIdx].Find(setItem); !found {
		return 0, false
	}

	return ids.Combine8(ids.ID8(columnIdx), itemIdx), true
}

// Index returns 0-based index of set in Collection8x8 array.
func (collection8x8 *Collection8x8) Index(setItem sets.Set8) (indexedIdx ids.ID16) {
	if len(setItem) == 0 {
		panic("empty set")
	}

	columnIdx := len(setItem) - 1
	if columnIdx >= len(*collection8x8) { // no enough columns extend inplace
		currentLen := len(*collection8x8)

		for i := currentLen; i <= columnIdx; i++ {
			*collection8x8 = append(*collection8x8, make(tables.Table8, 0))
		}
	}

	itemIdx := (*collection8x8)[columnIdx].Index(setItem)

	return ids.Combine8(ids.ID8(columnIdx), itemIdx)
}

// Get returns set by index.
func (collection8x8 Collection8x8) Get(itemIdx ids.ID16) (sets.Set8, bool) {
	if int(itemIdx.Upper()) >= len(collection8x8) {
		return nil, false
	}

	return collection8x8[itemIdx.Upper()].Get(itemIdx.Lower())
}
