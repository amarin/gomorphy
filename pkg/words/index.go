package words

import (
	"fmt"

	"github.com/amarin/binutils"
	"golang.org/x/text/encoding/charmap"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/common"
)

var (
	binaryCharmap = charmap.KOI8R // nolint:gochecknoglobals
)

// Index stores all added words with attached grammemes sets.
type Index struct {
	grammemesIndex *grammemes.Index
	container      *NodesContainer
	runesIndex     map[rune]*NodeList
}

// NewIndex creates new words index.
func NewIndex(grammemesIndex *grammemes.Index) *Index {
	return &Index{
		container:      NewNodeContainer(nil),
		grammemesIndex: grammemesIndex,
		runesIndex:     make(map[rune]*NodeList),
	}
}

// GrammemesIndex returns used grammemes index. Implements GrammemesIndexer.
func (index Index) GrammemesIndex() *grammemes.Index {
	return index.grammemesIndex
}

// AddWord adds word with its grammemes to index.
func (index *Index) AddWord(form *Word) error {
	addedNodes, err := index.container.AddWord(form)
	for _, node := range addedNodes {
		index.addNodeToRunesIndex(node)
	}

	return err
}

// addNodeToRunesIndex adds node into internal per-rune index.
func (index *Index) addNodeToRunesIndex(node *Node) {
	var nodesList *NodeList

	var ok bool
	if nodesList, ok = index.runesIndex[node.Rune()]; !ok {
		nodesList = NewNodeList()
	}

	*nodesList = append(*nodesList, node)
}

// SearchForms returns all known grammeme set for requested word.
// If nothing found, return empty set.
func (index Index) SearchForms(word text.RussianText) grammemes.ListList {
	return index.container.SearchForms(word)
}

// Container returns internal index nodes container.
// Assume you know what you do if you use it.
func (index Index) Container() *NodesContainer {
	return index.container
}

// Slice returns flat list of all index nodes. List sorted alphabetically and by length of words.
// Used during binary marshaling.
func (index *Index) Slice() NodeList {
	return index.container.Slice()
}

// Len returns length of direct children container.
func (index *Index) Len() int {
	return index.container.Len()
}

// MarshalBinary returns index binary representation.
// Implements BinaryMarshaler.
func (index *Index) MarshalBinary() (data []byte, err error) {
	slice := index.Slice()
	return slice.MarshalBinary()
}

// UnmarshalFromBuffer восстанавливает данные индекса из двоичного буфера.
func (index *Index) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
	nodeList := NewNodeList()
	if err := nodeList.UnmarshalFromBufferWithIndex(buffer, index.grammemesIndex); err != nil {
		return fmt.Errorf("%w: node list: %v", common.ErrUnmarshal, err)
	}
	// attach root nodes to root container
	for _, node := range *nodeList {
		if node.parent == nil {
			index.container.children[node.Rune()] = node
		}
	}

	return nil
}
