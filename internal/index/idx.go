package index

import (
	"fmt"
	"sync"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/dag"
)

// Index implements main dictionary index.
type Index struct {
	mu            *sync.Mutex            // protect internals below
	tags          dag.Idx                // Tag's storage
	tagSets       TagSetIndex            // TagSet's storage
	collectionIdx TableIDCollectionIndex // TableIDCollection storage
	items         Items                  // Items storage
	childrenMap   map[dag.ID]dag.IdMap   // children maps
	wordsCount    int
}

// New creates new empty list.
func New() *Index {
	return &Index{
		mu:            new(sync.Mutex),
		items:         *NewItems(),
		tags:          dag.NewIndex(),
		tagSets:       make(TagSetIndex, 0),
		collectionIdx: make(TableIDCollectionIndex, 0),
		childrenMap:   make(map[dag.ID]dag.IdMap),
		wordsCount:    0,
	}
}

func (index *Index) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	index.mu.Lock()
	defer index.mu.Unlock()

	if err = index.tags.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags index: %v", Error, err)
	}
	if err = index.tagSets.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags sets index: %v", Error, err)
	}
	if err = index.collectionIdx.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags set collections index: %v", Error, err)
	}
	if err = index.items.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: write: tags set collections index: %v", Error, err)
	}
	// if err = index.roots.BinaryWriteTo(writer); err != nil {
	// 	return fmt.Errorf("%w: write: roots index: %v", Error, err)
	// }

	return nil
}

func (index *Index) BinaryReadFrom(reader *binutils.BinaryReader) (n int64, err error) {
	index.mu.Lock()
	defer index.mu.Unlock()

	if _, err = index.tags.BinaryReadFrom(reader); err != nil {
		return 0, fmt.Errorf("%w: write: tags index: %v", Error, err)
	}
	if _, err = index.tagSets.BinaryReadFrom(reader); err != nil {
		return 0, fmt.Errorf("%w: write: tags sets index: %v", Error, err)
	}
	if _, err = index.collectionIdx.BinaryReadFrom(reader); err != nil {
		return 0, fmt.Errorf("%w: write: tags set collections index: %v", Error, err)
	}
	if _, err = index.items.BinaryReadFrom(reader); err != nil {
		return 0, fmt.Errorf("%w: write: tags set collections index: %v", Error, err)
	}

	index.getChildrenIDMap(0)
	for idx, item := range index.items.items {
		nodeID := dag.ID(idx)
		if _, ok := index.childrenMap[item.Parent]; !ok {
			index.childrenMap[item.Parent] = make(dag.IdMap, 16)
		}
		if _, ok := index.childrenMap[nodeID]; !ok {
			index.childrenMap[nodeID] = make(dag.IdMap, 16)
		}
		index.childrenMap[item.Parent][item.Letter] = nodeID
		if item.Variants != 0 {
			index.wordsCount++
		}

	}

	return 0, nil
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
	return index.getChildrenMap(0)
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

	return index.getNode(nodeIdx), nil
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

	return index.getNode(rootID)
}

// // AddChild returns new child of specified Node or error.
// func (index *Index) AddChild(node dag.Node, character rune) (*Node, error) {
// 	newItem := index.items.NewChild(node.ID(), character)
// 	log.Printf("idx %p: %v", index, index.Pretty(0, "-"))
//
// 	return index.getNode(newItem.ID), nil
// }

// getChildrenIDMap generates children nodes for Node specified by its ID.
func (index *Index) getChildrenIDMap(id dag.ID) (res dag.IdMap) {
	var ok bool

	if res, ok = index.childrenMap[id]; !ok {
		index.childrenMap[id] = make(dag.IdMap)
	}
	return index.childrenMap[id]
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
		childrenIDMap := index.getChildrenIDMap(currentParentID)
		if nextItemID, ok = childrenIDMap[firstRune]; !ok {
			node := index.getNode(currentParentID)
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
		childrenIDMap := index.getChildrenIDMap(currentParentID)
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

// getNode generates Node instance runtime.
func (index *Index) getNode(id dag.ID) *Node {
	return newNode(index, id)
}

// getChildren generates children nodes for Node specified by its ID.
func (index *Index) getChildrenMap(id dag.ID) dag.NodeMap {
	childrenItems := index.childrenMap[id]

	res := make(dag.NodeMap)
	for letter, chidlID := range childrenItems {
		res[letter] = index.getNode(chidlID)
	}

	return res
}

// getChildren generates children nodes for Node specified by its ID.
func (index *Index) getChild(id dag.ID, letter rune) *Node {
	childrenItems := index.childrenMap[id]

	if len(childrenItems) == 0 {
		return nil
	}

	return index.getNode(childrenItems[letter])
}

// TagID gets or creates tag in internal tag index and returns its ID.
func (index *Index) TagID(name dag.TagName, parent dag.TagName) dag.TagID {
	return index.tags.Index(name, parent)
}
