package words

import (
	"fmt"
	"strings"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/common"
)

// NodesContainer stores index nodes.
type NodesContainer struct {
	parent   *Node
	children map[rune]*Node
}

// NewNodeContainer creates new container of parent node if set.
func NewNodeContainer(parent *Node) *NodesContainer {
	return &NodesContainer{parent: parent, children: make(map[rune]*Node, 32)}
}

// String returns string representation of a container. Implements Stringer.
func (container NodesContainer) String() string {
	runes := make([]rune, len(container.children))
	usedIdx := 0
	for _, char := range strings.Split(RussianAlphabetLower, "") {
		r := []rune(char)[0]
		if container.HasChild(r) {
			runes[usedIdx] = r
			usedIdx++
		}
	}

	if container.parent != nil {
		return container.parent.String() + "->" + "С{" + string(runes) + "}"
	}

	return "-->С{" + string(runes) + "}"
}

// GoString returns string representation of a container. Implements GoStringer.
func (container NodesContainer) GoString() string {
	str := make([]string, 0)

	for _, char := range strings.Split(RussianAlphabetLower, "") {
		r := []rune(char)[0]
		if child, err := container.FindChild(r); err == nil {
			str = append(str, "\n"+child.GoString())
		}
	}

	return fmt.Sprintf(
		"C{%#p,%v}",
		container.parent, strings.Join(str, ","))
}

// Len returns container length. Only direct children ara counted.
func (container NodesContainer) Len() int {
	return len(container.children)
}

// FindChild returns node having required index rune or error if no such node found.
func (container NodesContainer) FindChild(char rune) (*Node, error) {
	if node, ok := container.children[char]; ok {
		return node, nil
	}

	return nil, fmt.Errorf("%w: `%v`", common.ErrUnknownNode, char)
}

// HasChild returns true if container has direct child node with required rune and false otherwise.
func (container NodesContainer) HasChild(character rune) bool {
	if _, err := container.FindChild(character); err != nil {
		return false
	}

	return true
}

// Child returns existed before or new created node with required rune.
func (container *NodesContainer) Child(char rune) *Node {
	if _, ok := container.children[char]; !ok {
		container.children[char] = NewMappingNode(container.parent, char)
	}

	return container.children[char]
}

// SearchForms returns all known grammemes set by request word. If nothing found returns empty list.
func (container NodesContainer) SearchForms(word text.RussianText) grammemes.ListList {
	if len(word) == 0 {
		return grammemes.ListList{}
	}

	wordText := strings.ToLower(string(word))            // translate to lower case
	firstChar := []rune(wordText)[0]                     // take first rune to search direct child.
	nextChars := text.RussianText([]rune(wordText)[1:])  // pack together characters from 2 till last
	firstCharNode, err := container.FindChild(firstChar) // take first char index node

	switch {
	case err != nil:
		// no node with required character
		return grammemes.ListList{}
	case len(nextChars) == 0:
		// final rune, return all grammemes set from current node.
		return firstCharNode.grammemes
	default:
		// not a final node, recursively load grammemes by runes reminder.
		return firstCharNode.Children().SearchForms(nextChars)
	}
}

// Slice returns flat list of all known nodes in container including all descendants nodes of direct children.
// Nodes is sorted by alphabet on all layers.
func (container NodesContainer) Slice() NodeList {
	nodes := make(NodeList, 0)

	for _, char := range strings.Split(RussianAlphabetLower, "") {
		r := []rune(char)[0]
		if node, ok := container.children[r]; ok {
			nodes = append(nodes, node.Slice()...)
		}
	}

	return nodes
}

// Children returns direct children as flat NodeList.
func (container NodesContainer) Children() NodeList {
	return container.Slice()
}

// AddWord adds word ending to container nodes.
// Creates required nodes recursively.
// If added word contains empty ending, stores word grammemes in current node.
// Returns added nodes list and possible error.
func (container *NodesContainer) AddWord(form *Word) ([]*Node, error) {
	wordText := strings.ToLower(string(form.Text()))

	if len(wordText) == 0 {
		return []*Node{}, fmt.Errorf("%w: cant ad empty container", common.ErrEmptyValue)
	}

	child := container.Child([]rune(wordText)[0])

	reducedForm := NewWord(form.GrammemesIndex(), text.RussianText([]rune(wordText)[1:]), form.grammemes.Slice()...)

	if reducedForm.Text().Len() == 0 {
		return []*Node{child}, child.AddGrammemes(form.Grammemes())
	}

	addedByChild, err := child.Children().AddWord(reducedForm)
	if err != nil {
		return []*Node{}, fmt.Errorf("%w: cant add: %v", common.ErrChildrenError, err)
	}

	addedNodes := make([]*Node, len(addedByChild)+1)
	addedNodes[0] = child
	copy(addedNodes[1:], addedByChild)

	return addedNodes, nil
}

// Endings returns final node list starting from nodes of current container.
func (container *NodesContainer) EndingsNodes() *NodeList {
	endings := NewNodeList()

	for _, node := range container.children {
		nodeEndings := node.children.EndingsNodes()
		*endings = append(*endings, *nodeEndings...)
	}

	return endings
}
