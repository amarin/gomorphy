package storage

// Table8 stores upto 256 unique ids.Set8 lists having equal length.
// Used in Collection8x8 to store different sized sets organized into stacks of equal-sized sets.
type Table8 []Set8

// Find returns 0-based index of ids.Set8 in Table8 storage. If no such set found returns -1.
// It required argument sets.Set8 to be sorted before.
func (table8 Table8) Find(item Set8) (ID8, bool) {
	for id, existedSet := range table8 {
		if existedSet.EqualTo(item) {
			return ID8(id), true
		}
	}

	return 0, false
}

// Index returns 0-based ids.ID8 index of ids.Set8 in Table8.
// Returns index of existed or appended item.
// Panics if specified set empty.
func (table8 *Table8) Index(item Set8) (id ID8) {
	var found bool

	if len(item) == 0 {
		panic("empty set")
	}

	item.Sort() // sort item before find or adding to index.

	if id, found = table8.Find(item); found {
		return id
	}

	id = ID8(len(*table8))

	*table8 = append(*table8, item)

	return id
}

// Get returns ids.Set8 by index if present or found indicator will be false.
func (table8 Table8) Get(itemIdx ID8) (targetSet Set8, found bool) {
	if int(itemIdx) >= len(table8) {
		return nil, false
	}

	return table8[itemIdx], true
}
