package grammemes

// import (
// 	"fmt"
// 	"strings"
//
// 	"github.com/amarin/binutils"
// 	"github.com/amarin/gomorphy/internal/grammemes"
// 	"github.com/amarin/gomorphy/pkg/common"
// )
//
// // ListsList содержит список наборов граммем. Используется для задания возможных вариантов распознавания формы слова.
// type ListsList struct {
// 	lists []*GrammemesSet
// 	index *Idx
// }
//
// func (listOfList *ListsList) Idx() *Idx {
// 	return listOfList.index
// }
//
// // NewListsList makes new list of grammemes lists filled with specified lists.
// func NewListsList(index *Idx, lists ...*GrammemesSet) *ListsList {
// 	listOfList := &ListsList{
// 		lists: make([]*GrammemesSet, 0),
// 		index: index,
// 	}
//
// 	copy(listOfList.lists, lists)
//
// 	return listOfList
// }
//
// // Append добавляет набор граммем в список.
// func (listOfList *ListsList) Append(lists ...*GrammemesSet) {
// 	listOfList.lists = append(listOfList.lists, lists...)
// }
//
// // String returns string representation of container.
// func (listOfList ListsList) String() string {
// 	res := make([]string, 0)
// 	for idx, list := range listOfList {
// 		res = append(res, fmt.Sprintf("%d:%v", idx, list.String()))
// 	}
//
// 	return strings.Join(res, ",")
// }
//
// // Len returns count of lists in container.
// func (listOfList *ListsList) Len() int {
// 	return len(*listOfList)
// }
//
// // MarshalBinaryWithIndex makes list of grammemes lists binary representation using grammemes lists index.
// func (listOfList *ListsList) MarshalBinaryWithIndex(listIndex *ListIndex) ([]byte, error) {
// 	buffer := binutils.NewEmptyBuffer()
// 	listIndexUsingBytes, err := binutils.CalculateUseBitsPerIndex(listIndex.Len(), true)
//
// 	if err != nil {
// 		return []byte{}, fmt.Errorf("%w: cant calculate bits width of list index: %v", common.ErrMarshal, err)
// 	}
//
// 	if _, err = buffer.WriteObject(listIndexUsingBytes); err != nil {
// 		return []byte{}, fmt.Errorf("%w: cant write bits width of list index: %v", common.ErrMarshal, err)
// 	}
//
// 	ownUsingBytes, err := binutils.CalculateUseBitsPerIndex(listOfList.Len(), false)
// 	if err != nil {
// 		return []byte{}, fmt.Errorf("%w: cant calculate bits width of list: %v", common.ErrMarshal, err)
// 	}
//
// 	if _, err = buffer.WriteObject(ownUsingBytes); err != nil {
// 		return []byte{}, fmt.Errorf("%w: cant write bits width of list: %v", common.ErrMarshal, err)
// 	}
//
// 	if _, err = binutils.WriteUint64ToBufferUsingBits(uint64(listOfList.Len()), buffer, ownUsingBytes); err != nil {
// 		return []byte{}, fmt.Errorf("%w: cant write length of list: %v", err)
// 	}
//
// 	maxListSize := 0
// 	for _, list := range *listOfList {
// 		if list.Len() > maxListSize {
// 			maxListSize = list.Len()
// 		}
// 	}
//
// 	// write grammemes list
// 	for listIdx, grammemesList := range *listOfList {
// 		idx := listIndex.GetOrCreateIdx(grammemesList)
// 		if _, err = binutils.WriteUint64ToBufferUsingBits(idx, buffer, listIndexUsingBytes); err != nil {
// 			return []byte{}, fmt.Errorf("%w: cant write list %d: %v", common.ErrMarshal, listIdx, err)
// 		}
// 	}
//
// 	return buffer.Bytes(), nil
// }
//
// // UnmarshalFromBuffer takes required binary data from binary buffer to decode ListOfList.
// // Implements binutils.BufferUnmarshaler.
// func (listOfList ListsList) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
// 	bpi := new(binutils.BitsPerIndex)
//
// 	if err := buffer.ReadObject(bpi); err != nil {
// 		return fmt.Errorf("%w: bytes per index: %v", grammemes.ErrDecode, err)
// 	}
//
// 	return nil
// }
//
// // UnmarshalBinary decodes binary data from bytes sequence. Implements encoding.BinaryUnmarshaler.
// func (listOfList *ListsList) UnmarshalBinary(data []byte) error {
// 	return listOfList.UnmarshalFromBuffer(binutils.NewBuffer(data))
// }
