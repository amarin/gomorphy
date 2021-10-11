package grammemes

// Column groups together a list of Set's having equal length.
// Used in SetIdx to store different sized sets organized into stacks of equal-sized sets.
type Column []Set

// Find returns 0-based index of set in GrammemesSet array. If no such set found returns -1.
// It expected GrammemesSet to find to be Sorted before.
func (idx Column) Find(setItem Set) (uint8, bool) {
	for id, existedSet := range idx {
		if existedSet.EqualTo(setItem) {
			return uint8(id), true
		}
	}

	return 0, false
}

// Index returns 0-based index of set in GrammemesSet array.
// Returns index of existed or appended item.
func (idx *Column) Index(setItem Set) (id uint8) {
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
func (idx Column) Get(itemIdx uint8) (Set, bool) {
	if int(itemIdx) >= len(idx) {
		return nil, false
	}

	return idx[itemIdx], true
}
