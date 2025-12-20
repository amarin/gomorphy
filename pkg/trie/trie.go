package trie

import (
	"errors"

	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/node"
)

var (
	ErrAddWord  = errors.New("add word")
	ErrNotFound = errors.New("not found")
)

// Trie stores sequences of characters of alphabet power A to provide words indexes of type M.
// Implements trie structure&
type Trie[A alphabetSize, M wordsSize] struct {
	indexer.IndexOf[M, node.Node[A, M], *node.Node[A, M]]
}

// New creates new trie for alphabet power A and words power M.
func New[A alphabetSize, M wordsSize]() *Trie[A, M] {
	graph := &Trie[A, M]{
		IndexOf: *indexer.New[M, node.Node[A, M], *node.Node[A, M]]("dag"),
	}

	// adding trie root
	_, _ = graph.IndexOf.Add(node.New[A, M]())

	return graph
}

func (graph *Trie[A, M]) addWordDirect(word []A) (nodeIdx M, err error) {
	var (
		currentParent = M(0)
		parent        *node.Node[A, M]
		nextIdx       M
		exists        bool
	)

	for _, charIdx := range word {
		if parent, err = graph.IndexOf.Get(currentParent); err != nil {
			return 0, ErrNotFound
		}

		if nextIdx, exists = parent.GetNext(charIdx); !exists {
			newNode := node.New[A, M]()
			if nextIdx, err = graph.IndexOf.Add(newNode); err != nil {
				return 0, errors.Join(ErrAddWord, err)
			}

			parent.SetNext(charIdx, nextIdx)
		}

		currentParent = nextIdx
	}

	return currentParent, nil
}

func (graph *Trie[A, M]) getWordDirect(word []A) (parentIdx M, err error) {
	var (
		parent  *node.Node[A, M]
		nextIdx M
		exists  bool
		charIdx A
	)

	parentIdx = M(0)

	for _, charIdx = range word {
		if parent, err = graph.IndexOf.Get(parentIdx); err != nil {
			return 0, errors.Join(ErrNotFound, err) // слово не в индексе
		}

		if nextIdx, exists = parent.GetNext(charIdx); !exists {
			return 0, ErrNotFound
		}

		parentIdx = nextIdx
	}

	return parentIdx, nil
}

// Add добавляет слово `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку добавления
func (graph *Trie[A, M]) Add(word []A) (M, error) {
	return graph.addWordDirect(word)
}

// Get находит идентификатор финального узла для слова `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку поиска слова
func (graph *Trie[A, M]) Get(word []A) (M, error) {
	return graph.getWordDirect(word)
}

// Get находит идентификатор финального узла для слова `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку поиска слова
func (graph *Trie[A, M]) GetNodes(word M) ([]A, error) {
	item, err := graph.IndexOf.Get(word)
	return graph.getWordDirect(word)
}

// NodesCount находит идентификатор финального узла для слова `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку поиска слова
func (graph *Trie[A, M]) NodesCount() int {
	return graph.IndexOf.Len()
}
