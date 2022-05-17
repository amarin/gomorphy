package index

import (
	"fmt"

	"github.com/amarin/gomorphy/pkg/dag"
)

// Node represents index node.
type Node struct {
	index *Index
	id    dag.ID
}

func (node *Node) Id() dag.ID {
	return node.id
}

func (node Node) Item() Item {
	return node.index.items.items[node.id]
}

// TagSets returns list of dag.TagSet. Implements dag.Node.
func (node *Node) TagSets() (res []dag.TagSet) {
	var (
		found      bool
		item       *Item
		collection TagSetIDCollection
		tagSetIDs  TagSet
		tag        dag.Tag
	)

	item = node.index.getItem(node.id)
	collection = node.index.collectionIdx.Get(item.Variants)
	res = make([]dag.TagSet, collection.Len())

	for idx, tableID := range collection {
		if tagSetIDs, found = node.index.tagSets.Get(tableID); !found {
			return res
		}
		res[idx] = make(dag.TagSet, tagSetIDs.Len())
		for tagIdx, tagID := range tagSetIDs {
			if tag, found = node.index.tags.Get(tagID); found {
				res[idx][tagIdx] = tag
			}
		}
	}

	return res
}

func (node *Node) AddTagSet(newTagSet ...dag.TagName) error {
	var found bool
	tagSet := make(TagSet, len(newTagSet))
	for idx, tagName := range newTagSet {
		tagSet[idx], found = node.index.tags.Find(tagName)
		if !found {
			return fmt.Errorf("%w: add tag set: unknown tag: %v", Error, tagName)
		}
	}

	item := node.index.getItem(node.id)
	collection := append(node.index.collectionIdx.Get(item.Variants), node.index.tagSets.Index(tagSet))
	if item.Variants == 0 {
		node.index.wordsCount++
	}
	item.Variants = node.index.collectionIdx.Index(collection)

	return nil
}

// Word returns sequence of characters from root upto current node wrapped into string.
func (node *Node) Word() string {
	prefix := ""
	if node.id == 0 {
		return ""
	}
	item := node.index.getItem(node.id)

	if item == nil {
		return fmt.Sprintf("no item: %v", node.id)
	}

	if item.Parent != 0 {
		prefix = node.index.GetItem(item.Parent).Word()
	}

	return prefix + string(item.Letter)
}

// String returns string representation of node. Implements fmt.Stringer.
func (node *Node) String() string {
	item := node.index.getItem(node.id)
	w := node.Word()

	return "Node(" + w[:len(w)-1] + "[" + string(item.Letter) + "]" + item.String()[1:] + ")"
}

func newNode(index *Index, id dag.ID) *Node {
	return &Node{
		index: index,
		id:    id,
	}
}
