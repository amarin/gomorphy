package node

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/amarin/gomorphy/internal/size"
)

var (
	ErrAddWord  = errors.New("add word")
	ErrNotFound = errors.New("not found")
)

// Graph хранит DAG символов в добавленных словах.
type Graph[A size.Alphabet, M size.Morphemes] struct {
	mu           *sync.RWMutex
	nodes        []Node[A, M]
	withAlphabet alphabetInterface[A]
}

func (graph *Graph[A, M]) BinaryWriteTo(writer io.Writer) error {
	if _, err := writer.Write([]byte("DAG")); err != nil {
		return err
	}

	alphabetString := graph.withAlphabet.String()
	if _, err := fmt.Fprintf(writer, "%dZ", len(alphabetString)); err != nil {
		return err
	}
	if _, err := writer.Write([]byte(alphabetString)); err != nil {
		return err
	}

	for _, node := range graph.nodes {
		if _, err := node.WriteTo(writer); err != nil {
			return err
		}
	}

	return nil
}

// NewGraph создаёт новый DAG.
func NewGraph[A size.Alphabet, M size.Morphemes](useAlphabet alphabetInterface[A]) *Graph[A, M] {
	graph := &Graph[A, M]{
		mu:           new(sync.RWMutex),
		nodes:        make([]Node[A, M], 0),
		withAlphabet: useAlphabet,
	}

	// добавляем корневой узел
	graph.nodes = append(graph.nodes, *New[A, M](0, 0, 0))
	return graph
}

func (graph *Graph[A, M]) addWordDirect(word string) (M, error) {
	parentIdx := M(0)

	for _, char := range []rune(word) {
		charIdx, err := graph.withAlphabet.GetOrCreate(char)
		if err != nil {
			return 0, errors.Join(ErrAddWord, err)
		}

		graph.mu.Lock()
		nextIdx, exists := graph.nodes[parentIdx].NextIdx(charIdx)
		if !exists {
			nextIdx = M(len(graph.nodes))
			graph.nodes = append(graph.nodes, *graph.nodes[parentIdx].Next(nextIdx, charIdx))
		}
		parentIdx = nextIdx
		graph.mu.Unlock()
	}

	return parentIdx, nil
}

func (graph *Graph[A, M]) getWordDirect(word string) (M, error) {
	parentIdx := M(0)

	for _, char := range []rune(word) {
		charIdx, err := graph.withAlphabet.Get(char)
		if err != nil {
			return 0, errors.Join(ErrNotFound, err) // символ не в алфавите
		}

		nextIdx, exists := graph.nodes[parentIdx].NextIdx(charIdx)
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
	return len(graph.nodes)
}
