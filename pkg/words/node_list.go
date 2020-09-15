package words

import (
	"fmt"
	"math"

	bin "github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/common"
)

// NodeList implements storage of index node pointers.
// Used internally in many cases.
type NodeList []*Node

// PointersStrings returns strings, containing descriptive information about each node in slice.
// Used internally.
func (list *NodeList) Strings() []string {
	sliceRepresentation := make([]string, 0)
	for idx, nodePtr := range *list {
		sliceRepresentation = append(
			sliceRepresentation,
			fmt.Sprintf("%d: %p %v -> %p", idx, nodePtr, *nodePtr, nodePtr.parent),
		)
	}

	return sliceRepresentation
}

// NewNodeList creates empty NodeList.
func NewNodeList() *NodeList {
	nodeList := make(NodeList, 0)
	return &nodeList
}

// RequiredBits returns minimal sizing for index values basing on node list size.
func (list *NodeList) RequiredBits() (bin.BitsPerIndex, error) {
	return bin.CalculateUseBitsPerIndex(len(*list), true)
}

// Len returns list length. Simple wrapper around len(list).
func (list *NodeList) Len() int {
	return len(*list)
}

// WriteIndexLen writes list length into buffer with usingBits data type.
func (list *NodeList) WriteIndexLen(buffer *bin.Buffer, usingBits bin.BitsPerIndex) (int, error) {
	switch usingBits {
	case bin.Use8bit:
		return buffer.WriteUint8(uint8(len(*list)))
	case bin.Use16bit:
		return buffer.WriteUint16(uint16(len(*list)))
	case bin.Use32bit:
		return buffer.WriteUint32(uint32(len(*list)))
	case bin.Use64bit:
		return buffer.WriteUint64(uint64(len(*list)))
	default:
		return 0, fmt.Errorf("%w: expected one of 8, 16, 32 or 64 bit", common.ErrIndexSize)
	}
}

// WriteNoParent writes into target buffer no parent index using requested index values sizing.
func (list *NodeList) WriteNoParent(buffer *bin.Buffer, usingBits bin.BitsPerIndex) (int, error) {
	switch usingBits {
	case bin.Use8bit:
		return buffer.WriteUint8(uint8(math.MaxUint8))
	case bin.Use16bit:
		return buffer.WriteUint16(uint16(math.MaxUint16))
	case bin.Use32bit:
		return buffer.WriteUint32(uint32(math.MaxUint32))
	case bin.Use64bit:
		return buffer.WriteUint64(uint64(math.MaxUint64))
	default:
		return 0, fmt.Errorf("%w: expected one of 8, 16, 32 or 64 bit", common.ErrIndexSize)
	}
}

// MarshalParentIdx writes index of node parent into target buffer using required index sizing.
func (list *NodeList) MarshalParentIdx(b *bin.Buffer, n *Node, bits bin.BitsPerIndex, m *NodePointersMap) (int, error) {
	// switch by parent
	switch n.parent {
	case nil:
		return list.WriteNoParent(b, bits) // no parent, write no parent bytes
	default:
		// take parent idx
		parentIdx, err := m.Idx(n.parent)
		if err != nil {
			return 0, fmt.Errorf("%w: cant find parent idx: %v", common.ErrIndexSize, err)
		}
		// parent idx found, write it using uin8, uint16, uint32 or uint64 depending of required bits per idx.
		switch bits {
		case bin.Use8bit:
			if parentIdx > math.MaxUint8-1 {
				return 0, fmt.Errorf("%w: sizing %d > max uint8", common.ErrIndexSize, parentIdx)
			}

			return b.WriteUint8(uint8(parentIdx))
		case bin.Use16bit:
			if parentIdx > math.MaxUint16-1 {
				return 0, fmt.Errorf("%w: sizing %d > max uint16", common.ErrIndexSize, parentIdx)
			}

			return b.WriteUint16(uint16(parentIdx))
		case bin.Use32bit:
			if parentIdx > math.MaxUint32-1 {
				return 0, fmt.Errorf("%w: sizing %d > max uint32", common.ErrIndexSize, parentIdx)
			}

			return b.WriteUint32(uint32(parentIdx))
		case bin.Use64bit:
			if parentIdx > math.MaxUint64-1 {
				return 0, fmt.Errorf("%w: sizing %d > max uint64", common.ErrIndexSize, parentIdx)
			}

			return b.WriteUint64(parentIdx)
		default:
			return 0, fmt.Errorf("%w: expected one of 8, 16, 32 or 64 bit", common.ErrIndexSize)
		}
	}
}

