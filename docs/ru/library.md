# Программное использование библиотеки

Импорт:

```go
import "github.com/amarin/gomorphy/pkg/morphology"
```

Публичный API целиком лежит в пакете `morphology` — вложенный пакет
`morphology/internal` не экспортируется и не предназначен для прямого
использования.

## Открытие словаря

Четыре точки входа, все возвращают `*morphology.Dictionary`:

```go
func Open(path string) (*Dictionary, error)
func OpenPyMorphy(dir string) (*Dictionary, error)
func OpenPyMorphyDense(dir string) (*Dictionary, error)
func CompileFromXML(r io.Reader, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromXMLFile(path string, progress opencorpora.Progress) (*Dictionary, error)
```

- **`Open(path)`** — загружает уже скомпилированный единый формат
  (`.dat`, секции + mmap). Горячие секции (`words.dawg`) отображаются
  без копирования; результат нужно закрыть `Close()`.
- **`OpenPyMorphy(dir)`** — читает словарь pymorphy2 прямо из
  директории с исходными файлами (`words.dawg`, `paradigms.array`,
  ...), без предварительной компиляции в `.dat`.
- **`OpenPyMorphyDense(dir)`** — как `OpenPyMorphy`, но пересобирает
  `words.dawg` под плотный 1-байтовый алфавит (см.
  [implementation/pymorphy2-dense-alphabet.md](../en/implementation/pymorphy2-dense-alphabet.md)).
  Даёт те же разборы, что `OpenPyMorphy`, но быстрее и компактнее в
  памяти; словарь с плотным алфавитом **нельзя** сохранить через
  `SaveTo` (см. ниже) и не поддерживает `Fuzzy`/`FuzzyTop` (возвращают
  `nil`).
- **`CompileFromXML`/`CompileFromXMLFile`** — компилирует словарь
  OpenCorpora из `dict.xml`. `progress` — необязательный callback
  `func(processed, total int)` для индикации хода компиляции (можно
  передать `nil`).

```go
d, err := morphology.Open(".data/opencorpora/opencorpora.dat")
if err != nil {
    log.Fatal(err)
}
defer d.Close()
```

`Dictionary.Close()` — no-op для словарей, открытых не через `Open`
(`OpenPyMorphy*`/`CompileFromXML*` не используют mmap), вызывать можно
безусловно.

### Получение исходных данных

Для CLI-утилиты (`gomorphy download`/`unpack`/`update`, см.
[cli.md](cli.md)) загрузку исходников делают `pkg/opencorpora.Loader` и
`pkg/pymorphy.Loader` — тот же API доступен и из библиотеки:

```go
loader := pymorphy.NewLoader("") // "" — путь по умолчанию, .data/pymorphy
if err := loader.Sync(false); err != nil { // false — не пропускать скачивание
    log.Fatal(err)
}
d, err := morphology.OpenPyMorphy(loader.UnpackedDirPath())
```

`opencorpora.Loader` — то же самое для `dict.xml`
(`loader.UnpackedFilePath()` вместо `UnpackedDirPath()`, дальше —
`morphology.CompileFromXMLFile`).

## Сборка словаря с нуля

Конструкторы выше компилируют уже существующий источник (OpenCorpora
XML, pymorphy2, UniMorph). Чтобы собрать словарь из собственных
словоформ — тематический словарь, список имён, небольшой
пользовательский лексикон — используйте `Builder`, `ImportTSV` или
`Merge`. Все три дают тот же внутренний формат, что и
`CompileFrom*Dense` (плотный 1-байтовый алфавит), и проходят
round-trip через `SaveTo`/`Open`. `Builder` и `ImportTSV` пересобирают
prediction из своих собственных слов; `Merge`, наоборот, сохраняет
prediction базового словаря (см. «`Merge` — объединение
скомпилированных словарей» ниже).

### `Builder` — регистрация словоформ программно

```go
b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
b.AddLemma("кот", "NOUN,anim,masc,sing,nomn")           // начальная форма -> сама себе
b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent")    // словоформа -> лемма
d, err := b.Build()
```

- `BuilderOptions{Language, Source}` — `Language` используется
  конвейером сборки; `Source` заполняет `BuildInfo.Source` (по
  умолчанию `"builder"`).
- Теги — непрозрачные строки, хранятся как есть и регистрируются
  автоматически как граммемы — без привязки к набору OpenCorpora.
- Пустая лемма делает словоформу леммой самой себе (auto-lemma).
- Повторяющиеся одинаковые записи `(word, lemma, tag)` дедуплицируются.
- `Builder` одноразовый: `Build()` его закрывает. `ErrNoEntries`
  возвращается, если `Build` вызван без зарегистрированных записей.

