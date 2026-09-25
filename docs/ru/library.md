# Программное использование библиотеки

Импорт:

```go
import "github.com/amarin/gomorphy/pkg/morphology"
```

Публичный API целиком лежит в пакете `morphology` — вложенный пакет
`morphology/internal` не экспортируется и не предназначен для прямого
использования.

Какой вызов решает какую задачу: [scenarios.md](scenarios.md).

Готовые самодостаточные программы для каждой точки входа ниже лежат в
[examples/](../../examples/README.md) — `go run ./examples/<name>`.
У большинства точек входа есть и функция `ExampleXxx` в
`pkg/morphology/example_test.go`, которую pkg.go.dev/godoc показывает
прямо на странице документации этой функции.

## Открытие словаря

Точки входа, все возвращают `*morphology.Dictionary`:

```go
func Open(path string) (*Dictionary, error)
func OpenBytes(data []byte) (*Dictionary, error)
func OpenPyMorphy(dir string) (*Dictionary, error)
func OpenPyMorphyDense(dir string) (*Dictionary, error)
func CompileFromXML(r io.Reader, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromXMLFile(path string, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromXMLDense(r io.Reader, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromXMLFileDense(path string, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromUniMorph(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFile(path string, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphDense(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFileDense(path string, opts UniMorphOptions) (*Dictionary, error)
```

- **`Open(path)`** — загружает уже скомпилированный единый формат
  (`.dat`, секции + mmap). Горячие секции (`words.dawg`) отображаются
  без копирования; результат нужно закрыть `Close()`.
- **`OpenBytes(data)`** — тот же формат, что `Open`, но из памяти
  (например, `//go:embed`). Контрольная сумма проверяется; `data` не
  копируется и должен жить дольше словаря; невыровненные секции
  копируются. Без mmap, поэтому работает и на Windows (`Open` там пока не
  поддерживается); `Close()` — no-op. См.
  [examples/embed](../../examples/embed/main.go).
  Сценарий: [scenarios.md](scenarios.md), п. 11.
- **`OpenPyMorphy(dir)`** — читает словарь pymorphy2 прямо из
  директории с исходными файлами (`words.dawg`, `paradigms.array`,
  ...), без предварительной компиляции в `.dat`.
- **`OpenPyMorphyDense(dir)`** — как `OpenPyMorphy`, но пересобирает
  `words.dawg` под плотный 1-байтовый алфавит (см.
  [implementation/pymorphy2-dense-alphabet.md](../en/implementation/pymorphy2-dense-alphabet.md)).
  Даёт те же разборы и те же результаты `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`,
  что `OpenPyMorphy`, но быстрее и компактнее в памяти. Проходит
  round-trip через `SaveTo`/`Open`, как любой другой словарь. Именно его
  по умолчанию использует `gomorphy build pymorphy` (см. [cli.md](cli.md));
  если нужен именно сырой вариант без плотного алфавита, вызывайте
  `OpenPyMorphy`.
- **`CompileFromXML`/`CompileFromXMLFile`** — компилирует словарь
  OpenCorpora из `dict.xml`. `progress` — необязательный callback
  `func(processed, total int)` для индикации хода компиляции (можно
  передать `nil`).
- **`CompileFromXMLDense`/`CompileFromXMLFileDense`** — как
  `CompileFromXML`/`CompileFromXMLFile`, но пересобирает `words.dawg`
  каждого шарда под один плотный 1-байтовый алфавит, общий для всего
  словаря (импорт OpenCorpora может дать несколько шардов, pymorphy2 —
  никогда). Те же гарантии, что у `OpenPyMorphyDense`: идентичные
  разборы, round-trip через `SaveTo`/`Open`. Именно его по умолчанию
  использует `gomorphy build opencorpora`.
