# Этап 11. Внутренний формат: TagSet + Paradigm + DAWG reader

## Содержание этапа

Создание фундамента нового внутреннего формата: примитивы для хранения
морфологических данных. Этот этап не зависит от текущего `internal/build`
и создаёт новый пакет `pkg/morphology/internal/`.

### TagSet (`tagset.go`)

Набор грамматических тегов для конкретного словаря/языка.

```go
type TagSet struct {
    Name     string
    Tags     []string          // id → name
    TagIndex map[string]uint16 // name → id
}
```

- Добавление тегов: `Add(name string) uint16` (дедупликация по имени)
- Поиск: `ID(name string) (uint16, bool)`, `Name(id uint16) string`
- Сериализация: JSON-массив строк (как в pymorphy2 gramtab)

### Paradigm (`paradigm.go`)

Шаблон склонения/спряжения. Плоский массив uint16:

```
[suffix_0, ..., suffix_N-1 | tag_0, ..., tag_N-1 | prefix_0, ..., prefix_N-1]
```

- Длина парадигмы = `len(data) / 3`
- Суффикс формы i: `Suffix(id, i) uint16`
- Тег формы i: `Tag(id, i) uint16`
- Префикс формы i: `Prefix(id, i) uint16`

### DAWG reader (`dawg.go`)

Чтение формата dawgdic (words.dawg из pymorphy2). Формат:
- `dictionary`: uint32 array. Каждый узел — uint32 с label (8 бит),
  offset (22 бита), leaf/extension флагами.
- `guide`: byte array. По 2 байта на узел: child + sibling.

Методы:
- `followByte(lbl byte, index uint32) uint32` — переход по байту
- `followRune(r rune, index uint32) uint32` — переход по руне (1–4 байта)
- `follow(s string, index uint32) uint32` — переход по строке
- `find(key string) uint32` — точный поиск → значение узла
- `valuesForIndex(index uint32) [][]byte` — все значения (обход suffixed values)

### CharPolicy (`char_policy.go`)

Набор подменяемых символов для поиска. По умолчанию пусто (нейтрально
к языку). Для русского: `[{from: 'е', to: 'ё'}]`.

```go
type CharPolicy struct {
    Substitutions []Substitution
}

type Substitution struct {
    From rune
    To   rune
}
```

### Similar items (`similar_items.go`)

Обход DAWG с учётом подмен символов. При обходе для каждого символа
из `CharPolicy.From` пробовать также `CharPolicy.To`. Реализация:
рекурсивный обход с «вилкой» на подменяемом символе.

### Dictionary (`dictionary.go`)

Иммутабельный снимок словаря:

```go
type Dictionary struct {
    Language    string
    TagSet      *TagSet
    Suffixes    []string
    Prefixes    []string
    Paradigms   []Paradigm
    Words       *DAWG
    Prediction  []*DAWG
    Probability *DAWG
    CharPolicy  *CharPolicy
}
```

## Проверка (тесты)

- unit-тест: DAWG reader — создание тестового DAWG в памяти, find/follow.
- unit-тест: similarItems с CharPolicy `е→ё` — проверка подмены.
- unit-тест: Paradigm — индексная арифметика по плоскому массиву.
- unit-тест: TagSet — Add/ID/Name, дедупликация.
- unit-тест: Dictionary — construction из компонентов.
- `go test ./pkg/morphology/... -race` — зелёные.

## Реализация (фактический API)

Пакет `pkg/morphology/internal` (package `internal`) — референс: формат
dawgdic/`opennota/morph` (fetch: dict.go, guide.go, dawg.go, completer.go).

Оформление от плана:

- Методы DAWG **экспортированы** (нужны importers/pymorphy2 в следующем пакете):
  `FollowByte`, `FollowRune`, `Follow`, `Find`, `HasValue`, `Value`,
  `ValuesForIndex`, `SimilarItems`. Внутренняя рекурсия — `similarItemsRecursive`.
- `TagSet`: поле `Name` и метод `TagName(id)` (метод `Name` в Go не может
  совпадать с полем).
- Конструкторы: `NewTagSet`, `NewParadigm(suffixes, tags, prefixes)`,
  `NewDAWG(dict, guide)`, `ReadDAWG(r io.Reader)` (формат words.dawg:
  `[uint32 n][n×uint32][uint32 g][g×2 байт]`), `NewCharPolicy(subs...)`,
  `RussianCharPolicy()`, `NewDictionary(language, tagSet, suffixes, prefixes,
  paradigms, words, charPolicy)`.
- Формат единицы словаря: bits 0–7 label, bit 8 has_leaf, bit 9 extension,
  bits 10–31 offset (у value-unit — значение, бит 31 isLeaf). Переход:
  `next = index ^ offset ^ label`.
- Payload-разделитель `payloadSeparator = 0x01`; значения в words.dawg —
  base64-закодированные байты записи (для `>HH` — 4 байта big-endian).
- Тесты: 19 unit-тестов включая независимый фикстурный билдер словаря+guide
  (`dawg_test.go: buildTestDAWG`), проверяющий reader «с другой стороны».
- Готово: `go test -race ./pkg/morphology/...` зелёный, `go vet` чист.