### `ImportTSV` — словоформы из TSV-потока или файла

```go
d, err := morphology.ImportTSV(r, morphology.BuilderOptions{Language: "ru"})
```

Читает строки `lemma<TAB>wordform[<TAB>tags]` из `io.Reader` по тем же
правилам, что и `Builder` (непрозрачные теги, auto-lemma, дедуп).
Пустые строки и комментарии `#` пропускаются, поля обрезаются, ошибки
строк указывают номер строки. Вариант для файла не предоставляется —
оберните путь самостоятельно.

### `Merge` — объединение скомпилированных словарей

```go
merged, err := morphology.Merge(base, overlays, morphology.MergeAdd)

merged, err = morphology.MergeWithOptions(base, overlays, morphology.MergeOptions{
    Mode:              morphology.MergeReplace,
    RebuildPrediction: true,
})
```

Объединяет уже скомпилированные словари (например, базовый `.dat` плюс
overlay-словари), не изменяя входные данные. Слияние структурное: база
сохраняет свои парадигмы, имя набора тегов (так что
`pkg/morphology/tagmap` продолжает работать), вероятности и prediction
для слов вне словаря; слова overlay добавляются в эту структуру.

- `MergeAdd` (значение `MergeOptions.Mode` по умолчанию) — слово
  overlay, которое уже есть в базе (или в более раннем overlay),
  пропускается; новые слова добавляются.
- `MergeReplace` — разборы слова из overlay заменяют существующие
  разборы этого слова; при нескольких overlay побеждает последний.
- `MergeOptions.RebuildPrediction` — пересобрать prediction из всех
  объединённых слов (полезно при слиянии тематических словарей друг с
  другом); по умолчанию сохраняется prediction базы, и слова overlay в
  него не попадают. Требует результат с одним шардом (иначе
  `ErrPredictionSharded`).
- Входные словари должны совпадать по языку в точности (словарь с
  пустым языком отклоняется против базы `"ru"`); два разных известных
  словаря тегов (например, `opencorpora-int` и `unimorph`) также
  отклоняются (`ErrIncompatibleDictionaries`).
- Выходной `BuildInfo.Source` — `"merge"`; язык базы, `SourceVersion`
  и `Description` переносятся без изменений.

См. ExampleMerge и ExampleMergeWithOptions.

## Точный поиск словоформы

`Parse` возвращает все грамматические разборы слова, отсортированные
по вероятности (убыванию), либо `nil`, если слово не найдено ни точно,
ни предсказанием по окончанию:

```go
readings := d.Parse("кота")
for _, r := range readings {
    fmt.Printf("%s -> %s (%s)\n", r.Word, r.Normal, r.Tag)
}
// кота -> кот (NOUN,anim,masc,sing,gent)
// кота -> кот (NOUN,anim,masc,sing,accs)
```

Тип `Reading`:

```go
type Reading struct {
    Word   string  // словоформа как в словаре (с «ё»)
    Normal string  // начальная форма (лемма)
    Tag    string  // граммемный тег, например "NOUN,anim,masc,sing,nomn"
    Para   uint16  // id парадигмы — уникален только вместе с Shard
    Form   uint16  // индекс формы в парадигме
    Shard  int     // индекс шарда словаря; всегда 0 для нешардированных словарей
    Dict   int     // индекс словаря в MultiDictionary; всегда 0 для Dictionary.Parse напрямую
    Prob   float64 // вероятность разбора (0, если probability недоступен)
}
```

Формат `Tag` зависит от источника словаря: `TagSet.Name` различает
`"opencorpora"` (comma-joined, из `dict.xml`) и `"opencorpora-int"`
(pymorphy2, свой синтаксис) — оба описывают один и тот же набор
граммем, но по-разному сериализованы. Для сравнения тегов между
словарями разного происхождения см.
[implementation/tag-mapping.md](../en/implementation/tag-mapping.md)
(`pkg/morphology/tagmap`).

Вход приводится к нижнему регистру автоматически.

## Начальные формы (леммы)

`Lemma` возвращает начальную форму (лемму) и её собственный тег для
каждого омонима слова:

```go
lemmas := d.Lemma("кота")
for _, l := range lemmas {
    fmt.Printf("%s (%s)\n", l.Normal, l.Tag)
}
// кот (NOUN,anim,masc,sing,nomn)
```

Тип `LemmaRef`:

```go
type LemmaRef struct {
    Normal string // начальная форма
    Tag    string // тег начальной формы (форма 0 парадигмы)
    Para   uint16
    Shard  int
    Dict   int
}
```

## Нечёткий поиск