- **`CompileFromUniMorph`/`CompileFromUniMorphFile`** — компилирует
  словарь UniMorph из TSV-файла `lemma<TAB>wordform<TAB>bundle`
  (например, `unimorph/rus`). Лемма и словоформа приводятся к нижнему
  регистру при чтении (с 1.2.0; `.dat` UniMorph, собранные 1.1.0 из строк
  в смешанном регистре, нужно пересобрать, чтобы они находились точным
  поиском); набор признаков (bundle) хранится как есть. `opts`
  (`UniMorphOptions`, алиас `importers/unimorph.Options`) содержит поля:
  - `Language` — обязательно; сейчас допустимо только `"ru"`.
  - `CharPolicy` — `nil` означает е→ё (русская политика по умолчанию);
    `NoCharPolicy()` или своя `NewCharPolicy(...)` её переопределяют.
  - `OnMalformed func(lineNumber int, text string)` — вызывается для
    каждой строки, где не ровно 3 поля через табуляцию; строка в любом
    случае пропускается (`nil` — пропускать молча).
  - `Progress` — необязательный callback прогресса; `SourceVersion` —
    записывается в `BuildInfo.SourceVersion`.

  См.
  [implementation/stage-16-import-unimorph.md](../en/implementation/stage-16-import-unimorph.md).
- **`CompileFromUniMorphDense`/`CompileFromUniMorphFileDense`** — как
  `CompileFromUniMorph`/`CompileFromUniMorphFile`, но пересобирает
  `words.dawg` каждого шарда под один плотный 1-байтовый алфавит, общий
  для всего словаря. Те же гарантии, что у `CompileFromXMLDense`:
  идентичные разборы, round-trip через `SaveTo`/`Open`. Именно его по
  умолчанию использует `gomorphy build unimorph`.

```go
d, err := morphology.Open(".data/opencorpora/opencorpora.dat")
if err != nil {
    log.Fatal(err)
}
defer d.Close()
```

`Dictionary.Close()` освобождает mmap-регион и является no-op для
словарей импортированных, собранных (`Builder`, `ImportTSV`, `Merge`)
или открытых через `OpenBytes` — они не используют mmap, так что
вызывать его можно безусловно. `Close` нельзя вызывать, пока другие
горутины ещё могут вызывать методы словаря (напрямую или через
`MultiDictionary`): незавершённые вызовы `Parse`/`Lemma`/`IsKnown`/`Fuzzy`/
`FuzzyTop`/`ContentHash` читают отображённую память, и снятие отображения
под ними роняет процесс с SIGSEGV или SIGBUS — это не перехватываемая
паника. Уже возвращённые значения (`Reading`, `LemmaRef`, `FuzzyMatch`,
`BuildInfo` и все их строки) — независимые копии и остаются валидными
после `Close`. Если словари подменяются во время работы, старый можно
закрывать только после завершения всех его текущих вызовов (например,
брать read-lock `sync.RWMutex` вокруг каждого вызова и write-lock перед
`Close`). После `Close` словарь использовать нельзя.

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

- `BuilderOptions{Language, Source, CharPolicy}` — `Language` используется
  конвейером сборки; `Source` заполняет `BuildInfo.Source` (по
  умолчанию `"builder"`).
- `AddForm`/`AddLemma` приводят `word`/`lemma`/`normal` к нижнему регистру
  (`strings.ToLower` — та же функция, что в `Parse`); `ImportTSV` так же
  приводит колонки `wordform` и `lemma`. Теги к нижнему регистру не
  приводятся никогда. Раньше форма, добавленная как «Москва», хранилась
  как есть и находилась только через приведение регистра в `Parse` плюс
  предсказание по окончанию — неотличимо от догадки; теперь это настоящая
  словарная запись, доступная как `Parse("москва")` или `Parse("Москва")`
  с `Predicted == false`.
- `AddForm`/`AddLemma` **не** обрезают пробелы (в отличие от `ImportTSV`,
  который обрезает каждое поле): `" кот "` сохраняется вместе с
  пробелами. Ошибкой считается только пустое или состоящее из одних
  пробелов `word`.
- `CharPolicy` — `nil` (по умолчанию) выбирает политику по `Language`: е→ё
  для `"ru"` и для пустого `Language` (= «ru»), для остальных языков —
  без замен. `NoCharPolicy()` отключает замены явно независимо от языка, а
  `NewCharPolicy(Substitution{From: 'и', To: 'і'})` задаёт свою политику.
  Политика хранится в словаре и применяется в `Parse`, `Lemma`, `IsKnown`,
  `Fuzzy` и `FuzzyTop`.
- Теги — непрозрачные строки, хранятся как есть и регистрируются
  автоматически как граммемы — без привязки к набору OpenCorpora.
