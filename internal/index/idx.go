package index

import (
	"fmt"
	"sort"
	"sync"

	"git.media-tel.ru/railgo/logging"
	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/dag"
)

const (
	binaryTagsPrefix     = "TD"
	binaryColIdxPrefix   = "CD"
	binaryItemsIdxPrefix = "ID"
)

// Index implements main dictionary index.
type Index struct {
	mu            *sync.Mutex          // protect internals below
	tags          dag.Idx              // Tag's storage
	tagSets       TagSetIndex          // TagSet's storage
	collectionIdx VariantsIndex        // TagSetIDCollection storage
	items         Items                // Items storage
	childrenMap   map[dag.ID]dag.IdMap // children maps
	wordsCount    int
}

// Tags returns internal tags index.
func (index Index) Tags() dag.Idx {
	return index.tags
}

// New creates new empty list.
func New() *Index {
	return &Index{
		mu:            new(sync.Mutex),
		items:         *NewItems(),
		tags:          dag.NewIndex(),
		tagSets:       make(TagSetIndex, 0),
		collectionIdx: make(VariantsIndex, 0),
		childrenMap:   make(map[dag.ID]dag.IdMap),
		wordsCount:    0,
	}
}

func (index *Index) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	index.mu.Lock()
	defer index.mu.Unlock()

	if err = index.writeTagsDefinitions(writer); err != nil {
		return err
	}
	if err = index.writeTagSetsDefinitions(writer); err != nil {
		return err
	}
	if err = index.writeCollectionsDefinitions(writer); err != nil {
		return err
	}
	if err = index.writeItemsDefinitions(writer); err != nil {
		return err
	}

	return nil
}

// writeTagsDefinitions writes tags index into specified binutils.BinaryWriter.
// A companion of readTagsDefinitions.
// Used from BinaryWriteTo.
func (index *Index) writeTagsDefinitions(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteStringZ(binaryTagsPrefix); err != nil {
		return fmt.Errorf("%w: write: tags prefix: %v", Error, err)
	}
	if err = index.tags.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags index: %v", Error, err)
	}

	return nil
}

// readTagsDefinitions reads tags index from specified binutils.BinaryReader.
// A companion of writeTagsDefinitions.
// Used from BinaryReadFrom.
func (index *Index) readTagsDefinitions(reader *binutils.BinaryReader) (err error) {
	var section string

	if section, err = reader.ReadStringZ(); err != nil {
		return fmt.Errorf("%w: read: tags prefix: %v", Error, err)
	}
	if section != binaryTagsPrefix {
		return fmt.Errorf("%w: read: expected section %v ", Error, binaryTagsPrefix)
	}

	if err = index.tags.BinaryReadFrom(reader); err != nil {
		return fmt.Errorf("%w: read: tags index: %v", Error, err)
	}

	return nil
}

// writeTagsDefinitions writes tags index into specified binutils.BinaryWriter.
// A companion of readTagsDefinitions.
// Used from BinaryWriteTo.
func (index *Index) writeTagSetsDefinitions(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteStringZ(binaryTagSetPrefix); err != nil {
		return fmt.Errorf("%w: write: tags prefix: %v", Error, err)
	}
	if err = index.tagSets.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags sets index: %v", Error, err)
	}

	return nil
}

// readTagsDefinitions reads tags index from specified binutils.BinaryReader.
// A companion of writeTagsDefinitions.
// Used from BinaryReadFrom.
func (index *Index) readTagSetsDefinitions(reader *binutils.BinaryReader) (err error) {
	var section string

	if section, err = reader.ReadStringZ(); err != nil {
		return fmt.Errorf("%w: read: tags prefix: %v", Error, err)
	}
	if section != binaryTagSetPrefix {
		return fmt.Errorf("%w: read: expected section %v ", Error, binaryTagSetPrefix)
	}

	if err = index.tagSets.BinaryReadFrom(reader); err != nil {
		return fmt.Errorf("%w: write: tags sets index: %v", Error, err)
	}

	return nil
}

// writeTagsDefinitions writes tags index into specified binutils.BinaryWriter.
// A companion of readTagsDefinitions.
// Used from BinaryWriteTo.
func (index *Index) writeCollectionsDefinitions(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteStringZ(binaryColIdxPrefix); err != nil {
		return fmt.Errorf("%w: write: tags prefix: %v", Error, err)
	}
	if err = index.collectionIdx.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags set collections index: %v", Error, err)
	}

	return nil
}