`Fuzzy` — все слова словаря в пределах расстояния Левенштейна
`maxDist` (по рунам; «е»/«ё» считаются одной заменой). `FuzzyTop` —
`maxWords` ближайших слов, с итеративным расширением расстояния.
Оба возвращают `[]FuzzyMatch`, отсортированный по (расстояние, слово),
без дублей слова:

```go
matches := d.Fuzzy("кот", 1)
top := d.FuzzyTop("кот", 5)
```

```go
type FuzzyMatch struct {
    Word     string
    Distance int
    Dict     int
}
```

Не работает (возвращает `nil`) для словарей с плотным алфавитом
(`OpenPyMorphyDense`) — см. «Открытие словаря» выше.

## Диагностические метаданные

`Info()` возвращает секцию `info` файла (когда и чем собран словарь),
или `nil`, если её нет (словари без `SaveTo`, либо собранные до
появления секции):

```go
type BuildInfo struct {
    BuiltAt        time.Time
    LibraryVersion string
    Source         string // "pymorphy2" / "opencorpora"
    SourceVersion  string
    Author         string
    Description    string
    SourceURL      string
}
```

## Несколько словарей одновременно: `MultiDictionary`

`MultiDictionary` агрегирует `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`/`Close`
по произвольному набору уже открытых словарей — сам он ничего не
открывает. Подробный дизайн:
[implementation/multi-dict.md](../en/implementation/multi-dict.md).

```go
oc, _ := morphology.Open("opencorpora.dat")
pm, _ := morphology.OpenPyMorphy(".data/pymorphy/data")
m := morphology.NewMultiDictionary(oc, pm)
defer m.Close()

for _, r := range m.Parse("кота") {
    fmt.Printf("dict#%d: %s -> %s (%s)\n", r.Dict, r.Word, r.Normal, r.Tag)
}
```

- `Reading.Dict`/`LemmaRef.Dict`/`FuzzyMatch.Dict` — индекс словаря в
  порядке, переданном в `NewMultiDictionary` (0-based).
- `Parse`/`Lemma`/`Fuzzy` — конкатенация результатов всех словарей, где
  слово нашлось, в порядке регистрации, без приоритетов и дедупа между
  словарями (каждый словарь уже дедуплицирует сам себя).
- `FuzzyTop(word, maxWords)` — единственный метод, где `maxWords`
  ограничивает **общий** результат, а не результат на каждый словарь:
  берёт top-`maxWords` от каждого словаря, затем сортирует и обрезает
  весь набор заново.
- `DictInfo(i)` — `BuildInfo` словаря с индексом `i` (`nil` для
  индекса вне диапазона или словаря без секции `info`).
- `Close()` закрывает каждый словарь набора, объединяя ошибки через
  `errors.Join`.

## Параллельная работа

Чтение открытого `*Dictionary` потокобезопасно — несколько горутин
могут одновременно вызывать `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`:

```go
var wg sync.WaitGroup
for _, word := range words {
    wg.Add(1)
    go func(w string) {
        defer wg.Done()
        readings := d.Parse(w)
        // обработка readings
    }(word)
}
wg.Wait()
```

## Сохранение на диск

`SaveTo` сохраняет словарь (в т.ч. собранный из XML или загруженный
через `OpenPyMorphy`) в единый формат:

```go
d, err := morphology.CompileFromXMLFile("dict.xml", nil)
if err != nil {
    log.Fatal(err)
}
if err := d.SaveTo("opencorpora.dat"); err != nil {
    log.Fatal(err)
}
```

Словарь с плотным алфавитом (`OpenPyMorphyDense`) сохранить нельзя —
`SaveTo` возвращает ошибку (см. «Открытие словаря» выше).

## Ошибки

Пакет определяет четыре экспортируемые сигнальные ошибки:

```go
var (
    ErrNoEntries                // Builder.Build без зарегистрированных записей
    ErrBuilderClosed            // Builder использован после Build
    ErrIncompatibleDictionaries // Merge: язык или набор тегов overlay
                                 // нельзя разделить с базой
    ErrPredictionSharded        // MergeWithOptions: задан RebuildPrediction,
                                 // но у объединённого словаря больше
                                 // одного шарда
)
```

Проверяйте их через `errors.Is`:

```go
d, err := b.Build()
if errors.Is(err, morphology.ErrNoEntries) {
    // ничего не зарегистрировано
}
```

Остальные функции возвращают обёрнутые (`fmt.Errorf("...: %w", err)`)
ошибки нижележащего слоя (файловая система, разбор формата и т.п.).
Проверяйте `err != nil` и, если нужно отличить конкретную причину,
разворачивайте через `errors.Is`/`errors.As` на известные ошибки
стандартной библиотеки (например, `os.ErrNotExist`).
