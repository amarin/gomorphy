package trie

import (
	"errors"
	"sync"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/node"
)

var (
	ErrAddWord              = errors.New("add word")
	ErrNotFound             = errors.New("not found")
	ErrUnexpectedAddedIndex = errors.New("unexpected added index")
)

// Trie хранит DAG символов в добавленных словах.
type Trie[A alphabetSize, M wordsSize] struct {
	mu           *sync.RWMutex
	nodes        indexer.IndexOf[M, node.Node[A, M], *node.Node[A, M]]
	withAlphabet alphabetInterface[A]
	log          logging.Logger
}

// NewGraph создаёт новый DAG.
func NewGraph[A alphabetSize, M wordsSize](useAlphabet alphabetInterface[A]) *Trie[A, M] {
	graph := &Trie[A, M]{
		mu:           new(sync.RWMutex),
		nodes:        *indexer.New[M, node.Node[A, M], *node.Node[A, M]]("dag"),
		withAlphabet: useAlphabet,
		log:          logging.NewNamedLogger("dag"),
	}

	// добавляем корневой узел
	_, _ = graph.nodes.Add(node.New[A, M]())

	return graph
}

func (graph *Trie[A, M]) addWordDirect(word string) (nodeIdx M, err error) {
	var (
		currentParent = M(0)
		parent        *node.Node[A, M]
		addedIdx      M
	)

	for _, char := range []rune(word) {
		if parent, err = graph.nodes.Get(currentParent); err != nil {
			return 0, ErrNotFound
		}

		charIdx, err := graph.withAlphabet.GetOrCreate(char)
		if err != nil {
			return 0, errors.Join(ErrAddWord, err)
		}

		nextIdx, exists := parent.NextIdx(charIdx)
		if !exists {
			newNode := node.New[A, M]()
			if addedIdx, err = graph.nodes.Add(newNode); err != nil {
				return 0, errors.Join(ErrAddWord, err)
			}

			parent.AddNext(charIdx, addedIdx)
		}

		currentParent = nextIdx
	}

	return currentParent, nil
}

func (graph *Trie[A, M]) getWordDirect(word string) (parentIdx M, err error) {
	var parent *node.Node[A, M]

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
func (graph *Trie[A, M]) Add(word string) (M, error) {
	return graph.addWordDirect(word)
}

// Get находит идентификатор финального узла для слова `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку поиска слова
func (graph *Trie[A, M]) Get(word string) (M, error) {
	return graph.getWordDirect(word)
}

// NodesCount находит идентификатор финального узла для слова `word` в хранилище, используя алфавит `useAlphabet`.
// Возвращает идентификатор финального узла или ошибку поиска слова
func (graph *Trie[A, M]) NodesCount() int {
	return graph.nodes.Len()
}
