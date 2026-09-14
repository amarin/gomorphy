# План работ

Все этапы 0–10 (исходная реализация) выполнены. Описание каждого этапа —
в [implementation.md](implementation.md) и отдельных файлах в
[implementation/](implementation/).

Новая реализация (этапы 11–18) заменяет внутренний формат хранения
(CSR-trie + exact-hash + пары → парадигмы + DAWG) с сохранением
переиспользуемого кода (xmlscan, intern, stringsx, mmapx, opencorpora).

## Требования к оформлению

Каждый этап — самостоятельный инкремент с проверяемым результатом.
Любой агент может продолжить с места остановки: состояние репозитория после
каждого этапа собирается (`go build ./...`), тесты этапа зелёные.
Отмечать выполненное: `[x]`.

## Выполненные этапы (исходная реализация)

### Этап 0. Анализ и подготовка репозитория — ВЫПОЛНЕН
[Подробное описание](implementation/stage-0-analysis.md)

### Этап 1. Примитивы формата (internal/format) — ВЫПОЛНЕН
[Подробное описание](implementation/stage-1-format.md)

### Этап 2. Интернирование строк (internal/intern, internal/stringsx) — ВЫПОЛНЕН
[Подробное описание](implementation/stage-2-intern.md)

### Этап 3. Сканер dict.xml (internal/xmlscan) — ВЫПОЛНЕН
[Подробное описание](implementation/stage-3-xmlscan.md)

### Этап 4. Builder и CSR-структуры (internal/build) — ВЫПОЛНЕН
[Подробное описание](implementation/stage-4-builder-csr.md)

### Этап 5. Компилятор и загрузчик файла — ВЫПОЛНЕН
[Подробное описание](implementation/stage-5-compiler-loader.md)

### Этап 6. Публичный фасад pkg/dictionary (FT7–FT9) — ВЫПОЛНЕН
[Подробное описание](implementation/stage-6-facade.md)

### Этап 7. Интеграция с OpenCorpora end-to-end — ВЫПОЛНЕН
[Подробное описание](implementation/stage-7-opencorpora.md)

### Этап 8. Поиск лемм FT5 — ВЫПОЛНЕН
[Подробное описание](implementation/stage-8-lemmas.md)

### Этап 9. Нечёткий поиск FT6 — ВЫПОЛНЕН
[Подробное описание](implementation/stage-9-fuzzy.md)

### Этап 10. Финализация — ВЫПОЛНЕН
[Подробное описание](implementation/stage-10-finalize.md)

---

## Сравнительный анализ: gomorphy vs PyMorphy2 (пересмотренный)

