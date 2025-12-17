package simple

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/internal/storage"
	"github.com/amarin/gomorphy/pkg/alphabet"
	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/node"
	"github.com/amarin/gomorphy/pkg/size"
	"github.com/amarin/gomorphy/pkg/tag"
	"github.com/amarin/gomorphy/pkg/tagset"
	"github.com/amarin/gomorphy/pkg/words"
)

var (
	ErrUnknownTag     = errors.New("unknown tag")
	ErrRegisterTagSet = errors.New("tag set registration failed")
	ErrRegisterWord   = errors.New("word registration failed")
	ErrNotFound       = errors.New("not found")
	ErrTagSetNotFound = errors.New("tag set not found")
)

// Index реализует управление и взаимодействие с индексом графа.
type Index[A size.Alphabet, N size.Morphemes, T size.TagsSize, S size.TagSetSize] struct {
	storage.Config
	mu         *sync.RWMutex
	alphabet   *alphabet.Alphabet[A]
	dag        *node.Graph[A, N]
	tags       *indexer.IndexOf[T, tag.Name, *tag.Name]
	tagsMap    map[tag.Name]T
	tagsSets   *tagset.Set[T, S]
	nodeToWord *indexer.IndexOf[N, node.ID[N], *node.ID[N]]
	words      *indexer.IndexOf[N, words.TagSets[S], *words.TagSets[S]]
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
		alphabet:   a,
		dag:        node.NewGraph[A, N](a),
		tags:       tagsIndex,
		tagsSets:   tagset.NewSet[T, S](tagsIndex),
		tagsMap:    make(map[tag.Name]T),
		nodeToWord: indexer.New[N, node.ID[N], *node.ID[N]]("nodeToWord"),
		words:      indexer.New[N, words.TagSets[S], *words.TagSets[S]]("wordsTagSets"),
		mu:         new(sync.RWMutex),
		log:        logging.NewNamedLogger(indexName).WithLevel(logging.LevelWarn),
	}

	return i
}

func (i *Index[A, N, T, S]) WordsCount() int {
	return i.words.Len()
}

func (i *Index[A, N, T, S]) NodesCount() int {
	return i.dag.NodesCount()
}

func (i *Index[A, N, T, S]) Optimize() {
}

func (i *Index[A, N, T, S]) Add(word string, tags ...tag.Name) (int, error) {
	log := i.log.WithKeys(logging.Keys{"word": word, "tags": tags})
	log.Info("adding word")

	i.mu.Lock()
	defer i.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Error("recovering from panic")
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
			panic(r)
		}
	}()

	tagsOk := true
	tagIds := make([]T, len(tags))

	// получаем идентификаторы тегов, должны быть зарегистрированы до этого с помощью RegisterTag
	log.Debug("lookup registered tags ids")
	for idx, tagName := range tags {
		log.Debugf("lookup tags %s id", tagName)
		tagIds[idx], tagsOk = i.tagsMap[tagName]
		if !tagsOk {
			log.WithKey("tag", tagName.String()).Error("get tag id failed")
			return 0, fmt.Errorf("%w: %s", ErrUnknownTag, tagName)
		}
	}

	// получаем идентификатор набора тегов
	log.Debug("lookup tag set id")
	tagSetIdx, err := i.tagsSets.GetOrCreate(tagIds...)
	if err != nil {
		log.WithError(err).Error("get tag set index failed")
		return 0, errors.Join(ErrRegisterTagSet, err)
	}
	log.Debugf("use tag set id %d", tagSetIdx)

	// получаем идентификатор узла
	log.Debug("lookup dag node id")
	nodeIdx, err := i.dag.Add(word)
	if err != nil {
		log.WithError(err).Error("get dag node id failed")
		return 0, errors.Join(ErrRegisterWord, err)
	}

	// получаем идентификатор слова
	log.Debugf("lookup word id, dag node id %d", nodeIdx)
	wordIdx, err := i.nodeToWord.Get(nodeIdx)
	switch {
	case err != nil:
		log.WithError(err).Debug("get word id failed, create new")

		wordIdx = node.NewID(N(i.words.Len()))
		if err = i.nodeToWord.Set(nodeIdx, *wordIdx); err != nil {
			log.WithError(err).Error("create word id failed")
			return 0, errors.Join(ErrRegisterWord, err)
		}
	case wordIdx == nil:
		log.WithError(err).Debug("get word id failed, wordIDx is nil")
		wordIdx = node.NewID(N(i.words.Len()))
		if err = i.nodeToWord.Set(nodeIdx, *wordIdx); err != nil {
			log.WithError(err).Error("create word id failed")
			return 0, errors.Join(ErrRegisterWord, err)
		}
	}

	// получаем текущий или создаём новый набор тегов для слова
	log.Debug("get or create word tag set list")
	wordTagSets, err := i.words.Get(wordIdx.Value())
	switch {
	case err != nil:
		log.WithError(err).Debug("get word tag set list failed, create new")
		wordTagSets = words.NewTagSets(tagSetIdx)
	case wordTagSets == nil:
		log.WithError(ErrTagSetNotFound).Error("tag set id is nil")
		return 0, fmt.Errorf("%w: %s", ErrTagSetNotFound, tagSetIdx)
	default:
		wordTagSets.Add(uint32(tagSetIdx))
	}

	// сохраняем в справочник слов набор тегов
	log.Debug("save word tag set list")
	if err = i.words.Set(wordIdx.Value(), *wordTagSets); err != nil {
		return 0, errors.Join(ErrRegisterWord, err)
	}

	log.Info("word registered")
	return int(nodeIdx), nil
}

func (i *Index[A, N, T, S]) Get(word string) ([][]*tag.Name, error) {
	// получаем идентификатор узла
	nodeIdx, err := i.dag.Get(word)
	if err != nil {
		return nil, errors.Join(ErrNotFound, err)
	}

	// получаем идентификатор слова
	wordIdx, err := i.nodeToWord.Get(nodeIdx)
	if err != nil {
		return nil, errors.Join(ErrNotFound, err)
	}

	// получаем набор тегов для слова
	wordTagSets, err := i.words.Get(wordIdx.Value())
	if err != nil {
		return nil, errors.Join(ErrNotFound, err)
	}

	// получаем наборы тегов для слова
	result := make([][]*tag.Name, 0)
	tagSetIterator := wordTagSets.Iterator()
	for {
		tagSetIdx := tagSetIterator.Next()
		tagsSet, err := i.tagsSets.GetTags(S(tagSetIdx))
		if err != nil {
			return nil, errors.Join(ErrNotFound, err)
		}
		result = append(result, tagsSet)
		if !tagSetIterator.HasNext() {
			break
		}
	}

	return result, nil
}

func (i *Index[A, N, T, S]) RegisterTag(t tag.Name) (int, error) {
	log := i.log.WithKeys(logging.Keys{"name": t})
	log.Info("registering tag")

	existedIdx, existed := i.tagsMap[t]
	if existed {
		log.Debug("tag existed")
		return int(existedIdx), nil
	}

	log.Debug("tag not existed, create new")
	newIdx, err := i.tags.Add(t)
	if err != nil {
		log.WithError(err).Error("create tag failed")
		return 0, err
	}

	i.tagsMap[t] = newIdx

	return int(newIdx), nil
}

func (i *Index[A, N, T, S]) TagsCount() int {
	return i.tags.Len()
}
