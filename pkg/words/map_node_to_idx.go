package words

import (
	"fmt"
	"sort"

	"github.com/amarin/gomorphy/pkg/common"
)

// NodePointersMap holds mapping of node pointer to its index in node slice.
// Used to marshal NodeSlice into binary representation.
type NodePointersMap map[*Node]uint64

// PointersStrings returns mapping string representation set to debug.
func (mapping NodePointersMap) PointersStrings() []string {
	mapRepresentation := make([]string, 0)
	for nodePtr, nodeIdx := range mapping {
		mapRepresentation = append(
			mapRepresentation,
			fmt.Sprintf(
				"%d:%p %v -> %p",
				nodeIdx, nodePtr, *nodePtr, nodePtr.parent,
			),
		)
	}

	sort.Strings(mapRepresentation)

	return mapRepresentation
}

// Idx returns uint64 node index or error if node not found.
// Used to store indexes of parents when marshaling NodeList.
func (mapping *NodePointersMap) Idx(node *Node) (uint64, error) {
	mappingIndirect := *mapping
	idx, ok := mappingIndirect[node]

	if !ok {
		return 0, fmt.Errorf("%w: %p", common.ErrUnknownNode, node)
	}

	return idx, nil
}

// Idx returns uint64 node index or error if node not found.
// Used to store indexes of parents when marshaling NodeList.
func (mapping *NodePointersMap) Map(node *Node, idx uint64) {
	mappingIndirect := *mapping
	mappingIndirect[node] = idx
}

// NewNodePointersMap makes new node pointers map.
// Simple wrapper over make(NodePointersMap, 0).
func NewNodePointersMap() *NodePointersMap {
	newMapping := make(NodePointersMap)
	return &newMapping
}
