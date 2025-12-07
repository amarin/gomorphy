package simple

import (
	"io"

	"github.com/amarin/gomorphy/internal/alphabet"
	"github.com/amarin/gomorphy/internal/node"
)

type Index struct {
	dag *node.Graph[uint8, uint16]
}

func New() *Index {
	a := alphabet.New[uint8]()
	return &Index{
		dag: node.NewGraph[uint8, uint16](a),
	}
}

func (i Index) WordsCount() int {
	return 0
}

func (i Index) NodesCount() int {
	return i.dag.NodesCount()
}

func (i Index) Optimize() {}

func (i Index) BinaryWriteTo(writer io.Writer) error {
	return i.dag.BinaryWriteTo(writer)
}

func (i Index) Add(word string, _ ...any) (int, error) {
	idx, err := i.dag.Add(word)
	if err != nil {
		return 0, err
	}

	return int(idx), nil
}