- Пустая лемма делает словоформу леммой самой себе (auto-lemma).
- Повторяющиеся одинаковые записи `(word, lemma, tag)` дедуплицируются.
- `Builder` одноразовый: `Build()` его закрывает, даже если сама сборка
  не удалась (например, из-за недопустимой `CharPolicy`), — дальнейшие
  `AddForm`/`Build` возвращают `ErrBuilderClosed`. `ErrNoEntries`
  возвращается (без закрытия `Builder`), если `Build` вызван без
  зарегистрированных записей.

### `ImportTSV` — словоформы из TSV-потока или файла

```go
d, err := morphology.ImportTSV(r, morphology.BuilderOptions{Language: "ru"})
```

Читает строки `lemma<TAB>wordform[<TAB>tags]` из `io.Reader` по тем же
правилам, что и `Builder` (непрозрачные теги, auto-lemma, дедуп,
приведение колонок словоформы и леммы к нижнему регистру, та же
`CharPolicy` по умолчанию). Пустые строки и комментарии `#`
пропускаются; у каждого поля, включая теги, обрезаются пробелы по краям;
пустая словоформа или строка не из 2–3 колонок — ошибка с номером строки.
Длина строки ограничена 1 МиБ. Пустой поток или
поток из одних комментариев — ошибка, оборачивающая `ErrNoEntries`, как у
`Builder.Build` (до 1.2.1 возвращался пустой словарь). Вариант для файла не
предоставляется — оберните путь самостоятельно.

Сценарий: [scenarios.md](scenarios.md), п. 5.

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
Сценарий: [scenarios.md](scenarios.md), п. 6.

### Получение исходных данных

Для CLI-утилиты (`gomorphy download`/`unpack`/`update`, см.
[cli.md](cli.md)) загрузку исходников делают `pkg/opencorpora.Loader`,
`pkg/pymorphy.Loader` и `pkg/unimorph.Loader` — тот же API доступен и из
библиотеки:

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

`unimorph.Loader` — то же самое для TSV UniMorph, но `NewLoader`
возвращает ошибку (неподдерживаемый `language` отклоняется сразу, а не
всплывает позже из `Sync`):

```go
loader, err := unimorph.NewLoader("ru", "") // "" — путь по умолчанию, .data/unimorph/ru
if err != nil {
    log.Fatal(err)
}
if err := loader.Sync(false); err != nil {
    log.Fatal(err)
}
d, err := morphology.CompileFromUniMorphFile(loader.UnpackedFilePath(), morphology.UniMorphOptions{Language: "ru"})
```

`Sync(false)` спрашивает источник, есть ли новый выпуск, скачивает его и
распаковывает поверх прежней копии; если источник недоступен, но архив
уже лежит на диске, работа продолжается с ним. `Sync(true)` не делает
сетевых запросов и только распаковывает уже скачанный архив. Скачивание и
распаковка пишут во временный файл (для pymorphy2 — каталог) и заменяют
прежний только при успехе, так что сбой или обрыв загрузки сохраняет
старые данные; файлы создаются в каталоге данных загрузчика, который
создаётся при необходимости. (До 1.2.1 новая загрузка не распаковывалась
поверх существующей копии, свой каталог данных игнорировался при создании
каталогов, `Sync(true)` всё равно обращался к источнику, а OpenCorpora
сохранял страницу с HTTP-ошибкой как архив.)

Логирование: загрузчики не требуют настройки. `NewLoader` в момент вызова
проверяет, настроен ли общий логгер процесса через `logging.Init`
(`github.com/amarin/logging`): если да, загрузчик пишет в него, как
раньше; если нет — получает логгер, который всё отбрасывает
(`common.NewLoaderLogger`); до 1.2.1 в этом случае была паника.
`logging.Init`, вызванный позже, на уже созданный загрузчик не влияет;
чтобы использовать свой логгер, присвойте его экспортируемому полю
`Logger` загрузчика.

Сценарий: [scenarios.md](scenarios.md), п. 15.

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