// ReadIndexLen takes node list length from buffer using predefined sizing information of length value.
func (list *NodeList) ReadIndexLen(buffer *bin.Buffer, usingBits bin.BitsPerIndex) (uint64, error) {
	switch usingBits {
	case bin.Use8bit:
		var indexLen uint8
		if err := buffer.ReadUint8(&indexLen); err != nil {
			return 0, fmt.Errorf("%w: cant read 1-byte index len", common.ErrIndexSize)
		}

		return uint64(indexLen), nil
	case bin.Use16bit:
		var indexLen uint16
		if err := buffer.ReadUint16(&indexLen); err != nil {
			return 0, fmt.Errorf("%w: cant read 2-byte index len", common.ErrIndexSize)
		}

		return uint64(indexLen), nil
	case bin.Use32bit:
		var indexLen uint32
		if err := buffer.ReadUint32(&indexLen); err != nil {
			return 0, fmt.Errorf("%w: cant read 4-byte index len", common.ErrIndexSize)
		}

		return uint64(indexLen), nil
	case bin.Use64bit:
		var indexLen uint64
		if err := buffer.ReadUint64(&indexLen); err != nil {
			return 0, fmt.Errorf("%w: cant read 8-byte index len", common.ErrIndexSize)
		}

		return indexLen, nil
	}

	return 0, fmt.Errorf("%w: expected one of 8, 16, 32 or 64 bit", common.ErrIndexSize)
}

// ReadParentIdx takes from buffer required bytes using usingBits information to detect expected sizing for value
// and places result into target uint64 pointer.
func (list *NodeList) ReadParentIdx(buffer *bin.Buffer, target *uint64, usingBits bin.BitsPerIndex) error {
	switch usingBits {
	case bin.Use8bit:
		var uint8ParentIdx uint8
		if err := buffer.ReadUint8(&uint8ParentIdx); err != nil {
			return fmt.Errorf("%w: cant read parent idx: %v", common.ErrIndexSize, err)
		}

		*target = uint64(uint8ParentIdx)
	case bin.Use16bit:
		var uint16ParentIdx uint16
		if err := buffer.ReadUint16(&uint16ParentIdx); err != nil {
			return fmt.Errorf("%w: cant read parent idx: %v", common.ErrIndexSize, err)
		}

		*target = uint64(uint16ParentIdx)
	case bin.Use32bit:
		var uint32ParentIdx uint32
		if err := buffer.ReadUint32(&uint32ParentIdx); err != nil {
			return fmt.Errorf("%w: cant read parent idx: %v", common.ErrIndexSize, err)
		}

		*target = uint64(uint32ParentIdx)
	case bin.Use64bit:
		var uint64ParentIdx uint64
		if err := buffer.ReadUint64(&uint64ParentIdx); err != nil {
			return fmt.Errorf("%w: cant read parent idx: %v", common.ErrIndexSize, err)
		}

		*target = uint64ParentIdx
	default:
		return fmt.Errorf("%w: expected one of 8, 16, 32 or 64 bit", common.ErrIndexSize)
	}

	return nil
}

// MakeReverseIndex prepares idx to node mapping filling nil values for no parent using sizing information in usingBits.
func (list *NodeList) MakeReverseIndex(usingBits bin.BitsPerIndex) (map[uint64]*Node, error) {
	// map memory to index->node index.
	idxToNodeMapper := make(map[uint64]*Node)

	switch usingBits {
	case bin.Use8bit:
		idxToNodeMapper[uint64(math.MaxUint8)] = nil
	case bin.Use16bit:
		idxToNodeMapper[uint64(math.MaxUint16)] = nil
	case bin.Use32bit:
		idxToNodeMapper[uint64(math.MaxUint32)] = nil
	case bin.Use64bit:
		idxToNodeMapper[uint64(math.MaxUint64)] = nil
	default:
		return idxToNodeMapper, fmt.Errorf("%w: expected one of 8, 16, 32 or 64 bit", common.ErrIndexSize)
	}

	return idxToNodeMapper, nil
}

