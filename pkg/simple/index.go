package simple

const cyrillicLetters = "абвгдеёжзийклмнопрстуфхцчшщьыъэюя"

// Index реализует управление и взаимодействие с индексом графа.
//type Index struct {
//	storage.Config
//	mu         *sync.RWMutex
//	alphabet   *alphabet.Alphabet[uint8]
//	dag        *node.Graph[uint8, uint16]
//	tagIndexer *indexer.IndexOf[uint8, tag.Tag, *tag.Tag]
//	tagSets    *tagset.Set[uint8, uint16]
//	words      indexer.IndexOf[uint16, node.ID[uint16], *node.ID[uint16]]
//	log        logging.Logger
//}

//// New создаёт новый экземпляр индекса.
//func New() *Index {
//	a := alphabet.New[uint8]()
//	// загружаем основной алфавит
//	if err := a.Reset(cyrillicLetters); err != nil {
//		panic(err)
//	}
//
//	tagsIndex := tag.NewIndexer()
//
//	i := &Index{
//		dag:        node.NewGraph[uint8, uint16](a),
//		alphabet:   a,
//		tagIndexer: tagsIndex,
//		tagSets:    tagset.NewSet(tagsIndex),
//		mu:         new(sync.RWMutex),
//		log:        logging.NewNamedLogger(indexName),
//	}
//
//	return i
//}
//
//// WordsCount возвращает количество слов в индексе.
//func (i *Index) WordsCount() int {
//	return 0
//}
//
//// NodesCount возвращает количество узлов в индексе.
//func (i *Index) NodesCount() int {
//	return i.dag.NodesCount()
//}
//
//// Optimize выполняет внутреннюю оптимизацию индекса.
//func (i *Index) Optimize() {}
//
//// Add добавляет слово и его характеристики в индекс.
//func (i *Index) Add(word string, _ ...any) (int, error) {
//	idx, err := i.dag.Add(word)
//	if err != nil {
//		return 0, err
//	}
//
//	return int(idx), nil
//}
//
//func (i *Index) RegisterTag(t tag.Tag) (int, error) {
//	ti, err := i.tagIndexer.Add(t)
//	if err != nil {
//		return 0, err
//	}
//
//	return int(ti), nil
//}
//
//func (i *Index) RemoveTagSet(t ...tag.Tag) (int, error) {
//	tagsIndexes := make([]S, le)
//}
