package simple

import (
	"io"

	"github.com/amarin/gomorphy/internal/alphabet"
	"github.com/amarin/gomorphy/internal/node"
)

// Index реализует управление и взаимодействие с индексом графа.
type Index struct {
	dag *node.Graph[uint8, uint16]
}

// New создаёт новый экземпляр индекса.
func New() *Index {
	a := alphabet.New[uint8]()
	return &Index{
		dag: node.NewGraph[uint8, uint16](a),
	}
}

// WordsCount возвращает количество слов в индексе.
func (i Index) WordsCount() int {
	return 0
}

// NodesCount возвращает количество узлов в индексе.
func (i Index) NodesCount() int {
	return i.dag.NodesCount()
}

// Optimize выполняет внутреннюю оптимизацию индекса.
func (i Index) Optimize() {}

// BinaryWriteTo записывает байтовое представление индекса в заданный io.Writer.
func (i Index) BinaryWriteTo(writer io.Writer) error {
	return i.dag.BinaryWriteTo(writer)
}

// Add добавляет слово и его характеристики в индекс.
func (i Index) Add(word string, _ ...any) (int, error) {
	idx, err := i.dag.Add(word)
	if err != nil {
		return 0, err
	}

	return int(idx), nil
}
