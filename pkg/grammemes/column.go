package grammemes

// Column groups together a list of Set's having equal length.
// Used in SetIdx to store different sized sets organized into stacks of equal-sized sets.
type Column []Set

// Find returns 0-based index of set in GrammemesSet array. If no such set found returns -1.
// It expected GrammemesSet to find to be Sorted before.
func (idx Column) Find(setItem Set) (SetID, bool) {
	for id, existedSet := range idx {
		if existedSet.EqualTo(setItem) {
			return SetID(id), true
		}
	}

	return 0, false
}

// Index returns 0-based index of set in GrammemesSet array.
// Returns index of existed or appended item.
func (idx *Column) Index(setItem Set) (id SetID) {
	var found bool

	if len(setItem) == 0 {
		panic("empty set")
	}

	if id, found = idx.Find(setItem); found {
		return id
	}

	id = SetID(len(*idx))

	*idx = append(*idx, setItem)

	return id
}

// Get returns set by index.
func (idx Column) Get(itemIdx SetID) (Set, bool) {
	if int(itemIdx) >= len(idx) {
		return nil, false
	}

	return idx[itemIdx], true
}
