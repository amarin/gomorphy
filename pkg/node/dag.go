package node

import (
	"errors"
	"sync"

	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/size"
)

var (
	ErrAddWord              = errors.New("add word")
	ErrNotFound             = errors.New("not found")
	ErrUnexpectedAddedIndex = errors.New("unexpected added index")
)

// Graph хранит DAG символов в добавленных словах.
type Graph[A size.Alphabet, M size.Morphemes] struct {
	mu           *sync.RWMutex
	nodes        indexer.IndexOf[M, Node[A, M], *Node[A, M]]
	withAlphabet alphabetInterface[A]
}

// NewGraph создаёт новый DAG.
func NewGraph[A size.Alphabet, M size.Morphemes](useAlphabet alphabetInterface[A]) *Graph[A, M] {
	graph := &Graph[A, M]{
		mu:           new(sync.RWMutex),
		nodes:        *indexer.New[M, Node[A, M], *Node[A, M]]("dag"),
		withAlphabet: useAlphabet,
	}

	// добавляем корневой узел
	_, _ = graph.nodes.Add(*New[A, M](0, 0, 0))

	return graph
}

func (graph *Graph[A, M]) addWordDirect(word string) (nodeIdx M, err error) {
	var (
		parentIdx = M(0)
		parent    *Node[A, M]
		addedIdx  M
	)

	for _, char := range []rune(word) {
		if parent, err = graph.nodes.Get(parentIdx); err != nil {
			return 0, ErrNotFound
		}

		charIdx, err := graph.withAlphabet.GetOrCreate(char)
		if err != nil {
			return 0, errors.Join(ErrAddWord, err)
		}

		graph.mu.Lock()

		nextIdx, exists := parent.NextIdx(charIdx)
		if !exists {
			nextIdx = M(graph.nodes.Len())
			if addedIdx, err = graph.nodes.Add(*parent.Next(nextIdx, charIdx)); err != nil {
				return 0, errors.Join(ErrAddWord, err)
			}

			if addedIdx != nextIdx {
				return 0, ErrUnexpectedAddedIndex
			}

		}

		parentIdx = nextIdx
		if parent, err = graph.nodes.Get(parentIdx); err != nil {
			return 0, errors.Join(ErrNotFound, err)
		}

		graph.mu.Unlock()
	}

	return parentIdx, nil
}

func (graph *Graph[A, M]) getWordDirect(word string) (parentIdx M, err error) {
	var parent *Node[A, M]

	parentIdx = M(0)

	for _, char := range []rune(word) {
		charIdx, err := graph.withAlphabet.Get(char)
		if err != nil {
			return 0, errors.Join(ErrNotFound, err) // символ не в алфавите
		}

		if parent, err = graph.nodes.Get(parentIdx); err != nil {
			return 0, errors.Join(ErrNotFound, err) // слово не в индексе
		}

		nextIdx, exists := parent.NextIdx(charIdx)
		if !exists {
			return 0, ErrNotFound
		}

		parentIdx = nextIdx
	}

	return parentIdx, nil
}

// Add добавляет слово `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку добавления
func (graph *Graph[A, M]) Add(word string) (M, error) {
	return graph.addWordDirect(word)
}

// Get находит идентификатор финального узла для слова `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку поиска слова
func (graph *Graph[A, M]) Get(word string) (M, error) {
	return graph.getWordDirect(word)
}

// NodesCount находит идентификатор финального узла для слова `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку поиска слова
func (graph *Graph[A, M]) NodesCount() int {
	return graph.nodes.Len()
}
