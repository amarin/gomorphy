# Реализация gomorphy (as-is)

Документ описывает реализацию в формулировках свершившегося факта.
Соответствует требованиям `docs/requirements.md` и плану из `docs/todo.md`.

## Состав репозитория

```
pkg/dictionary    публичный фасад библиотеки (FT7–FT9)
internal/build    Builder: сборка словаря из XML или программно, CSR-структуры
internal/format   формат файла: header, каталог секций, varint/delta-кодирование
internal/xmlscan  байтовый сканер dict.xml без аллокаций на токен
internal/intern   таблица интернирования строк (open addressing)
internal/stringsx строковая арена + таблица смещений
internal/mmapx    mmap-ридер секций
pkg/opencorpora   загрузчик/распаковщик OpenCorpora (без изменений по поведению)
cmd/opencorpora_update CLI: обновить, распаковать, скомпилировать
cmd/gomorphy      CLI: точный поиск, леммы, нечёткий поиск по .dat
```

## Модель данных в памяти

Все сущности нормализованы в справочники; словоформа — пара идентификаторов.

### Справочники
- `grammemes []string` — имена граммем, id = индекс (uint8).
- Анкоды: CSR `ancodeOff []uint32` + `ancodeGrams []uint8` — 876 уникальных
  наборов граммем, id = uint16. Ключ набора при сборке — отсортированная
  последовательность id граммем.

### Тексты
- `textsArena []byte` — все уникальные тексты подряд;
- `textOff []uint32` — N+1 границ; текст i — срез `[textOff[i], textOff[i+1])`.
Интернирование: хеш байтового среза (без создания string) → open-addressing
таблица `(hash, offset, len)` → при промахе append в арену.

### Леммы и строки (CSR)
- `lemmaText []uint32`, `lemmaRowOff []uint32` (L+1);
- строки всех лемм подряд: `rowText []uint32`, `rowAncode []uint16`.
Строка = текст + анкод; `<l>`-запись является первой строкой леммы.

### Префиксный индекс (CSR-trie)
- `stateOff []uint32` (S+1), переходы `transitions []{char uint8; next uint32}`
  внутри состояния упорядочены по `char`;
- финальность состояний — битмап;
- постинг-листы: `postOff []uint32` (N+1) + массив пар
  `{lemmaId uint32; ancodeId uint16}`, дедуплицированных и упорядоченных.
Выборка словоформы: проход по переходам до узла → срез
`posts[postOff[i]:postOff[i+1]]`. Один внутренний запрос, ноль аллокаций.

### Exact-hash
Open-addressing таблица `hash(text) → диапазон постингов` для точного поиска
без обхода дерева; строится при компиляции одним проходом.

### Ссылки
Секции links/link_types хранятся как есть (id-тройки), холодные данные.

## Сборка (Builder)

`internal/build.Builder` — изменяемая фаза, непотокобезопасна (один писатель),
документировано. Внутренние операции:

- `AddGrammeme(name)` → id;
- `AddLemma(text string, grammemes ...string)` → lemmaId;
- `AddForm(lemmaId, text string, grammemes ...string)` → rowId.

Оба источника данных сводятся к этим операциям:
1. **XML**: `internal/xmlscan` читает поток событий `grammeme | lemma | form`
   напрямую из буфера bz2→bufio; атрибуты — срезы буфера; интернирование по хешу.
2. **Программный API** (FT8): `dictionary.NewEmpty()` → `Builder` → методы выше.

Завершение сборки (`Build()`) выполняет: сортировку уникальных текстов,
построение exact-hash, компиляцию trie→CSR (опционально минимизация DAFSA),
сортировку и дедупликацию постингов, заморозку структур.

## Формат файла

```
magic "GMRF" | version u32 | xxh3 чексумма содержимого
каталог: [имя секции, offset u64, size u64] × N
секции: meta, grammemes, ancodes, textsArena, textOff,
        states, transitions, finals, exactHash, postings,
        lemmaIndex, rowAncodes, links
```

Кодирование: `textOff/stateOff/postOff` — delta+varint; postings —
delta-varint lemmaId + varint ancodeId; transitions — `char u8 + varint delta`.
Холодные секции (links) могут быть zstd-сжаты, горячие хранятся сырыми для mmap.
Чтение: `internal/mmapx` открывает файл, проверяет magic/version/чексумму,
разрезает по каталогу; типизированные представления без копирования где возможно.

## Рантайм (Dictionary)

`pkg/dictionary.Dictionary` — иммутабельный снимок после загрузки/компиляции:

```go
func Open(path string) (*Dictionary, error)              // FT7
func NewEmpty() (*Dictionary, error)                     // FT8
func (d *Dictionary) Lookup(word string) ([]Wordform, error)          // FT2
func (d *Dictionary) Lemmas(word string) ([]LemmaRef, error)          // FT5
func (d *Dictionary) Fuzzy(word string, maxDist int) ([]FuzzyMatch, error) // FT6
func (d *Dictionary) SaveTo(path string) error           // FT3, FT8
func (d *Dictionary) Builder() *build.Builder            // FT8
```

- `Wordform` — значение (не указатель): текст, анкод, разложенные граммемы, lemmaId.
- Чтение конкурентно безопасно (иммутабельные структуры + mmap read-only).
- Экземпляры независимы: свои интернинги, свои mmap-регионы; глобальных
  переменных в пакетах нет. Публичный API не мутирует состояние словаря.
- Обновление словаря (FT7): `opencorpora.Loader.Update()` (загрузка+распаковка)
  → `dictionary.CompileFromXML(path)` → `SaveTo(path)`; атомарная запись через
  временный файл + rename.

## Поиск

- **Точный (FT2)**: exact-hash → postings → значения через арены.
  Резервный путь — обход CSR-trie.
- **Леммы (FT5)**: postings → `lemmaText[l]`; начальная форма — первая строка
  леммы.
- **Нечёткий (FT6)**: совместный обход CSR-trie и DFA Левенштейна с отсечением
  по порогу k; результаты группируются по дистанции (возрастание дистанции =
  убывание релевантности).

## Загрузчик OpenCorpora

`pkg/opencorpora`: проверка обновления по Last-Modified, загрузка `dict.xml.bz2`,
распаковка bzip2, пути в `.data/opencorpora`. Поведение не менялось.

## Метрики (контрольные точки)

| Показатель | Значение |
|---|---|
| Компиляция dict.xml | единицы секунд, крупные аллокации только под итоговые структуры |
| Размер .dat | ~110–120 МБ (varint/delta), ~35–45 МБ с zstd |
| Загрузка .dat | mmap, миллисекунды |
| Точный поиск | < 10 мкс, ноль аллокаций |
| Нечёткий поиск k≤2 | доли секунды на 3М слов |
