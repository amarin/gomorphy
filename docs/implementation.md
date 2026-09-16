# Реализация gomorphy (as-is + план)

Документ описывает текущую реализацию (этапы 0–10) и план новой реализации
(этапы 11–18) с заменой внутреннего формата хранения.

Соответствует требованиям [requirements.md](requirements.md) и плану из [todo.md](todo.md).

## Описание этапов

### Исходная реализация (выполнена)

- [Этап 0. Анализ и подготовка репозитория](implementation/stage-0-analysis.md)
- [Этап 1. Примитивы формата](implementation/stage-1-format.md)
- [Этап 2. Интернирование строк](implementation/stage-2-intern.md)
- [Этап 3. Сканер dict.xml](implementation/stage-3-xmlscan.md)
- [Этап 4. Builder и CSR-структуры](implementation/stage-4-builder-csr.md)
- [Этап 5. Компилятор и загрузчик файла](implementation/stage-5-compiler-loader.md)
- [Этап 6. Публичный фасад pkg/dictionary](implementation/stage-6-facade.md)
- [Этап 7. Интеграция с OpenCorpora end-to-end](implementation/stage-7-opencorpora.md)
- [Этап 8. Поиск лемм FT5](implementation/stage-8-lemmas.md)
- [Этап 9. Нечёткий поиск FT6](implementation/stage-9-fuzzy.md)
- [Этап 10. Финализация](implementation/stage-10-finalize.md)

### Новая реализация (выполнена)

- [x] [Этап 11. Внутренний формат: TagSet + Paradigm + DAWG reader](implementation/stage-11-internal-format.md) — ВЫПОЛНЕН
- [x] [Этап 12. Импорт PyMorphy2](implementation/stage-12-import-pymorphy2.md) — ВЫПОЛНЕН
- [x] [Этап 13. Публичный API: Parse, Lemma, Fuzzy](implementation/stage-13-public-api.md) — ВЫПОЛНЕН
- [x] [Этап 14. Сериализация: единый формат на диске](implementation/stage-14-serialization.md) — ВЫПОЛНЕН
- [x] [Этап 15. Импорт OpenCorpora](implementation/stage-15-import-opencorpora.md) — ВЫПОЛНЕН
- [ ] [Этап 16. Импорт UniMorph](implementation/stage-16-import-unimorph.md)
- [ ] [Этап 17. Сужение типов ID и zstd](implementation/stage-17-optimize.md)
- [ ] [Этап 18. Финализация: CLI, документация, тесты](implementation/stage-18-finalize.md)
- [ ] [Этап 19. Тематические словари: TSV-импорт, CLI-батчи, навыки, решение по MCP](todo.md)
- [ ] [Этап 20. База синонимов: группы, теги, sidecar-файл](todo.md)

Терминология проекта — в [glossary.md](glossary.md).

## Текущая реализация (этапы 0–10)

### Состав репозитория

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

### Модель данных в памяти (текущая)

Все сущности нормализованы в справочники; словоформа — пара идентификаторов.

- `grammemes []string` — имена граммем, id = индекс.
- Анкоды: CSR `ancodeOff []uint32` + `ancodeGrams []uint32` — 876 уникальных
  наборов граммем, id = uint16.
- `textsArena []byte` + `textOff []uint32` — арена текстов.
- CSR-trie: `stateOff`, `TransLabel`, `TransTarget`, `Finals`.
- Exact-hash: open-addressing `hash(text) → trie state`.
- Постинг-листы: `PostingsOff` + `Postings` (пары `lemmaId, ancodeId`).

### Формат файла (текущий)

```
magic "GMRF" | version u32 | xxh3 чексумма
каталог: [имя секции, offset u64, size u64] × N
секции: meta, grammemes, ancodes, textsArena, textOff,
        states, transitions, finals, exactHash, postings,
        lemmaIndex, rowAncodes, links
```

### Рантайм (текущий)

```go
func Open(path string) (*Dictionary, error)
func (d *Dictionary) Lookup(word string) ([]Wordform, error)
func (d *Dictionary) Lemmas(word string) ([]LemmaRef, error)
func (d *Dictionary) Fuzzy(word string, maxDist int) ([]FuzzyMatch, error)
func (d *Dictionary) FuzzyTop(word string, maxWords int) ([]FuzzyMatch, error)
func (d *Dictionary) SaveTo(path string) error
```

## Новая реализация (этапы 11–18)

### Состав репозитория (после этапа 18)

```
pkg/morphology/               публичный фасад: Open, Parse, Lemma, Fuzzy
pkg/morphology/internal/      внутренний формат: TagSet, Paradigm, DAWG, Dictionary
pkg/morphology/importers/     импортёры из разных форматов
pkg/morphology/importers/pymorphy2/   чтение words.dawg + paradigms.array
pkg/morphology/importers/opencorpora/ dict.xml → парадигмы → DAWG
pkg/morphology/importers/unimorph/    TSV → парадигмы → DAWG

internal/xmlscan              (переиспользуется) сканер dict.xml
internal/mmapx                (переиспользуется) mmap-ридер
pkg/opencorpora               (переиспользуется) загрузчик OpenCorpora

cmd/gomorphy                  CLI: lookup/fuzzy/top/lemmas, cli, download/unpack/build/update
                               (folded cmd/gomorphy_build's update/compile into this single
                               binary during the CLI redesign, see docs/superpowers/specs/
                               2026-09-16-cli-redesign-design.md)
```

