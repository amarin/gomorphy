package grammemes

// SetCollectionColumn groups SetCollectionID lists by list length.
type SetCollectionColumn []SetCollection

// Find returns SetCollectionID of specified SetCollection in SetCollectionColumn.
// If no such collection found, returns false found indicator.
func (setCollections SetCollectionColumn) Find(collection SetCollection) (ID SetCollectionID, found bool) {
	collection.Sort()
	for id, existedCollection := range setCollections {
		if existedCollection.EqualTo(collection) {
			return SetCollectionID(id), true
		}
	}

	return 0, false
}

// Index returns 0-based index of set in GrammemesSet array.
// Returns index of existed or appended item.
func (setCollections *SetCollectionColumn) Index(collection SetCollection) (id SetCollectionID) {
	var found bool

	if len(collection) == 0 {
		panic("empty collection")
	}

	if id, found = setCollections.Find(collection); found {
		return id
	}

	id = SetCollectionID(len(*setCollections))

	*setCollections = append(*setCollections, collection)

	return id
}

// Get returns SetCollection by SetCollectionID or false found indicator if no such SetCollectionID present.
func (setCollections SetCollectionColumn) Get(itemIdx SetCollectionID) (collection SetCollection, found bool) {
	if int(itemIdx) >= len(setCollections) {
		return nil, false
	}

	return setCollections[itemIdx], true
}
