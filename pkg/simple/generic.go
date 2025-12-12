package simple

import (
	"sync"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/internal/storage"
	"github.com/amarin/gomorphy/pkg/alphabet"
	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/node"
	"github.com/amarin/gomorphy/pkg/size"
	"github.com/amarin/gomorphy/pkg/tag"
	"github.com/amarin/gomorphy/pkg/tagset"
)

// Index реализует управление и взаимодействие с индексом графа.
type Index[A size.Alphabet, N size.Morphemes, T size.TagsSize, S size.TagSetSize] struct {
	storage.Config
	mu         *sync.RWMutex
	alphabet   *alphabet.Alphabet[A]
	dag        *node.Graph[A, N]
	tagIndexer *indexer.IndexOf[T, tag.Tag, *tag.Tag]
	tagSets    *tagset.Set[T, S]
	words      indexer.IndexOf[N, node.ID[N], *node.ID[N]]
	log        logging.Logger
}

// New создаёт новый экземпляр индекса.
func newIndex[A size.Alphabet, N size.Morphemes, T size.TagsSize, S size.TagSetSize]() *Index[A, N, T, S] {
	a := alphabet.New[A]()
	// загружаем базовый алфавит
	if err := a.Reset(cyrillicLetters); err != nil {
		panic(err)
	}

	tagsIndex := tag.NewIndexer[T]()

	i := &Index[A, N, T, S]{
		dag:        node.NewGraph[A, N](a),
		alphabet:   a,
		tagIndexer: tagsIndex,
		tagSets:    tagset.NewSet[T, S](tagsIndex),
		mu:         new(sync.RWMutex),
		log:        logging.NewNamedLogger(indexName),
	}

	return i
}

func (i Index[A, N, T, S]) WordsCount() int {
	//TODO implement me
	panic("implement me")
}

func (i Index[A, N, T, S]) NodesCount() int {
	//TODO implement me
	panic("implement me")
}

func (i Index[A, N, T, S]) Optimize() {
	//TODO implement me
	panic("implement me")
}

func (i Index[A, N, T, S]) Add(word string, tags ...any) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (i Index[A, N, T, S]) RegisterTag(t tag.Tag) (int, error) {
	//TODO implement me
	panic("implement me")
}
