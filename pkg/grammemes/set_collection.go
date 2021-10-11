package grammemes

import (
	"sort"
)

// SetCollection groups Set lists together.
// Used by ListIndex to group same-sized Set's lists in plain orthogonal array's.
type SetCollection []SetID

// SetCollectionID represents SetCollection ID in SetCollectionIdx.
type SetCollectionID uint32

// Len returns length of grammemes GrammemesSet. Implements sort.Interface.
func (setCollection SetCollection) Len() int {
	return len(setCollection)
}

// Less reports whether the element with index i must sort before the element with index j.
// Implements sort.Interface.
func (setCollection SetCollection) Less(i, j int) bool {
	return setCollection[i] < setCollection[j]
}

// Swap swaps the elements with indexes i and j.
// Implements sort.Interface.
func (setCollection SetCollection) Swap(i, j int) {
	setCollection[i], setCollection[j] = setCollection[j], setCollection[i]
}

// Sort is a convenience method: x.Sort() calls sort.Sort(x).
func (setCollection SetCollection) Sort() { sort.Sort(setCollection) }

// EqualTo compares grammemes GrammemesSet with another one.
// Returns true if both sets are contains the same grammemes or false otherwise.
func (setCollection SetCollection) EqualTo(another SetCollection) bool {
	if setCollection.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < setCollection.Len(); idx++ {
		if setCollection[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}

// Find returns SetID index in SetCollection. If not present found indicator will equal to false.
func (setCollection SetCollection) Find(setID SetID) (collectionID SetCollectionID, found bool) {
	for idx, existedSetID := range setCollection {
		if existedSetID == setID {
			return SetCollectionID(idx), true
		}
	}

	return 0, false
}

// Index returns 0-based index of SetID in SetCollection.
// Returns index of existed or appended item.
func (setCollection *SetCollection) Index(setID SetID) (id SetCollectionID) {
	var found bool

	if id, found = setCollection.Find(setID); found {
		return id
	}

	id = SetCollectionID(len(*setCollection))
	*setCollection = append(*setCollection, setID)

	return id
}

// Get returns set by index.
func (setCollection SetCollection) Get(itemIdx SetCollectionID) (SetID, bool) {
	if int(itemIdx) >= len(setCollection) {
		return 0, false
	}

	return setCollection[itemIdx], true
}