Сценарий: [scenarios.md](scenarios.md), п. 1.

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
    // Predicted — true, если разбор получен предсказанием по окончанию
    // (слова нет в словаре), false — для словарного разбора.
    Predicted bool
}
```

Формат `Tag` зависит от источника словаря, который определяется именем
набора тегов — `Dictionary.TagSetName()` или
`MultiDictionary.DictTagSetName(i)` для индекса `Dict` разбора:
`"opencorpora"` (comma-joined, из `dict.xml`), `"opencorpora-int"`
(pymorphy2, свой синтаксис — те же граммемы, но иначе сериализованные) и
`"unimorph"` (наборы признаков UniMorph, например `N;ACC;SG`). Словари
`Builder` и `ImportTSV` сообщают `"builder"` и `"tsv"`: их теги — строки
самого вызывающего, которых `pkg/morphology/tagmap` не знает
(`tagmap.Known` — false); `Merge` сохраняет имя базы. Для сравнения тегов
между словарями разного происхождения см.
[implementation/tag-mapping.md](../en/implementation/tag-mapping.md)
(`pkg/morphology/tagmap`) и [examples/tagmap](../../examples/tagmap/main.go).
Сценарий: [scenarios.md](scenarios.md), п. 13.

Вход приводится к нижнему регистру автоматически.

### Словарные слова и предсказания

`Parse` при отсутствии слова в словаре подставляет предсказание по
окончанию, и предсказанный `Reading` выглядит как любой другой — те же
поля, только `Predicted == true`. Чтобы различить их или проверить
принадлежность слова словарю, не платя за предсказание, используйте
`Predicted` или `IsKnown`:

```go
d.IsKnown("кота")            // true: есть в словаре
d.IsKnown("бота")            // false, хотя Parse("бота") может предсказать разборы
d.Parse("бота")[0].Predicted // true
```

`IsKnown(word)` сообщает, есть ли у `word` (приведённого к нижнему
регистру, с применённой `CharPolicy` — тот же поиск, что делает `Parse`)
хотя бы один словарный разбор; никогда не предсказывает.
`MultiDictionary.IsKnown` — `true`, если слово знает хотя бы один словарь
набора. Поиск именованных сущностей по словарю на основе `IsKnown` — в
[examples/ner](../../examples/ner/main.go).

Сценарии: [scenarios.md](scenarios.md), п. 3 и 4.

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

Сценарий: [scenarios.md](scenarios.md), п. 2.

Тип `LemmaRef`:

```go
type LemmaRef struct {
    Normal string // начальная форма
    Tag    string // тег начальной формы (форма 0 парадигмы)
    Para   uint16
    Shard  int
    Dict   int
    // Predicted — true, если все разборы за этой леммой предсказаны.
    Predicted bool
}
```

## Нечёткий поиск

`Fuzzy` возвращает все слова словаря в пределах расстояния Левенштейна
`maxDist` (по рунам; применяется `CharPolicy` словаря — для русского «е»
в запросе совпадает с хранимой «ё» на расстоянии 0, в одну сторону, как
в `Parse`; хранимая «е» против «ё» в запросе по-прежнему стоит 1).
`FuzzyTop` возвращает `maxWords` ближайших слов, итеративно расширяя
расстояние; `maxWords ≤ 0` означает только совпадения на расстоянии 0,
как `Fuzzy(word, 0)`, — само слово и его варианты по `CharPolicy`
(например, «ёлка» для «елка»). Оба возвращают `[]FuzzyMatch`,
отсортированный по (расстояние, слово), без дублей слова. Запрос
приводится к нижнему регистру, как вход `Parse`:

```go
matches := d.Fuzzy("кот", 1)
top := d.FuzzyTop("кот", 5)
```

```go
type FuzzyMatch struct {
    Word     string // слово как в словаре (с «ё»)
    Distance int    // расстояние Левенштейна до запроса, в рунах
    Dict     int
}
```

Для словарей с плотным алфавитом (конструкторы `…Dense` и любой результат
`Builder`, `ImportTSV` и `Merge`) работает так же — те же совпадения, что
у такого же словаря без плотного алфавита. См.
[examples/typos](../../examples/typos/main.go).
Сценарии: [scenarios.md](scenarios.md), п. 8 (опечатки) и 9 (е/ё).

## Диагностические метаданные

`Info()` возвращает секцию `info` словаря (когда и чем он собран). Все
импортёры, `Builder`, `ImportTSV` и `Merge` заполняют как минимум
`Source`; `BuiltAt` и `LibraryVersion` проставляет `SaveTo` (до
сохранения они нулевые). `nil` возвращается только для nil-словаря или
файла, записанного до появления секции `info`:

```go
type BuildInfo struct {
    BuiltAt        time.Time // проставляет SaveTo
    LibraryVersion string    // проставляет SaveTo: Version, записавшая файл
    Source         string    // "pymorphy2" / "opencorpora" / "unimorph" / "builder" / "tsv" / "merge"
                             // (или BuilderOptions.Source)
    SourceVersion  string    // версия исходных данных, если известна
    Author         string
    Description    string
    SourceURL      string
}
```

`Language()` возвращает код языка словаря (`"ru"` для всех встроенных
импортёров; `BuilderOptions.Language` для `Builder`/`ImportTSV`, `"ru"`,
если пусто). `TagSetName()` возвращает имя набора тегов (см. «Точный
поиск словоформы» выше).

`ContentHash()` возвращает стабильный hex-дайджест (xxh3-128, 32 hex-символа
в нижнем регистре) секций содержимого словаря без `info` (`BuiltAt`,
`LibraryVersion`, `Source`…) — поэтому пересохранение неизменённого
словаря или открытие через `Open`/`OpenBytes` сохраняют дайджест, а два
словаря с равным `ContentHash` разбирают любое слово одинаково. Дайджест
описывает кодирование, а не только смысл: те же слова с другим алфавитом
или `CharPolicy` дают другой хеш. Первый вызов кодирует все секции (для
большого словаря — копия его DAWG слов); результат кешируется, так что
последующие вызовы дешёвые. Возвращает `""` для nil-словаря (или при
внутренней ошибке кодирования, которую сегодня не вызывает ни один
словарь, собранный этой библиотекой). См.
[examples/contenthash](../../examples/contenthash/main.go).
Сценарий: [scenarios.md](scenarios.md), п. 12.

## Несколько словарей одновременно: `MultiDictionary`

`MultiDictionary` агрегирует `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`/`IsKnown`/
`Close` по произвольному набору уже открытых словарей — сам он ничего не
открывает. Подробный дизайн:
[implementation/multi-dict.md](../en/implementation/multi-dict.md).
Сценарий: [scenarios.md](scenarios.md), п. 7.

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
- `Len()` — число словарей в наборе.
- `DictInfo(i)` — `BuildInfo` словаря с индексом `i` (`nil` для
  индекса вне диапазона или словаря без секции `info`).
- `DictTagSetName(i)` — имя набора тегов словаря с индексом `i` (`""` для
  индекса вне диапазона); типичное использование:
  `tagmap.Map(m.DictTagSetName(r.Dict), r.Tag)`.
- `IsKnown(word)` — `true`, если слово знает хотя бы один словарь набора
  (никогда не предсказывает).
- `Close()` закрывает каждый словарь набора, объединяя ошибки через
  `errors.Join`. Действует то же правило, что и для `Dictionary.Close`:
  ни на наборе, ни на его словарях не должно быть незавершённых вызовов.

## Параллельная работа

Чтение открытого `*Dictionary` потокобезопасно — несколько горутин
могут одновременно вызывать `Parse`/`Lemma`/`IsKnown`/`Fuzzy`/`FuzzyTop`/
`ContentHash`/`Info`/`Language`/`TagSetName` (и те же методы
`MultiDictionary`), но никогда — одновременно с `Close` (см. «Открытие
словаря» выше):

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

`SaveTo` сохраняет словарь (в т.ч. собранный из XML, загруженный через
`OpenPyMorphy` или пересобранный через `OpenPyMorphyDense`) в единый
формат:

```go
d, err := morphology.CompileFromXMLFile("dict.xml", nil)
if err != nil {
    log.Fatal(err)
}
if err := d.SaveTo("opencorpora.dat"); err != nil {
    log.Fatal(err)
}
```

Словарь с плотным алфавитом (конструкторы `…Dense`, `Builder`,
`ImportTSV`, `Merge`) проходит round-trip через `SaveTo`/`Open`, как любой
другой: кодек алфавита записывается в собственную секцию `.dat` и
восстанавливается при `Open`.

## Ошибки

Пакет определяет четыре экспортируемые сигнальные ошибки:

```go
var (
    ErrNoEntries                // Builder.Build без зарегистрированных записей;
                                 // оборачивается ImportTSV (нет строк) и Merge
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