// readTagsDefinitions reads tags index from specified binutils.BinaryReader.
// A companion of writeTagsDefinitions.
// Used from BinaryReadFrom.
func (index *Index) readCollectionsDefinitions(reader *binutils.BinaryReader) (err error) {
	var section string

	if section, err = reader.ReadStringZ(); err != nil {
		return fmt.Errorf("%w: read: tags prefix: %v", Error, err)
	}
	if section != binaryColIdxPrefix {
		return fmt.Errorf("%w: read: expected section %v ", Error, binaryColIdxPrefix)
	}

	if err = index.collectionIdx.BinaryReadFrom(reader); err != nil {
		return fmt.Errorf("%w: write: tags set collections index: %v", Error, err)
	}

	return nil
}

// writeTagsDefinitions writes tags index into specified binutils.BinaryWriter.
// A companion of readTagsDefinitions.
// Used from BinaryWriteTo.
func (index *Index) writeItemsDefinitions(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteStringZ(binaryItemsIdxPrefix); err != nil {
		return fmt.Errorf("%w: write: tags prefix: %v", Error, err)
	}
	if err = index.items.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags set collections index: %v", Error, err)
	}

	return nil
}

// readTagsDefinitions reads tags index from specified binutils.BinaryReader.
// A companion of writeTagsDefinitions.
// Used from BinaryReadFrom.
func (index *Index) readItemsDefinitions(reader *binutils.BinaryReader) (err error) {
	var section string

	if section, err = reader.ReadStringZ(); err != nil {
		return fmt.Errorf("%w: read: tags prefix: %v", Error, err)
	}
	if section != binaryItemsIdxPrefix {
		return fmt.Errorf("%w: read: expected section %v ", Error, binaryItemsIdxPrefix)
	}

	if err = index.items.BinaryReadFrom(reader); err != nil {
		return fmt.Errorf("%w: read: node index: %v", Error, err)
	}

	return nil
}

func (index *Index) rebuildChildrenIndex() {
	index.GetChildrenIDMap(0)
	for idx, item := range index.items.items {
		nodeID := dag.ID(idx)
		index.GetChildrenIDMap(item.Parent)
		index.GetChildrenIDMap(nodeID)
		index.childrenMap[item.Parent][item.Letter] = nodeID
		if item.Variants != 0 {
			index.wordsCount++
		}
	}
}

// BinaryReadFrom reads index data from specified binutils.BinaryReader.
// Implements binutils.BinaryReaderFrom.
func (index *Index) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	index.mu.Lock()
	defer index.mu.Unlock()

	if err = index.readTagsDefinitions(reader); err != nil {
		return fmt.Errorf("%w: read: tags: %v", Error, err)
	}
	if err = index.readTagSetsDefinitions(reader); err != nil {
		return fmt.Errorf("%w: read: tags: %v", Error, err)
	}
	if err = index.readCollectionsDefinitions(reader); err != nil {
		return fmt.Errorf("%w: read: collections index: %v", Error, err)
	}
	if err = index.readItemsDefinitions(reader); err != nil {
		return err
	}

	index.rebuildChildrenIndex()

	return nil
}

// AddRunes adds runes sequence into container.
// Returns final node filled with node data or error if add caused error.
func (index *Index) AddRunes(runes []rune) (dag.Node, error) {
	return index.AddToNode(0, runes)
}

// AddString adds string word into index. Returns final node or error if add caused error.
func (index *Index) AddString(word string) (node dag.Node, err error) {
	if len(word) == 0 {
		return nil, fmt.Errorf("%w: empty word", Error)
	}

	return index.AddRunes([]rune(word))
}

// FetchRunes lookups runes sequence in container.
// If found returns final node or error if not found.
func (index *Index) FetchRunes(runes []rune) (dag.Node, error) {
	return index.FetchFromItem(0, runes)
}

// FetchString lookups string in container.
// If found returns final node or error if not found.
func (index *Index) FetchString(word string) (dag.Node, error) {
	return index.FetchRunes([]rune(word))
}

// Children returns rootNode runes mapped to its nodes. Implements dag.Index.
func (index *Index) Children() dag.NodeMap {
	return index.GetChildrenMap(0)
}

// getItem returns node by its index or error if no such node found. Implements dag.Index.
func (index *Index) getItem(nodeIdx dag.ID) *Item {
	return index.items.Get(nodeIdx)
}

// Get returns node by its index or error if no such node found. Implements dag.Index.
func (index *Index) Get(nodeIdx dag.ID) (node dag.Node, err error) {
	if nodeIdx >= index.items.NextID() {
		return nil, fmt.Errorf("%w: no such node: %d", Error, nodeIdx)
	}

	return index.GetItem(nodeIdx), nil
}