// MarshalBinary returns binary representation of node list.
// Used by Index.MarshalBinary. Implements BinaryMarshaler.
func (list *NodeList) MarshalBinary() (data []byte, err error) {
	buffer := bin.NewEmptyBuffer()
	usingBits, err := list.RequiredBits()

	if err != nil {
		return []byte{}, fmt.Errorf("%w: cant calculate index values sizing: %v", common.ErrIndexSize, err)
	}

	if _, err := buffer.WriteObject(usingBits); err != nil {
		return []byte{}, fmt.Errorf("%w: cant write indexes sizing byte: %v", common.ErrIndexSize, err)
	} else if _, err := list.WriteIndexLen(buffer, usingBits); err != nil {
		return []byte{}, fmt.Errorf("%w: cant write index len: %v", common.ErrIndexSize, err)
	}

	mapper := NewNodePointersMap()

	for nodeIndex, nodePtr := range *list {
		if _, err := list.MarshalParentIdx(buffer, nodePtr, usingBits, mapper); err != nil {
			return []byte{}, fmt.Errorf("%w: cant write parent idx: %v", common.ErrIndexSize, err)
		}
		// write node rune
		if runeByte, err := text.EncodeString(string(nodePtr.Rune()), binaryCharmap); err != nil {
			return []byte{}, fmt.Errorf("%w: cant encode rune: %v", common.ErrIndexSize, err)
		} else if _, err := buffer.WriteBytes(runeByte); err != nil {
			return []byte{}, fmt.Errorf("%w: cant write rune: %v", common.ErrIndexSize, err)
		}
		// write node grammemes list
		if _, err := buffer.WriteUint8(uint8(len(nodePtr.Forms()))); err != nil {
			return []byte{}, fmt.Errorf("%w: write grammemes list length: %v", common.ErrIndexSize, err)
		}
		// write grammemes list
		for grammemesListIdx, grammemesList := range nodePtr.Forms() {
			if _, err := buffer.WriteObject(grammemesList); err != nil {
				return []byte{}, fmt.Errorf("%w: write grammemes list %d: %v", common.ErrIndexSize, grammemesListIdx, err)
			}
		}

		mapper.Map(nodePtr, uint64(nodeIndex))
	}

	return buffer.Bytes(), nil
}

// MakeListIndex creates used grammemes lists index.
func (list *NodeList) MakeListIndex(grammemesIndex *grammemes.Index) (*grammemes.ListIndex, error) {
	listIndex := grammemes.NewListIndex(grammemesIndex)
	// build grammemes lists index
	for _, node := range *list {
		for _, grammemesList := range node.Forms() {
			_ = listIndex.GetOrCreateIdx(grammemesList)
		}
	}

	return listIndex, nil
}

