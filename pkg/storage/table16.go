package storage

// Table16 stores unique ids.Set16 lists having equal length.
// It Uses its own ID16 index to address target sets.set16 stack in addition to address element in stack.
// Used in Collection16 to store different sized sets organized into stacks of equal-sized sets.
type Table16 []Set16

// Find returns 0-based index of ids.Set16 in Table16 storage. If no such set found returns -1.
// It required sets.Set16 argument to be sorted before.
func (table16 Table16) Find(item Set16) (ID16, bool) {
	for id, existedSet := range table16 {
		if existedSet.EqualTo(item) {
			return ID16(id), true
		}
	}

	return 0, false
}

// Index returns 0-based ids.ID16 index of ids.Set16 in Table16 instance.
// Returns index of existed or appended item.
// Panics if specified set empty.
func (table16 *Table16) Index(item Set16) (id ID16) {
	var found bool

	if len(item) == 0 {
		panic("empty set")
	}

	item.Sort() // sort item before find or adding to index.

	if id, found = table16.Find(item); found {
		return id
	}

	id = ID16(len(*table16))

	*table16 = append(*table16, item)

	return id
}

// Get returns ids.Set16 by index if present or found indicator will be false.
func (table16 Table16) Get(id ID16) (targetSet Set16, found bool) {
	if int(id) >= len(table16) {
		return nil, false
	}

	return table16[id], true
}
