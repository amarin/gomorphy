package simple

import (
	"io"

	"github.com/amarin/gomorphy/internal/storage"
)

const (
	indexName = "simple"
)

func (i *Index[A, N, T, S]) storageConfig() storage.Config {
	return *storage.Define(
		*storage.Block("magic", storage.StaticBytes([]byte(`SIMPLE`))),
		*storage.Block("alpha", i.alphabet),
		*storage.Block("tags", i.tags),
		*storage.Block("sets", i.tagsSets),
		*storage.Block("dag", i.dag),
		*storage.Block("nmap", i.nodeToWord),
		*storage.Block("nmap", i.nodeToWord),
		*storage.Block("words", i.words),
	)
}

func (i *Index[A, N, T, S]) ReadFrom(r io.Reader) (n int64, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return storage.NewReader(indexName, i.storageConfig()).ReadFrom(r)
}

func (i *Index[A, N, T, S]) WriteTo(w io.Writer) (n int64, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return storage.NewWriter(indexName, i.storageConfig()).WriteTo(w)
}