Исходный материал: [pymorphy2 — внутреннее устройство](https://pymorphy2.readthedocs.io/en/stable/internals/index.html),
[opennota/morph](https://gitlab.com/opennota/morph) (Go-реализация чтения pymorphy2).

### Архитектура хранения PyMorphy2

Ключевая идея: **парадигмы + DAWG**.

1. **Парадигмы.** Каждая лемма разбирается на префикс + стем + суффикс.
   Стем отбрасывается; (префикс, суффикс, тег) кодируются числовыми индексами.
   Результат — шаблон склонения (парадигма). Для русского: ~3 000 парадигм
   из ~400K лексем. Парадигма хранится как `array.array("<H")`: N суффиксов
   + N тегов + N префиксов.

2. **DAWG слов.** Все слова — в минимизированном конечном автомате.
   Ключ: `<слово>\x00<para_id><form_idx>`. DAWG сливает общие префиксы и
   суффиксы. 5 млн словоформ ≈ 7 МБ.

3. **Теги и суффиксы.** Пулы строк: `suffixes.json` (~5K суффиксов),
   `paradigm-prefixes.json` (~3 префикса), `gramtab-opencorpora-int.json`
   (~1K тегов). Хранятся как JSON-массивы строк.

4. **Предсказание.** Отдельные DAWG для 1–5-буквенных окончаний → наборы
   разборов. Обеспечивает разбор несловарных слов.

5. **Чтение в Go.** Библиотека `opennota/morph` (~500 строк) читает формат
   pymorphy2 напрямую: dictionary+guide массивы для DAWG, binary Read для
   paradigms.array, JSON для суффиксов/тегов. Ё-обработка на лету.

| Сущность | Кол-во | Объём |
|---|---|---|
| Парадигмы | ~3 000 | ~3–4 МБ |
| Суффиксы/префиксы/теги | ~6K | ~0.5 МБ |
| Слова в DAWG | ~5 млн | ~7 МБ |
| Предсказание (3 DAWG) | — | ~3–4 МБ |
| **Итого** | | **~15 МБ** |

### Архитектура хранения gomorphy (текущая)

Ключевая идея: **интернирование + CSR-trie + exact-hash**.

| Сущность | Кол-во | Объём на диске |
|---|---|---|
| Уникальные тексты (TextData) | 3 065 312 | ~67 МБ |
| Exact-hash таблица | ~4,4 млн слотов × 16 байт | ~47 МБ |
| Pairs (text_id + ancode_id) | 5 393 737 | ~62 МБ (raw u32) |
| Trie (CSR) | сотни тыс. состояний | ~30–40 МБ |
| Постинг-листы | 5,4 млн записей | ~20 МБ |
| Леммы + анкоды | 391К + 876 | ~5 МБ |
| **Итого на диске** | | **~305 МБ** |

### Причины расхождения в 20×

| Причина | Доля экономии pymorphy2 | Комментарий |
|---|---|---|
| Парадигмы вместо плоских текстов | ~40 МБ (67→27 МБ) | 3K шаблонов вместо 3M текстов |
| DAWG вместо CSR-trie | ~15–20 МБ | Слияние эквивалентных состояний |
| Встроенные метаданные в DAWG | ~42 МБ | PairTexts+PairAncodes не нужны |
| Отсутствие exact-hash | ~47 МБ | DAWG обеспечивает O(len) lookup |
| Отсутствие постинг-листов | ~20 МБ | Метаданные в DAWG-значениях |

### Почему рефакторинг текущего кода не работает

Текущая модель gomorphy **принципиально отличается** от pymorphy2:

1. **Пары (textID, ancodeID)** — центральная единица хранения. Один текст
   может иметь несколько анкодов (омонимия). В pymorphy2 это не нужно:
   DAWG хранит `(слово → para_id, form_idx)`, и тег берётся из парадигмы.

2. **Постинг-листы** привязаны к узлам trie. В pymorphy2 их нет:
   DAWG сам содержит `(para_id, form_idx)` как значение.

3. **Exact-hash** — отдельная 47 МБ таблица. В pymorphy2 DAWG обеспечивает
   быстрый поиск без дополнительной структуры.

Пошаговое «внедрение парадигм» в текущую модель не даёт основного выигрыша
(встраивание метаданных в граф), а создаёт гибрид без преимуществ ни одной
модели.

### Почему rewrite оправдан

1. **opennota/morph доказывает** DAWG-модель в Go: ~500 строк, чтение формата
   pymorphy2, Ё-обработка, prediction. Формат прост: dictionary uint32[] +
   guide byte[].

2. **XML-пайплайн переиспользуется.** `internal/xmlscan` читает dict.xml —
   эту часть не нужно переписывать. Новый импортёр берёт события xmlscan
   и строит парадигмы + DAWG.

3. **CLI адаптируется.** Интерактивный режим и команды — 90% кода остаётся.
   Добавляются `import` и multi-source.

4. **FT8 (Builder) сохраняется.** API `AddGrammeme/AddLemma/AddForm` остаётся —
   это вход для программного наполнения.

---

## План работ (новая реализация, этапы 11–18)

Каждый этап — самостоятельный инкремент с проверяемым результатом.
Переиспользуемый код: `internal/xmlscan`, `internal/intern`,
`internal/stringsx`, `internal/mmapx`, `pkg/opencorpora`.

### Этап 11. Внутренний формат: TagSet + Paradigm + DAWG reader — ВЫПОЛНЕН

**Сводка**: Реализовать основные примитивы внутреннего формата:
TagSet (набор тегов), Paradigm (шаблон склонения), DAWG reader
(чтение формата dawgdic из pymorphy2). Этот этап создаёт фундамент
для всех последующих.

**Инкремент**:
- Новый пакет `pkg/morphology/internal/`:
  - `tagset.go`: `TagSet{Name string, Tags []string, Index map[string]uint16}`
  - `paradigm.go`: `Paradigm` — плоский `[]uint16` (N suffixes + N tags + N prefixes)
  - `dawg.go`: DAWG reader (dictionary + guide массивы, формат dawgdic).
    Reading: `followByte`, `followRune`, `find`, `similarItems` (с Ё-заменой).
    Формат: uint32 array (dictionary) + byte array (guide, по 2 байта на узел).
  - `char_policy.go`: `CharPolicy` — набор пар заменяемых символов
    (по умолчанию `е→ё` для русского, пусто для других языков).
  - `dictionary.go`: `Dictionary` — иммутабельный снимок:
    `TagSet`, `Suffixes []string`, `Prefixes []string`, `Paradigms []Paradigm`,
    `Words *DAWG`, `CharPolicy`.

**Автоматические проверки (тесты)**:
- unit-тест: DAWG reader читает тестовый DAWG (создан в памяти).
- unit-тест: `followByte`/`followRune` возвращают корректные переходы.
- unit-тест: `similarItems` с CharPolicy `е→ё` находит варианты.
- unit-тест: Paradigm — индексная арифметика (suffix, tag, prefix по индексу).
- unit-тест: TagSet — добавление и поиск тегов.
- `go test ./pkg/morphology/... -race` — зелёные.

**Ручные проверки**:
- Прочитать `words.dawg` из pymorphy2-dicts-ru: `morph.InitWith(path)`.
- `similarItems("кота")` → найдены разборы.

### Этап 12. Импорт PyMorphy2 — ВЫПОЛНЕН

> Примечание: roundtrip-тест из списка проверок (`ImportFromDir → SaveTo →
> Open`) отложен до этапа 14, где реализуется сериализация `SaveTo`/`Open`.

**Сводка**: Загрузка словаря pymorphy2 из директории (words.dawg +
paradigms.array + suffixes.json + gramtab-*.json) → иммутабельный
`Dictionary`. Прямое чтение формата без конвертации.

**Инкремент**:
- `pkg/morphology/importers/pymorphy2/`:
  - `import.go`: `ImportFromDir(dir string) (*Dictionary, error)`
  - Чтение `paradigms.array`: uint16 count + для каждой: uint16 len + []uint16 data
  - Чтение `suffixes.json`, `paradigm-prefixes.json`: JSON-массив строк
  - Чтение `gramtab-opencorpora-int.json`: JSON-массив строк → TagSet
  - Чтение `words.dawg`: dictionary + guide → DAWG
  - Опционально: `prediction-suffixes-N.dawg` → Prediction []*DAWG
  - Опционально: `p_t_given_w.intdawg` → probability DAWG
- Публичный API в `pkg/morphology/`:
  - `OpenPyMorphy(dir string) (*Dictionary, error)` — обёртка над импортёром

**Автоматические проверки (тесты)**:
- unit-тест: маленький DAWG + парадигмы → Dictionary.
- unit-тест: roundtrip: ImportFromDir → SaveTo → Open → данные идентичны.
- integration-тест: полный словарь pymorphy2-dicts-ru → Parse("все") → ≥4 разбора.
- `go test ./pkg/morphology/... -race` — зелёные.

**Ручные проверки**:
- `gomorphy -dict pymorphy2.dat lookup кота` → результаты идентичны pymorphy2.

### Этап 13. Публичный API: Parse, Lemma, Fuzzy — ВЫПОЛНЕН

**Сводка**: Реализовать публичный API поверх Dictionary:
Parse (точный поиск + предсказание), Lemma (начальная форма),
Fuzzy (нечёткий поиск). Адаптация CLI.

**Инкремент**:
- `pkg/morphology/parse.go`:
  - `Parse(word string) []Reading` по модели pymorphy2 → opennota: чтение
    payload words.dawg = `(para, form)` (2×uint16 BE), норма = premises
    `TrimPrefix/TrimSuffix` → `prefix[0]+stem+suffix[0]` для form≠0;
    сортировка по prob (ключ `word+":"+tag`, `prob/1e6`), только если есть
    ненулевая. Prediction по окончаниям: 6-байтовые значения
    `(count, para, form)`, productive-граммемы, `suffixSplits` до 5 рун,
    break при totalCount>1, дедуп по (Word,Normal,Tag).
  - Обёртка `pkg/morphology.Dictionary{d *internal.Dictionary}`;
    `OpenPyMorphy` возвращает `*Dictionary` (было `*internal.Dictionary`),
    метод `Language()`.
  - CLI-адаптация **перенесена на этап 14** (нужны SaveTo/Open — `-dict`
    открывает скомпилированный `.dat`).
- `pkg/morphology/lemma.go`:
  - `Lemma(word string) []LemmaRef` — начальная форма через Parse;
    дедуп по (Normal, Tag) — омонимы сохраняются.
- `pkg/morphology/fuzzy.go`:
  - `Fuzzy(word, maxDist) []FuzzyMatch` — совместный обход DAWG и banded DP
    Левенштейна (метрика по рунам); терминал узла — по guide
    (`HasPayloadChild`), т.к. FollowByte-проба ловит коллизии double-array.
  - `FuzzyTop(word, maxWords) []FuzzyMatch` — расширение радиуса до верхней
    границы `len(query)+наиб. длина слова` (дедуп по узлам)
  - `internal`: `PayloadSeparator` (exported), `DAWG.ForEachChild`,
    `DAWG.HasPayloadChild`.

**Автоматические проверки (тесты)**:
- unit-тест: `Parse("кота")` → 1 разбор `NOUN,anim,masc,sing,gent`, Normal "кот".
- unit-тест: `Parse("кот")` → 2 разбора, сортировка по prob (NOUN 0.0005 > VERB 0.0001).
- unit-тест: `Parse("КОТ")` — lowercase, `Parse("котёнка")` — предсказание.
- unit-тест: `Lemma("кота")`→{кот,nomn,para0}; «кот» → 2 омонима.
- unit-тест: `Fuzzy("кот",1)`→{кот:0,код:1,кота:1,крот:1}; е/ё метрика;
  `FuzzyTop("кот",3)`→[кот,код,кота]; FuzzyTop охват всего словаря.
- `go test ./pkg/morphology/... -race` и `go vet` — зелёные.

**Ручные проверки**:
- `gomorphy -dict pymorphy2.dat lookup кота` → корректный разбор.
- `gomorphy -dict pymorphy2.dat fuzzy кот 1` → список слов.
- Интерактивный режим: все команды работают.

### Этап 14. Сериализация: единый формат на диске — ВЫПОЛНЕН

**Сводка**: Реализованы единый дисковый формат GMOR (заголовок + каталог +
секции, checksum xxh3-64, смещения секций не выровнены по 8 байтам), кодек
секций (meta, tagset, suffixes, prefixes, paradigms, words.dawg,
prediction-N, probability) с zero-copy mmap-алиасингом words.dawg,
`Dictionary.SaveTo`/`Open`/`Close` и миграция CLI на новую публичную модель
(`import pymorphy2`, `lookup`, `lemmas`, `fuzzy`, `top`). zstd-сжатие секций
отложено на этап 17 (флаг сжатия в каталоге уже предусмотрен, чтение
сжатой секции сейчас завершается ошибкой).

**Сводка**: Определить и реализовать единый формат файла для хранения
словаря: секции (header, meta, suffixes, prefixes, tagset, paradigms,
words.dawg, prediction). Формат совместим с pymorphy2 или является
его расширением.

**Инкремент**:
- `pkg/morphology/internal/format.go`:
  - Формат: magic "GMOR" | version uint32 | checksum xxh3
  - Каталог секций: [name, offset, size] × N
  - Секции: meta, tagset, suffixes, prefixes, paradigms, words.dawg,
    prediction-0..N, probability
  - Чтение: mmap + срезание по каталогу (как в текущем format/)
  - Запись: посекционная запись с возможным zstd-сжатием холодных секций
- `pkg/morphology/save.go`:
  - `Dictionary.SaveTo(path string) error`
  - Сериализация: suffixes/prefixes как varint-length-prefixed строки,
    paradigms как uint16 array, words.dawg как raw bytes.
- `pkg/morphology/open.go`:
  - `Open(path string) (*Dictionary, error)` — загрузка из файла
  - mmap для горячих секций (words.dawg), полная загрузка для холодных

**Автоматические проверки (тесты)**:
- unit-тест: roundtrip ImportFromDir → SaveTo → Open → Parse результаты идентичны.
- unit-тест: формат-версия корректно записывается и читается.
- unit-тест: повреждённый файл → осмысленная ошибка.
- `go test ./pkg/morphology/... -race` — зелёные.

**Ручные проверки**:
- Сравнить размер `.dat` с размером директории pymorphy2.
- `gomorphy -dict pymorphy2.dat lookup кота` → идентично прямой загрузке.

### Этап 15. Импорт OpenCorpora

**Сводка**: Импорт словаря OpenCorpora (`dict.xml`) через извлечение
парадигм из лемм. Переиспользование `internal/xmlscan` для чтения XML.
Результат — тот же внутренний формат (paradigm + DAWG).

**Инкремент**:
- `pkg/morphology/importers/opencorpora/`:
  - `import.go`: `ImportFromXML(r io.Reader, tagSet *TagSet) (*Dictionary, error)`
  - Pipeline:
    1. Сканировать XML через `xmlscan` (события: grammeme, lemma, form)
    2. Для каждой леммы: собрать все формы → вычислить стем (LCP) →
       суффиксы = хвосты форм минус стем → парадигма = (suffix_id, tag_id, 0)
    3. Дедуплицировать парадигмы ( map[paradigmKey]paradigmID )
    4. Построить DAWG из всех слов с `(para_id, form_idx)` как values
    5. Вернуть Dictionary
  - TagSet по умолчанию: `gramtab-opencorpora-int.json` из pymorphy2
    (совместимые теги) или пользовательский.
- `pkg/morphology/compile.go`:
  - `CompileFromXML(xmlPath, outPath string) error` — полный цикл
  - `CompileFromXMLFile(path string) (*Dictionary, error)` — в память

**Автоматические проверки (тесты)**:
- unit-тест: маленький XML (5–10 лемм) → парадигмы извлечены корректно.
- unit-тест: roundtrip: CompileFromXML → SaveTo → Open → Parse идентичен.
- unit-тест: число парадигм < число лемм (для dict.xml).
- property-тест: N случайных слов → все найдены.
- integration-тест: 100 случайных словоформ → сверка с независимым разбором.
- `go test ./pkg/morphology/... -race` — зелёные.

**Ручные проверки**:
- `gomorphy import opencorpora dict.xml -o oc.dat` → файл создан.
- `gomorphy -dict oc.dat lookup кота` → корректный разбор.
- Сравнить результаты с pymorphy2 (для слов, которые есть в обоих).

### Этап 16. Импорт UniMorph

**Сводка**: Импорт словаря UniMorph из TSV (`лемма<TAB>форма<TAB>bundle`).
Аналогичен этапу 15, но проще: без XML, потоковое чтение. Признаки UniMorph
сохраняются как есть (opaque TagSet); маппинг на OpenCorpora-теги — опция.
Подробный разбор — в [unimorph.md](unimorph.md).

**Инкремент**:
- `pkg/morphology/importers/unimorph/`:
  - `import.go`: `ImportFromTSV(r io.Reader, opts Options) (*Dictionary, error)`
  - Pipeline:
    1. Потоковое чтение (bufio.Scanner), разбиение по `\t` на 3 поля:
       лемма, словоформа, bundle
    2. Лемма → `AddLemma`; форма → `AddForm(lemmaID, словоформа, SPLIT(bundle))`
    3. Опция `Mapping`: проекция признаков UniMorph на другой TagSet
       (например, OpenCorpora); неполные соответствия — как есть
    4. `Compile()` → Dictionary (единый `.dat`-формат, общий для источников)
- `pkg/morphology/compile.go`:
  - `CompileFromUniMorph(r io.Reader) (*Dictionary, error)`
  - `CompileFromUniMorphFile(path string) (*Dictionary, error)`
- CLI: `gomorphy import unimorph <rus.tsv> -o ru-unimorph.dat`

**Автоматические проверки (тесты)**:
- unit-тест: мини-TSV (5–10 лемм) → корректные леммы и парадигмы.
- unit-тест: roundtrip ImportFromTSV → SaveTo → Open → Parse идентичен.
- unit-тест: синкретизм: одна форма с несколькими bundles → все чтения.
- unit-тест: пустая лемма → лемма == словоформа.
- unit-тест: маппинг тегов (opaque и `Mapping`).
- integration-тест: весь `rus` → `Lookup`/`Lemmas` на выборочных словах.
- integration-тест: парадигмы < лемм (для `rus`: 28 068 < 28 069).
- `go test ./pkg/morphology/... -race` — зелёные.

**Ручные проверки**:
- `gomorphy import unimorph rus -o ru-unimorph.dat` → файл создан.
- `gomorphy -dict ru-unimorph.dat lookup кота` → `N;ACC;SG`.
- Другой язык (`eng`): `gomorphy -dict en.dat lookup cats` — без изменений (FT10).
- Сверка по словам, общим с OpenCorpora-словарём.

### Ускорение сборки DAWG: free-list вместо O(n²) сканирования — ВЫПОЛНЕНО

**Проблема**: после этапа 15 сборка реального словаря OpenCorpora
(dict.xml, 3 065 312 лемм) занимала около 24 часов. Причина —
`compileImpl` (`pkg/morphology/internal/dawgbuild.go`) искал свободный
`base`-слот double-array раскладки линейным сканированием бита за битом от
`base=1` для каждого узла; по мере заполнения массива стоимость поиска на
узел росла вместе с числом уже размещённых узлов, что даёт квадратичную
асимптотику.

**Решение**: placement-алгоритм заменён на intrusive doubly-linked free
list (техника dawgdic/cedar/Darts, Aoe 1989): свободные слоты связаны в
список, поиск посещает только свободные слоты, плюс кэш подсказок по
первому байту метки. Формат файла и публичный API не изменились.
Подробности — в [design-спеке](superpowers/specs/2026-09-14-dawg-build-freelist-design.md)
и [плане реализации](superpowers/plans/2026-09-14-dawg-build-freelist.md).

**Результат** (полный `dict.xml`, `gomorphy_build compile`):

| Метрика | До фикса | После фикса |
|---|---|---|
| Время сборки (OpenCorpora, 3.06М лемм) | ~24 ч | ~24 с |
| Peak RSS | — | ~8.2 ГБ |

Синтетический бенчмарк (`dawgbuild_scaling_test.go`, `-tags scaling`)
подтверждает сублинейно-квадратичный (не O(n²)) рост при 100К–5М ключей.

### Этап 17. Сужение типов ID и zstd

**Сводка**: Оптимизация размера: uint16 для paradigm/suffix/tag IDs,
zstd-сжатие холодных секций (suffixes, prefixes, tagset).

**Инкремент**:
- Paradigm: `[]uint16` вместо `[]uint32` (значения ≤ 10K)
- Tag IDs: `uint16` (значения ≤ 1K)
- Suffix/Prefix IDs: `uint16` (значения ≤ 10K)
- zstd-сжатие для секций: suffixes, prefixes, tagset, paradigms
  (холодные данные, декомпрессия один раз при загрузке)
- Флаг сжатия в каталоге секций

**Автоматические проверки (тесты)**:
- unit-тест: roundtrip сжатый → несжатый → данные идентичны.
- unit-тест: сжатая секция меньше несжатой.
- benchmark: Parse до и после (ожидается 0% regression).
- `go test ./pkg/morphology/... -race` — зелёные.

**Ручные проверки**:
- `make compile`: сравнить размер `.dat` до и после.
- `gomorphy -dict pymorphy2.dat lookup кота` → идентично.

### Этап 18. Финализация: CLI, документация, тесты

**Сводка**: Обновление CLI (все команды через аргументы + интерактивный
режим), финализация документации, полный прогон тестов.

**Инкремент**:
- `cmd/gomorphy/main.go`:
  - Все команды доступны через CLI-аргументы: `lookup`, `lemmas`, `fuzzy`,
    `top`, `import`
  - Интерактивный режим: `parse`, `lemma`, `fuzzy`, `top`, `import`, `help`
  - Мульти-словарь на уровне CLI: `-dict` принимает путь к .dat файлу
- `cmd/opencorpora_update/main.go`:
  - Объединение с `import`: `gomorphy update` = загрузка + import
- Документация: обновление `docs/`, `README.md`, godoc
- Навык `skills/use-dictionary/SKILL.md` — «использование словаря gomorphy»:
  lookup/lemmas/fuzzy/top/import, работа с несколькими `.dat` (создаётся
  вместе с причёсыванием CLI).
- Полный прогон: `go vet ./...`, `go test -race ./...`, `go build ./...`

**Автоматические проверки (тесты)**:
- Все unit-тесты зелёные.
- Все integration-тесты зелёные.
- `go vet ./...` без замечаний.
- `go build ./...` без ошибок.

**Ручные проверки**:
- Полный цикл: `import pymorphy2` → `lookup` → `fuzzy` → `top`.
- Полный цикл: `import opencorpora` → `lookup` → `fuzzy` → `top`.
- Полный цикл: `import unimorph` → `lookup` → `fuzzy` → `top`.
- Интерактивный режим: все команды работают с tab-completion.

---

## Итоговые метрики (ожидаемые)

| Показатель | Текущее | После этапа 18 |
|---|---|---|
| Размер .dat (pymorphy2) | — | ~15–20 МБ |
| Размер .dat (opencorpora) | 305 МБ | ~20–30 МБ |
| Размер .dat с zstd | — | ~10–15 МБ |
| Загрузка | mmap, мс | mmap, мс |
| Parse (точный) | < 10 мкс | < 10 мкс |
| Parse (предсказание) | нет | < 50 мкс |
| Lemmas | < 10 мкс | < 10 мкс |
| Fuzzy k≤2 | доли сек | доли сек |
| Поддержка pymorphy2 | нет | да |
| Поддержка OpenCorpora | да | да (новый формат) |
| Поддержка UniMorph (TSV) | нет | да (169 языков) |
| Множественные словари | на уровне приложения | на уровне приложения |

---

## Этап 19. Тематические словари: TSV-импорт, CLI-батчи, навыки, решение по MCP — ЗАПЛАНИРОВАН

**Сводка**: Сделать библиотеку и CLI удобным инструментом для создания
**тематических словарей** программным агентом без внешних ресурсов (нет
интернета, нет базового словаря OpenCorpora): подготовка набора словоформ в
простом текстовом формате → импорт → `.dat`. Сопровождается навыком
«тематический словарь» для агента (навык «использование словаря gomorphy» —
в этапе 18) и зафиксированным решением не реализовывать MCP
([docs/mcp.md](mcp.md)).

Формат строки — **TSV, третий столбец опционален**:

```
лемма<TAB>форма[<TAB>теги через запятую]
```

Ключевые свойства:
- **Опциональные теги.** Третий столбец можно опускать: для тематического
  словаря связь «форма → лемма» важнее граммематики. Без тегов агент не
  тратит токены на подбор OpenCorpora-тегов.
- **Opaque-теги.** Теги — произвольные строки, включая кастомные метки
  пользователя (`разг`, `устар`, `флотск` и т.п.); регистрируются
  автоматически как граммемы. Никакого маппинга на OpenCorpora-набор.
- **Авто-лемма.** Пустая лемма → лемма := словоформа (неизменяемые формы).
- **Дедупликация.** Повторы пары (форма, теги) в пределах леммы отбрасываются
  Builder'ом.

**Инкремент**:
- `pkg/dictionary` (текущий фасад, FT8 Builder сохраняется и в новой
  реализации `pkg/morphology`):
  - `import_tsv.go`: `ImportTSV(r io.Reader, opts Options) (*Dictionary, error)`,
    `ImportTSVFile(path string, opts Options) (*Dictionary, error)` — потоковое
    чтение `bufio.Scanner`, разбиение по `\t` на 2–3 поля → `AddLemma`/`AddForm`,
    авторегистрация граммем, дедуп.
  - `report.go`: отчёт импорта — число лемм/форм, леммы без тегов,
    предупреждения о расходящихся стемах (LCP) и аномально коротких
    парадигмах (замена внешней верификации в offline-сценарии).
- CLI `cmd/gomorphy`:
  - `gomorphy import tsv <file> -o out.dict [--summary]`;
  - **батч-режимы запросов**: `lookup`/`lemmas`/`fuzzy` принимают несколько
    слов в аргументах или читают из stdin по одному слову на строку (аргумент
    `-`); результаты — секции по слову. Батчинг закрывает главный источник
    раздувания токенов у агента (см. [docs/mcp.md](mcp.md), §Токены).
- Навык `skills/thematic-dictionary/SKILL.md` — «тематический словарь»
  (оформляется как `SKILL.md` в репозитории, версионируется вместе с
  библиотекой, копируется в конфигурацию агента): как подготовить TSV для
  произвольной лексики (существительные, названия кораблей/областей,
  изменяющиеся как прилагательные), таблицы парадигм, кастомные теги,
  автономность без OpenCorpora, верификация через батч-`lookup` и отчёт
  импорта. (Скилл «использование словаря gomorphy» — в этапе 18.)
- Решение по MCP: [docs/mcp.md](mcp.md) — зафиксировать намерение не
  реализовывать встроенный MCP-сервер с аргументацией (нет экономии токенов,
  накладные расходы, закрывается CLI-батчами).

**Автоматические проверки (тесты)**:
- unit: мини-TSV (5–10 лемм) → корректные леммы, формы, теги;
- unit: опциональные теги, пустая лемма, кастомные теги, дедуп;
- unit: отчёт импорта — счётчики и предупреждения;
- roundtrip: `ImportTSV → SaveTo → Open → Lookup` идентичен прямому построению;
- CLI: батч `lookup` по нескольким аргументам и по stdin;
- `go test -race ./...` — зелёные.

**Ручные проверки**:
- `gomorphy import tsv names.tsv -o names.dict --summary` → `.dat` создан;
- `gomorphy -dict names.dict lookup - < words.txt` → секции по слову;
- прогон скилла «тематический словарь» через агента на примере «словарь
  прилагательных-названий кораблей» без загруженного OpenCorpora.

---

## Этап 20. База синонимов: группы, теги, sidecar-файл — ЗАПЛАНИРОВАН (сценарии — открытый вопрос)

**Сводка**: Рядом со словарём словоформ — отдельная база **синонимов и
производных понятий** (sidecar-файл `.syn`). Задачи, которые она решает и
которые не выводятся из словоформ:

- «основное / официальное понятие»: топорище → топор, холодильник → холод;
- «неканоническая форма → официальная»: Лёша → Алексей, Дима → Дмитрий,
  Шурик → Александр (формы могут быть вообще не связаны по Левенштейну);
- **обратный запрос** «все производные от базового понятия»: холод →
  {холодильник, холодец, охлаждение}; Александр → {Саша, Шура, Шурик, Саня};
- **m2m**: у леммы может быть несколько интерпретаций (Лёня → и Леонид,
  и Алексей) — модель «групп» (clique), а не пар.

Связи достаточно задавать на уровне **лемм** (грамматические производные
вроде «Шуриком от Александра» докручиваются в коде через `Lookup`).

**Гипотеза о применении (проверить сценариями)**: база синонимов рядом со
словарём словоформ и словообразованием — или сама по себе — перспективна для
задач обработки и генерации текстов: синонимическая замена, генерация
вариантов, нормализация неканонических форм. Сценарии вырабатываются до
реализации (см. «Открытые вопросы»).

### Теги разметки (требуется дальнейшее проектирование)

Внутри секций синонимов желательна разметка тегами, и набор тегов
существенно зависит от домена слова:

- для имён: «официальное», «просторечное», «ласкательное», «грубое»;
- для инструментов: официальные названия, просторечные, местные/диалектные
  и т.д.

Поэтому этап **не фиксирует набор тегов**: теги задаются пользователем как
opaque-граммемы (аналогично тематическим словарям, этап 19) и регистрируются
при импорте.

**Открытые вопросы (проработать до реализации, помечено в плане)**:
- семантика тегов: набор задаёт библиотека или пользователь; обязательны ли
  они; размечают группу целиком или отдельных участников;
- домен-специфичность наборов тегов (имена vs инструменты) — как её
  моделировать без жёстких предопределённых списков;
- ключ синонимов: текст леммы (просто, омонимия смешивается) vs `LemmaID`
  (точно, но привязка к версии `.dat`);
- хранилище: sidecar `.syn` (независимое обновление, не требует пересборки
  большого словаря) vs секция в `.dat` (пересборка при каждой правке
  синонимов) — предпочтителен sidecar;
- совместный и standalone-режимы: вход через существующие `Lookup`/`Lemmas`
  при наличии словаря, либо прямые запросы по ключу без словаря.

**Инкремент** (уточняется после проработки сценариев):
- пакет `pkg/synonyms`:
  - модель «групп»: лемма → []id групп; группа → []участников + опциональные
    теги (opaque-граммемы);
  - импорт TSV `группа<TAB>лемма[<TAB>теги]` (переиспользование схемы импорта
    этапа 19), дедуп, авторегистрация тегов;
  - сериализация в компактный sidecar-файл (varint-массивы, xxh3, mmap);
  - API: `Synonyms(lemma)`, `Derivations(lemma)`; при наличии словаря — вход
    через `Lookup`;
  - CLI: `gomorphy import synonyms <file> -o dict.syn`,
    `gomorphy -dict dict.dat synonyms <word>`, `derivations <word>`;
    standalone-режим без `-dict`.

**Автоматические проверки (тесты)**:
- unit: m2m (Лёня → {Леонид, Алексей});
- unit: обратный запрос «производные от базового понятия» (холод → …);
- unit: импорт с тегами (именительный набор / доменный набор), дедуп;
- roundtrip: `import → SaveTo → Open → Synonyms/Derivations` идентичен
  прямому построению;
- интеграция: словоформа → `Lookup` → лемма → `Synonyms`;
- `go test -race ./...` — зелёные.

**Ручные проверки**:
- пример на именах: «Лёша → Алексей», «Шурик → Александр», обратный
  «Александр → {Саша, Шура, Шурик, Саня}»;
- пример на инструментах с доменным набором тегов;
- standalone-режим без загруженного словаря словоформ.
