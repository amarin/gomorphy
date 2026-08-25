# Программное использование библиотеки

Импорт:

```go
import "github.com/amarin/gomorphy/pkg/dictionary"
```

## Подключение и получение данных словаря

### Открытие скомпилированного словаря

Словарь загружается через mmap и готов к использованию немедленно:

```go
d, err := dictionary.Open(".data/opencorpora/opencorpora.dict")
if err != nil {
    log.Fatal(err)
}
defer d.Close()
```

### Точный поиск словоформы

`Lookup` возвращает все грамматические разборы заданного слова:

```go
forms, err := d.Lookup("кота")
for _, f := range forms {
    fmt.Printf("%s %s lemma#%d\n", f.Text, f.Ancode, f.LemmaID)
}
// кота NOUN,anim,masc,sing,gent lemma#140411
// кота NOUN,anim,masc,sing,accs lemma#140411
```

Тип `Wordform`:

```go
type Wordform struct {
    Text      string   // словоформа
    Ancode    string   // полный грамматический разбор
    Grammemes []string // разложенные граммемы
    LemmaID   uint32   // идентификатор леммы
}
```

### Начальные формы (леммы)

`Lemmas` находит начальную форму и базовые граммемы:

```go
lemmas, err := d.Lemmas("кота")
for _, l := range lemmas {
    fmt.Printf("#%d %s %s\n", l.ID, l.Text, strings.Join(l.Grammemes, ","))
}
// #140411 кот NOUN,anim,masc
```

Тип `LemmaRef`:

```go
type LemmaRef struct {
    ID        uint32
    Text      string
    Grammemes []string
}
```

### Нечёткий поиск

`Fuzzy` — поиск слов с расстоянием Левенштейна до `maxDist`:

```go
matches, err := d.Fuzzy("кот", 1)
for _, m := range matches {
    fmt.Printf("%d %s\n", m.Distance, m.Text)
}
// 0 кот
// 1 код
// 1 крот
```

`FuzzyTop` — N ближайших слов по расстоянию:

```go
top, err := d.FuzzyTop("кот", 5)
// 0  кот
// 1  бот
// 1  вот
// ...
```

Тип `FuzzyMatch`:

```go
type FuzzyMatch struct {
    Text     string
    Distance int
}
```

### Параллельная работа

Чтение словаря потокобезопасно. Несколько горутин могут одновременно вызывать Lookup/Lemmas/Fuzzy:

```go
var wg sync.WaitGroup
for _, word := range words {
    wg.Add(1)
    go func(w string) {
        defer wg.Done()
        forms, _ := d.Lookup(w)
        // обработка forms
    }(word)
}
wg.Wait()
```

### Компиляция из XML

Словарь можно собрать из `dict.xml` без использования `opencorpora_update`:

```go
d, err := dictionary.CompileFromXMLFile("path/to/dict.xml")
if err != nil {
    log.Fatal(err)
}
defer d.Close()

err = d.SaveTo("output.dict")
```

### Ошибки

| Ошибка | Описание |
|--------|----------|
| `ErrClosed` | словарь закрыт (после `Close()`) |
| `ErrNotFound` | слово не найдено в словаре |
| `ErrInvalidMaxDist` | отрицательное значение maxDist |
| `ErrInvalidMaxWords` | отрицательное значение maxWords |

---

## Создание собственных словарей

Библиотека позволяет программно создавать словари без XML-файла.

### Базовый пример

```go
b := dictionary.NewBuilder()

// Регистрация граммем
b.AddGrammeme("NOUN")
b.AddGrammeme("anim")
b.AddGrammeme("masc")
b.AddGrammeme("sing")
b.AddGrammeme("nomn")
b.AddGrammeme("gent")
b.AddGrammeme("accs")

// Добавление леммы с начальной формой и базовыми граммемами
lemmaID, _ := b.AddLemma("кот", "NOUN", "anim", "masc")

// Добавление словоформ
b.AddForm(lemmaID, "кот", "NOUN", "anim", "masc", "sing", "nomn")
b.AddForm(lemmaID, "кота", "NOUN", "anim", "masc", "sing", "gent")
b.AddForm(lemmaID, "кота", "NOUN", "anim", "masc", "sing", "accs")

// Компиляция в иммутабельный словарь
d := b.Compile()
defer d.Close()

// Использование
forms, _ := d.Lookup("кота")
// [{кота NOUN,anim,masc,sing,gent ...} {кота NOUN,anim,masc,sing,accs ...}]
```

### API Builder

```go
func dictionary.NewBuilder() *Builder
```

Создаёт новый пустой Builder.

#### `AddGrammeme`

```go
func (b *Builder) AddGrammeme(name string) (uint32, error)
```

Регистрирует граммему и возвращает её ID. Повторный вызов с тем же именем возвращает тот же ID.

#### `AddLemma`

```go
func (b *Builder) AddLemma(text string, grammemes ...string) (int, error)
```

Добавляет лемму (начальную форму). `grammemes` — базовые граммемы леммы (POS и др.). Возвращает ID леммы для передачи в `AddForm`.

#### `AddForm`

```go
func (b *Builder) AddForm(lemma int, text string, grammemes ...string) error
```

Добавляет словоформу, привязанную к лемме. `grammemes` — полные граммемы словоформы (включая падеж, число и т.д.).

#### `Compile`

```go
func (b *Builder) Compile() *Dictionary
```

Компилирует наполненный Builder в иммутабельный Dictionary. После вызова Builder использовать нельзя.

### Сохранение на диск

Скомпилированный словарь можно сохранить и загрузить позже:

```go
d := b.Compile()
defer d.Close()

// Атомарная запись (temp-файл + rename)
err := d.SaveToAtomic("my.dict")
if err != nil {
    log.Fatal(err)
}

// Загрузка
d2, err := dictionary.Open("my.dict")
```

### Изменение существующего словаря

Можно получить Builder от существующего словаря, добавить данные и перекомпилировать:

```go
d, _ := dictionary.Open("existing.dict")
b, _ := d.Builder()

b.AddLemma("новое_слово", "NOUN")
b.AddForm(/* ... */)

d2 := b.Compile()
d2.SaveTo("extended.dict")
```

### Примечания

- Формат файла един для словарей из XML и программно созданных.
- Builder непотокобезопасен — используйте из одной горутины.
- Чтение Dictionary потокобезопасно после компиляции.