func (index *Index) rootNode(letter rune) (root dag.Node) {
	var (
		ok     bool
		rootID dag.ID
	)

	index.mu.Lock()

	defer index.mu.Unlock()

	if rootID, ok = index.childrenMap[0][letter]; !ok {
		return nil
	}

	return index.GetItem(rootID)
}

// // AddChild returns new child of specified Node or error.
// func (index *Index) AddChild(node dag.Node, character rune) (*Node, error) {
// 	newItem := index.items.NewChild(node.ID(), character)
// 	log.Printf("idx %p: %v", index, index.Pretty(0, "-"))
//
// 	return index.GetItem(newItem.ID), nil
// }

// GetChildrenIDMap generates children nodes for Node specified by its ID.
func (index *Index) GetChildrenIDMap(id dag.ID) (res dag.IdMap) {
	var ok bool

	if res, ok = index.childrenMap[id]; !ok {
		index.childrenMap[id] = make(dag.IdMap)
	}
	return index.childrenMap[id]
}

func (index *Index) FetchItemFromParent(parentID dag.ID, runes []rune) (*Node, error) {
	var (
		nextItemID      dag.ID
		ok              bool
		node            *Node
		currentParentID = parentID
		currentIndex    = 0
	)

	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: empty runes", Error)
	}

	for {
		firstRune := runes[currentIndex]
		childrenIDMap := index.GetChildrenIDMap(currentParentID)
		if nextItemID, ok = childrenIDMap[firstRune]; !ok {
			node = index.GetItem(currentParentID)
			if node == nil {
				return nil, fmt.Errorf("%w: fetch: no node: `%s[%s]`", Error, string(runes[:currentIndex]), string(firstRune))
			}

			return nil, fmt.Errorf("%w: fetch: not found: `%s[%s]`", Error, node.Word(), string(firstRune))
		}

		if currentIndex == len(runes)-1 {
			return index.GetItem(nextItemID), nil
		}

		currentParentID = nextItemID
		currentIndex += 1
	}
}

func (index *Index) FetchFromItem(parentID dag.ID, runes []rune) (dag.Node, error) {
	var (
		nextItemID      dag.ID
		ok              bool
		currentParentID = parentID
		currentIndex    = 0
	)

	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: empty runes", Error)
	}

	for {
		firstRune := runes[currentIndex]
		childrenIDMap := index.GetChildrenIDMap(currentParentID)
		if nextItemID, ok = childrenIDMap[firstRune]; !ok {
			node := index.GetItem(currentParentID)
			if node == nil {
				return nil, fmt.Errorf("%w: fetch: no node: `%s[%s]`", Error, string(runes[:currentIndex]), string(firstRune))
			}

			return nil, fmt.Errorf("%w: fetch: not found: `%s[%s]`", Error, node.Word(), string(firstRune))
		}

		if currentIndex == len(runes)-1 {
			return index.Get(nextItemID)
		}

		currentParentID = nextItemID
		currentIndex += 1
	}
}

func (index *Index) AddToNode(parentID dag.ID, runes []rune) (dag.Node, error) {
	var (
		nextItemID      dag.ID
		ok              bool
		currentParentID = parentID
		currentIndex    = 0
	)

	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: add: empty runes", Error)
	}

	for {
		if currentParentID != 0 { // prevent adding to not existed parent
			if parent := index.items.Get(currentParentID); parent == nil {
				return nil, fmt.Errorf("%w: add: no parent: %d", Error, parentID)
			}
		}

		rootLetter := runes[currentIndex]
		childrenIDMap := index.GetChildrenIDMap(currentParentID)
		if nextItemID, ok = childrenIDMap[rootLetter]; !ok {
			newRootItem := index.items.NewChild(currentParentID, rootLetter)
			nextItemID = newRootItem.ID
			index.childrenMap[currentParentID][rootLetter] = newRootItem.ID
		}

		if currentIndex == len(runes)-1 {
			return index.Get(nextItemID)
		}

		currentParentID = nextItemID
		currentIndex += 1
	}
}

// WordsCount returns count of indexed words.
func (index *Index) WordsCount() int {
	return index.wordsCount
}

// NodesCount returns count of indexed nodes.
func (index *Index) NodesCount() int {
	return int(index.items.NextID() - 1)
}

// GetItem generates Node instance runtime.
func (index *Index) GetItem(id dag.ID) *Node {
	return newNode(index, id)
}