// MarshalBinary returns binary representation of node list.
// Used by Index.MarshalBinary. Implements BinaryMarshaller.
func (list *NodeList) MarshalBinaryWith(listIndex *grammemes.ListIndex) (data []byte, err error) {
	buffer := bin.NewEmptyBuffer()
	usingBits, err := list.RequiredBits()

	if err != nil {
		return []byte{}, fmt.Errorf("%w: cant calculate index values sizing: %v", common.ErrIndexSize, err)
	}

	if _, err := buffer.WriteObject(usingBits); err != nil {
		return []byte{}, fmt.Errorf("%w: cant write indexes sizing byte: %v", common.ErrIndexSize, err)
	} else if _, err := list.WriteIndexLen(buffer, usingBits); err != nil {
		return []byte{}, fmt.Errorf("%w: cant write index len: %v", common.ErrIndexSize, err)
	}

	mapper := NewNodePointersMap()

	for nodeIndex, nodePtr := range *list {
		if _, err := list.MarshalParentIdx(buffer, nodePtr, usingBits, mapper); err != nil {
			return []byte{}, fmt.Errorf("%w: cant write parent idx: %v", common.ErrIndexSize, err)
		}
		// write node rune
		if runeByte, err := text.EncodeString(string(nodePtr.Rune()), binaryCharmap); err != nil {
			return []byte{}, fmt.Errorf("%w: cant encode rune: %v", common.ErrIndexSize, err)
		} else if _, err := buffer.WriteBytes(runeByte); err != nil {
			return []byte{}, fmt.Errorf("%w: cant write rune: %v", common.ErrIndexSize, err)
		}
		// write node grammemes list
		if _, err := buffer.WriteUint8(uint8(len(nodePtr.Forms()))); err != nil {
			return []byte{}, fmt.Errorf("%w: cant write grammemes list length: %v", common.ErrIndexSize, err)
		}
		// write grammemes list
		for grammemesListIdx, grammemesList := range nodePtr.Forms() {
			if _, err := buffer.WriteObject(grammemesList); err != nil {
				return []byte{}, fmt.Errorf("%w: write grammemes list %d: %v", common.ErrIndexSize, grammemesListIdx, err)
			}
		}

		mapper.Map(nodePtr, uint64(nodeIndex))
	}

	return buffer.Bytes(), nil
}

// UnmarshalFromBufferWithIndex takes from buffer required.
func (list *NodeList) UnmarshalFromBufferWithIndex(buffer *bin.Buffer, index *grammemes.Index) error {
	var (
		parentIdx, listLen           uint64
		grammemesCount, nodeCharByte uint8
		parent, node                 *Node
		ok                           bool
		err                          error
		characterAsString            string
		mapper                       map[uint64]*Node
		usingBits                    bin.BitsPerIndex
	)

	if err = buffer.ReadObjectBytes(&usingBits, 1); err != nil {
		return fmt.Errorf("%w: cant read indexes size byte: %v", common.ErrIndexSize, err)
	} else if listLen, err = list.ReadIndexLen(buffer, usingBits); err != nil {
		return fmt.Errorf("%w: cant read index len: %v", common.ErrIndexSize, err)
	} else if mapper, err = list.MakeReverseIndex(usingBits); err != nil {
		return fmt.Errorf("%w: cant init parents index: %v", common.ErrIndexSize, err)
	}

	// load nodes
	for idx := uint64(0); idx < listLen; idx++ {
		if err = list.ReadParentIdx(buffer, &parentIdx, usingBits); err != nil {
			return fmt.Errorf("%w: cant read parent grammemeListIdx: %v", common.ErrIndexSize, err)
		} else if err = buffer.ReadUint8(&nodeCharByte); err != nil {
			return fmt.Errorf("%w: cant read character byte: %v", common.ErrIndexSize, err)
		} else if err = buffer.ReadUint8(&grammemesCount); err != nil {
			return fmt.Errorf("%w: cant read grammemes list len: %v", common.ErrIndexSize, err)
		} else if parent, ok = mapper[parentIdx]; !ok {
			return fmt.Errorf("%w: cant found parent node in index", common.ErrChildrenError)
		} else if characterAsString, err = text.DecodeBytes([]byte{nodeCharByte}, binaryCharmap); err != nil {
			return fmt.Errorf("%w: cant decode character byte: %v", common.ErrIndexSize, err)
		}

		switch parent {
		case nil:
			node = NewMappingNode(nil, []rune(characterAsString)[0])
		default:
			node = parent.Child([]rune(characterAsString)[0])
		}
		// load node grammemes
		for grammemeListIdx := uint8(0); grammemeListIdx < grammemesCount; grammemeListIdx++ {
			grammemesInNode := grammemes.NewList(index)
			if err := buffer.ReadObject(grammemesInNode); err != nil {
				return fmt.Errorf("%w: read grammemes list %d: %v", common.ErrIndexSize, grammemeListIdx, err)
			} else if err := node.AddGrammemes(grammemesInNode); err != nil {
				return fmt.Errorf("%w: add grammemes to node: %v", common.ErrIndexSize, err)
			}
		}

		mapper[idx] = node
		*list = append(*list, node)
	} //

	return nil
}
