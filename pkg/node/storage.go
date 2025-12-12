package node

import (
	"io"

	"github.com/amarin/gomorphy/internal/storage"
)

const (
	blockName = "nodes"
)

func (graph *Graph[A, M]) StorageConfig() storage.Config {
	return *storage.Define(
		*storage.Block("magic", storage.StaticBytes([]byte("DAG"))),
		*storage.Block("nodes", &graph.nodes),
	)
}

func (graph *Graph[A, M]) ReadFrom(r io.Reader) (n int64, err error) {
	graph.mu.Lock()
	defer graph.mu.Unlock()

	return storage.NewReader(blockName, graph.StorageConfig()).ReadFrom(r)
}

func (graph *Graph[A, M]) WriteTo(w io.Writer) (n int64, err error) {
	graph.mu.Lock()
	defer graph.mu.Unlock()

	return storage.NewWriter(blockName, graph.StorageConfig()).WriteTo(w)
}
