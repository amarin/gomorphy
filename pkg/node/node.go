package node

import (
	"bytes"
	"fmt"
	"io"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/size"
)

// Node задаёт структуру хранения данных об узле DAG.
type Node[A size.Alphabet, M size.Morphemes] struct {
	idx     M
	prev    M
	charIdx A
	next    map[A]M
}

// New создаёт новый узел DAG c индексом idx, индексом символа charIdx и индексом предыдущего узла prev.
func New[A size.Alphabet, M size.Morphemes](idx M, charIdx A, prev M) *Node[A, M] {
	return &Node[A, M]{
		idx:     idx,
		charIdx: charIdx,
		prev:    prev,
		next:    make(map[A]M),
	}
}

// Next создаёт новый узел-потомок с индексом idx и индексом символа charIdx, добавляя его в список потомков текущего.
func (node *Node[A, M]) Next(idx M, charIdx A) *Node[A, M] {
	node.next[charIdx] = idx
	return New(idx, charIdx, node.idx)
}

// PrevIdx возвращает индекс предыдущего узла.
func (node *Node[A, M]) PrevIdx() M {
	return node.prev
}

// NextIdx возвращает индекс потомка с заданным индексом символа.
func (node *Node[A, M]) NextIdx(charIdx A) (M, bool) {
	next, ok := node.next[charIdx]
	return next, ok
}

// HasNext возвращает true, если зарегистрирован потомок с заданным индексом символа.
func (node *Node[A, M]) HasNext(charIdx A) bool {
	_, ok := node.next[charIdx]
	return ok
}

// CharIdx возвращает индекс символа в алфавите.
func (node *Node[A, M]) CharIdx() A {
	return node.charIdx
}

// WriteTo записывает бинарное представление узла в заданный io.Writer.
func (node *Node[A, M]) WriteTo(w io.Writer) (bytesWritten int64, err error) {
	var (
		idx     any = node.idx
		prev    any = node.prev
		charIdx any = node.charIdx
	)

	bytesWritten = 0
	res := binutils.NewBinaryWriter(w)

	switch typedValue := idx.(type) {
	case uint16:
		if err = res.WriteUint16(typedValue); err != nil {
			return bytesWritten, err
		}
		bytesWritten += 2
	case uint32:
		if err = res.WriteUint32(typedValue); err != nil {
			return bytesWritten, err
		}
		bytesWritten += 4
	}

	switch typedValue := prev.(type) {
	case uint16:
		if err = res.WriteUint16(typedValue); err != nil {
			return bytesWritten, err
		}
		bytesWritten += 2
	case uint32:
		if err = res.WriteUint32(typedValue); err != nil {
			return bytesWritten, err
		}
		bytesWritten += 4
	}

	switch typedValue := charIdx.(type) {
	case uint8:
		if err = res.WriteUint8(typedValue); err != nil {
			return bytesWritten, err
		}
		bytesWritten++
	case uint16:
		if err = res.WriteUint16(typedValue); err != nil {
			return bytesWritten, err
		}
		bytesWritten += 2
	case uint32:
		if err = res.WriteUint32(typedValue); err != nil {
			return bytesWritten, err
		}
		bytesWritten += 4
	}

	return bytesWritten, nil
}

// Bytes возвращает байтовое представление узла.
func (node *Node[A, M]) Bytes() ([]byte, error) {
	writer := new(bytes.Buffer)
	if _, err := node.WriteTo(writer); err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func (node *Node[A, M]) Hex() (string, error) {
	data, err := node.Bytes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", data), nil
}

// ReadFrom записывает бинарное представление узла в заданный io.Writer.
func (node *Node[A, M]) ReadFrom(r io.Reader) (count int64, err error) {
	var (
		idx     any = node.idx
		prev    any = node.prev
		charIdx any = node.charIdx
	)

	count = 0
	res := binutils.NewBinaryReader(r)

	switch idx.(type) {
	case uint16:
		if idx, err = res.ReadUint16(); err != nil {
			return int64(res.BytesTaken()), err
		}
	case uint32:
		if idx, err = res.ReadUint32(); err != nil {
			return int64(res.BytesTaken()), err
		}
	}
	node.idx = idx.(M)

	switch prev.(type) {
	case uint16:
		if prev, err = res.ReadUint16(); err != nil {
			return int64(res.BytesTaken()), err
		}
	case uint32:
		if prev, err = res.ReadUint32(); err != nil {
			return int64(res.BytesTaken()), err
		}
	}
	node.prev = prev.(M)

	switch charIdx.(type) {
	case uint8:
		if charIdx, err = res.ReadUint8(); err != nil {
			return int64(res.BytesTaken()), err
		}
	case uint16:
		if charIdx, err = res.ReadUint16(); err != nil {
			return int64(res.BytesTaken()), err
		}
	case uint32:
		if charIdx, err = res.ReadUint32(); err != nil {
			return int64(res.BytesTaken()), err
		}
	}
	node.charIdx = charIdx.(A)

	return int64(res.BytesTaken()), nil
}
