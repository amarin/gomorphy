package grammemes

import (
	"fmt"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/internal/grammeme"
	"github.com/amarin/gomorphy/pkg/common"
)

// ListIndex allow store indexed grammemes lists.
type ListIndex struct {
	index *grammeme.Index
	items ListList
}

// NewListIndex creates new grammemes lists index for requested grammemes index.
func NewListIndex(index *grammeme.Index, lists ...*List) *ListIndex {
	return &ListIndex{index: index, items: lists}
}

// String returns string representation of grammemes lists index.
func (listIndex ListIndex) String() string {
	return "ListIndex{" + listIndex.items.String() + "}"
}

// Slice returns slice of grammemes lists.
func (listIndex ListIndex) Slice() ListList {
	return listIndex.items
}

// Len returns count of registered grammemes sets.
func (listIndex ListIndex) Len() int {
	return len(listIndex.items)
}

// Add adds grammemes list into index if such list not added before.
func (listIndex *ListIndex) Add(list *List) {
	for _, existedList := range listIndex.items {
		if existedList.EqualTo(list) {
			return
		}
	}

	listIndex.items = append(listIndex.items, list)
}

// GetOrCreateIdx get list id from index. Adds list into index if not added before.
func (listIndex *ListIndex) GetOrCreateIdx(list *List) uint64 {
	for idx, existedList := range listIndex.items {
		if existedList.EqualTo(list) {
			return uint64(idx)
		}
	}

	listIndex.items = append(listIndex.items, list)

	return uint64(listIndex.Len() - 1)
}

// Idx returns id of specified list in index. Returns error if no such grammemes list in index found.
func (listIndex *ListIndex) Idx(list *List) (uint64, error) {
	for idx, existedList := range listIndex.items {
		if existedList.EqualTo(list) {
			return uint64(idx), nil
		}
	}

	return 0, NewErrorf("List not indexed: %v", list)
}

// MarshalBinary returns binary representation of known grammemes sets.
// Returns error if anything goes wrong.
func (listIndex *ListIndex) MarshalBinary() (data []byte, err error) {
	var usingBits binutils.BitsPerIndex

	buffer := binutils.NewEmptyBuffer()

	usingBits, err = binutils.CalculateUseBitsPerIndex(listIndex.Len(), false)

	if err != nil {
		return []byte{}, fmt.Errorf("%w: cant calculate required bits for indexing items: %v", common.ErrMarshal, err)
	}

	if _, err = buffer.WriteObject(usingBits); err != nil {
		return []byte{}, fmt.Errorf("%w: cant write sizing bit: %v", common.ErrMarshal, err)
	}

	if _, err = binutils.WriteUint64ToBufferUsingBits(uint64(listIndex.Len()), buffer, usingBits); err != nil {
		return buffer.Bytes(), fmt.Errorf("%w: cant write list len: %v", common.ErrMarshal, err)
	}

	for _, existedList := range listIndex.items {
		if _, err = buffer.WriteObject(existedList); err != nil {
			return buffer.Bytes(), err
		}
	}

	return buffer.Bytes(), nil
}

// UnmarshalFromBuffer takes required bytes from buffer to unmarshal grammemes sets.
// Returns error if anything goes wrong.
func (listIndex *ListIndex) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	var (
		expectedLen uint64
		usingBits   binutils.BitsPerIndex
	)

	if err = buffer.ReadObjectBytes(&usingBits, 1); err != nil {
		return fmt.Errorf("%w: cant read sizing byte: %v", common.ErrUnmarshal, err)
	}

	if err = binutils.ReadUint64FromBufferUsingBits(&expectedLen, buffer, usingBits); err != nil {
		return fmt.Errorf("%w: cant read index len: %v", common.ErrUnmarshal, err)
	}

	for currentIdx := 0; uint64(currentIdx) < expectedLen; currentIdx++ {
		list := NewList(listIndex.index)
		if err = buffer.ReadObject(list); err != nil {
			return fmt.Errorf("%w: item %d, buffer len %d: %v", currentIdx, buffer.Len(), common.ErrUnmarshal, err)
		}
		listIndex.Add(list)
	}

	return err
}
