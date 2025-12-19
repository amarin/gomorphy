package node

import (
	"bytes"
	"fmt"
)

// Node defines trie node for alphabet size A and words size M
type Node[A alphabetSize, M wordsSize] map[A]M

func New[A alphabetSize, M wordsSize]() Node[A, M] {
	return make(Node[A, M])
}

// Bytes возвращает байтовое представление узла.
func (node Node[A, M]) Bytes() ([]byte, error) {
	writer := new(bytes.Buffer)
	if _, err := node.WriteTo(writer); err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func (node Node[A, M]) Hex() (string, error) {
	data, err := node.Bytes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", data), nil
}

// AddNext добавляет потомка с заданным символом и номером ноды.
func (node Node[A, M]) AddNext(charIdx A, wordIdx M) {
	(node)[charIdx] = wordIdx
}

// NextIdx возвращает индекс потомка с заданным индексом символа.
func (node Node[A, M]) NextIdx(charIdx A) (M, bool) {
	next, ok := (node)[charIdx]
	return next, ok
}

// HasNext возвращает true, если зарегистрирован потомок с заданным индексом символа.
func (node Node[A, M]) HasNext(charIdx A) bool {
	_, ok := (node)[charIdx]
	return ok
}