// GetChildrenMap generates children nodes for Node specified by its ID.
func (index *Index) GetChildrenMap(id dag.ID) dag.NodeMap {
	childrenItems := index.childrenMap[id]

	res := make(dag.NodeMap)
	for letter, chidlID := range childrenItems {
		res[letter] = index.GetItem(chidlID)
	}

	return res
}

// getChildren generates children nodes for Node specified by its ID.
func (index *Index) getChild(id dag.ID, letter rune) *Node {
	childrenItems := index.childrenMap[id]

	if len(childrenItems) == 0 {
		return nil
	}

	return index.GetItem(childrenItems[letter])
}

// TagID gets or creates tag in internal tag index and returns its ID.
func (index *Index) TagID(name dag.TagName, parent dag.TagName) dag.TagID {
	return index.tags.Index(name, parent)
}

func (index *Index) TagSet(tagSet TagSet) (res dag.TagSet, err error) {
	res = make(dag.TagSet, len(tagSet))
	for idx, tagID := range tagSet {
		tag, found := index.tags.Get(tagID)
		if !found {
			return res, fmt.Errorf("no such tagID %d", tagID)
		}
		res[idx] = tag
	}

	return res, nil
}

// TagSetIndex returns internal TagSetIndex.
func (index *Index) TagSetIndex() TagSetIndex {
	return index.tagSets
}

// Optimize reduces index deleting unused tag set's and collections;
func (index *Index) Optimize() {
	logger := logging.NewNamedLogger("optimize").WithLevel(logging.LevelDebug)
	usedCollectionID := make(map[VariantID][]dag.ID)
	knownCollections := index.collectionIdx.KnownID()
	logger.Infof("check %d known collections", len(knownCollections))
	for _, node := range index.items.items {
		if node.Variants == 0 {
			continue
		}
		if _, ok := usedCollectionID[node.Variants]; !ok {
			usedCollectionID[node.Variants] = make([]dag.ID, 0)
		}
		usedCollectionID[node.Variants] = append(usedCollectionID[node.Variants], node.ID)
	}

	logger.Debugf("lookup unused collections")
	unusedCollections := make(CollectionIDList, 0)
	for _, knownCollectionID := range knownCollections {
		if _, ok := usedCollectionID[knownCollectionID]; !ok {
			unusedCollections = append(unusedCollections, knownCollectionID)
		}
	}

	logger.Debugf("eliminate %d unused collections", len(unusedCollections))
	if len(unusedCollections) == 0 {
		logger.Debugf("no unused collections")
		return
	}

	sort.Sort(unusedCollections)

	type replacement struct {
		old VariantID
		new VariantID
	}

	newIndex := make(VariantsIndex, 0)
	replaceCollections := make([]replacement, 0)
	for _, collectionID := range knownCollections {
		if _, ok := usedCollectionID[collectionID]; ok {
			collection := index.collectionIdx.Get(collectionID)
			newCollectionID := newIndex.Index(collection)
			replacementPair := replacement{
				old: collectionID,
				new: newCollectionID,
			}
			newCollection := newIndex.Get(newCollectionID)
			if !collection.EqualTo(newCollection) {
				logger.Errorf("old id %X collection %v", collectionID, collection)
				logger.Errorf("new id %X collection %v", newCollectionID, newCollection)

				panic(fmt.Errorf("replacement differs"))
			}
			replaceCollections = append(replaceCollections, replacementPair)
		}
	}

	logger.Infof("have %d collectionID to replace in items", len(replaceCollections))
	itemsUpdated := 0
	for _, replacementPair := range replaceCollections {
		itemsToUpdate, ok := usedCollectionID[replacementPair.old]
		if !ok {
			panic(fmt.Errorf("no items to update with collectionID %d", replacementPair.old))
		}

		for _, itemID := range itemsToUpdate {
			index.items.items[itemID].Variants = replacementPair.new
			itemsUpdated++
		}
	}
	logger.Infof("%d items VariantID updated", itemsUpdated)
	logger.Infof("reduced collection from %d to %d items", len(index.collectionIdx.KnownID()), len(newIndex.KnownID()))
	index.collectionIdx = newIndex

	knownTagSets := index.tagSets.TableIDs()
	// tagSetStrings := make([]string, len(knownTagSets))
	logger.Infof("has %d possible tag sets", len(knownTagSets))
}

// Variants returns TagSet collection
func (index *Index) Variants(id VariantID) (tagSetCollection TagSetIDCollection, err error) {
	return index.collectionIdx.Get(id), err
}
