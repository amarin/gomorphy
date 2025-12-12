package simple

import (
	"io"

	"github.com/amarin/gomorphy/internal/storage"
)

const (
	indexName = "simple"
)

func (i *Index) storageConfig() storage.Config {
	return *storage.Define(
		*storage.Block("magic", storage.StaticBytes([]byte(`DAG`))),
		*storage.Block("alphabet", i.alphabet),
		*storage.Block("tags", i.tagIndexer),
		*storage.Block("tagSets", i.tagSets),
		*storage.Block("dag", i.dag),
	)
}

func (i *Index) ReadFrom(r io.Reader) (n int64, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return storage.NewReader(indexName, i.storageConfig()).ReadFrom(r)
}

func (i *Index) WriteTo(w io.Writer) (n int64, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return storage.NewWriter(indexName, i.storageConfig()).WriteTo(w)
}

// BinaryWriteTo записывает байтовое представление индекса в заданный io.Writer.
func (i *Index) BinaryWriteTo(writer io.Writer) (err error) {
	var storageSize int64

	if storageSize, err = i.WriteTo(writer); err != nil {
		return err
	}

	i.log.Infof("stored %d nodes into %d bytes", i.dag.NodesCount(), storageSize)

	return nil
}