### Модель данных в памяти (новая)

```
Dictionary
  TagSet         *TagSet        // набор граммем (имя → id)
  Suffixes       []string       // набор суффиксов (id → текст)
  Prefixes       []string       // набор префиксов (id → текст)
  Paradigms      []Paradigm     // шаблоны склонения
  Words          *DAWG          // слова → (para_id, form_idx)
  Prediction     []*DAWG        // предсказание по окончаниям (опционально)
  Probability    *DAWG          // вероятности (опционально)
  CharPolicy     *CharPolicy    // подмены символов (е↔ё и т.д.)
```

DAWG reader (формат dawgdic):
- `Dictionary []uint32` — массив узлов (label + offset + leaf bits)
- `Guide []byte` — навигация: child + sibling по 2 байта на узел
- Поиск: `followByte(r, idx)` → O(1), `find(key)` → O(len)
- Ё-обработка: `similarItems(key)` с подменой `е→ё` на лету

### Формат файла (новый)

```
magic "GMOR" | version u32 | xxh3 чексумма
каталог: [имя секции, offset u64, size u64, flags u8] × N
секции:
  meta            — language, counts, version
  tagset          — JSON-массив имён граммем (общий для всех шардов)
  prefixes        — varint-length-prefixed строки (общий для всех шардов)
  suffixes-N      — varint-length-prefixed строки (по одному набору на шард, N от 0)
  paradigms-N     — uint16 array (N суффиксов + N тегов + N префиксов; по одному набору на шард)
  words.dawg-N    — dictionary uint32[] + guide byte[] (по одному на шард)
  prediction-N    — prediction DAWGs (опционально)
  probability     — probability DAWG (опционально)
```

### Рантайм (новый)

```go
// Открытие из разных источников
func Open(path string) (*Dictionary, error)                      // из .dat файла
func OpenPyMorphy(dir string) (*Dictionary, error)               // из директории pymorphy2
func CompileFromXML(r io.Reader) (*Dictionary, error)            // из dict.xml
func CompileFromUniMorph(r io.Reader) (*Dictionary, error)       // из TSV UniMorph

// Публичный API (FT2, FT5, FT6)
func (d *Dictionary) Parse(word string) []Reading                // точный + предсказание
func (d *Dictionary) Lemma(word string) []LemmaRef               // начальная форма
func (d *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch // нечёткий поиск
func (d *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch

// Сериализация (FT3, FT7)
func (d *Dictionary) SaveTo(path string) error

// Информация
func (d *Dictionary) Language() string
func (d *Dictionary) TagSet() *TagSet

// Закрытие
func (d *Dictionary) Close() error
```

- `Reading` — значение (текст, нормальная форма, теги, вероятность, источник).
- Чтение конкурентно безопасно (иммутабельный снимок).
- Один экземпляр = один язык/источник. Несколько словарей = несколько экземпляров.

### Поиск (новый)

- **Точный (Parse)**: DAWG lookup → `(para_id, form_idx)` → индексная
  арифметика по парадигме → `stem + suffix` + тег. O(len) обход DAWG.
- **Предсказание**: если слово не найдено — поиск по prediction DAWGs
  (1–5-буквенные окончания → наборы разборов).
- **Леммы (Lemma)**: DAWG → `(para_id, 0)` → `stem + suffix[0]`.
- **Нечёткий (Fuzzy)**: совместный обход DAWG и DFA Левенштейна
  с отсечением по порогу k. Метрика по рунам.

## Метрики (контрольные точки)

| Показатель | Текущее (этап 10) | Цель (этап 18) |
|---|---|---|
| Размер .dat (OpenCorpora) | ~305 МБ | ~20–30 МБ |
| Размер .dat (PyMorphy2) | — | ~15–20 МБ |
| Размер .dat с zstd | — | ~10–15 МБ |
| Сборка .dat (OpenCorpora, 3.06М лемм) | ~24 ч (до free-list фикса) | ~24 с |
| Загрузка | mmap, мс | mmap, мс |
| Parse (точный) | < 10 мкс | < 10 мкс |
| Parse (предсказание) | нет | < 50 мкс |
| Lemmas | < 10 мкс | < 10 мкс |
| Fuzzy k≤2 | доли сек | доли сек |
| Поддержка pymorphy2 | нет | да |
| Поддержка UniMorph | нет | да (169 языков, opaque) |
| Множественные словари | на уровне приложения | на уровне приложения |
| Языковая нейтральность | нет (русский) | да |
