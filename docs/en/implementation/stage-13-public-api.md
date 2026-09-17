# Этап 13. Публичный API: Parse, Lemma, Fuzzy

## Содержание этапа

Реализация публичного API поверх `Dictionary` из этапа 11: Parse
(точный поиск + предсказание), Lemma (начальная форма), Fuzzy
(нечёткий поиск). Адаптация CLI.

### Parse (`pkg/morphology/parse.go`)

```go
func (d *Dictionary) Parse(word string) []Reading
```

Логика:
1. `d.Words.find(word)` → если найден, получить `(para_id, form_idx)`.
   DAWG-значение кодируется как `(para_id << 16) | form_idx`.
2. Для каждого `(para_id, form_idx)`:
   - `paradigm = d.Paradigms[para_id]`
   - `suffix_id = paradigm.Suffix(form_idx)`
   - `tag_id = paradigm.Tag(form_idx)`
   - `prefix_id = paradigm.Prefix(form_idx)`
   - `normal = d.Prefixes[prefix_id] + stem + d.Suffixes[0]` (начальная форма)
   - `tags = d.TagSet.Names(tag_id)`
   - Вероятность: если `d.Probability != nil` → lookup по `(para_id, form_idx)`
3. Если слово не найдено и prediction DAWGs доступны:
   - Для каждого `d.Prediction[i]`:
     - `suffix = word[len(word)-i:]` (последние i символов)
     - `d.Prediction[i].find(suffix)` → набор `(para_id, form_idx)`
     - Те же вычисления что и выше

### Lemma (`pkg/morphology/lemma.go`)

```go
func (d *Dictionary) Lemma(word string) []LemmaRef
```

Логика:
1. DAWG lookup → `(para_id, form_idx)` для каждой вхождения.
2. Начальная форма: `d.Prefixes[paradigm.Prefix(para_id)] + stem + d.Suffixes[0]`.
3. Теги начальной формы: `d.TagSet.Names(paradigm.Tag(para_id, 0))`.
4. Дедупликация по тексту начальной формы.

### Fuzzy (`pkg/morphology/fuzzy.go`)

```go
func (d *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch
func (d *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch
```

Логика (совместный обход DAWG и DFA Левенштейна):
1. DFA Левенштейна: `[]int` — текущее расстояние по столбцам (по байтам).
2. Рекурсивный обход: для каждого перехода DAWG `(byte → target_state)`:
   - Вычислить расстояние с учётом замены/вставки/удаления.
   - Если текущее расстояние ≤ maxDist — продолжить рекурсию.
   - Если узел DAWG финальный и расстояние ≤ maxDist — добавить результат.
3. Метрика по рунам (для корректности «ё/е» = 1 подстановка).
4. `FuzzyTop` — итеративное расширение maxDist от 1 доatisfied.

### CLI (`cmd/gomorphy/main.go`)

Новые команды:
```
gomorphy -dict <path> lookup <word>      точный разбор
gomorphy -dict <path> lemma <word>       начальная форма
gomorphy -dict <path> fuzzy <word> <k>   нечёткий поиск
gomorphy -dict <path> top <word> <n>     топ N нечётких
```

Интерактивный режим: `parse`, `lemma`, `fuzzy`, `top`, `import`, `help`.

## Проверка (тесты)

- unit-тест: `Parse("кота")` → ≥1 разбор с NOUN,anim,masc,sing,gent.
- unit-тест: `Parse("все")` → ≥4 разбора (местоимение, глагол, и т.д.).
- unit-тест: `Lemma("кота")` → "кот".
- unit-тест: `Fuzzy("кот", 1)` → содержит "кота", "коты" и т.д.
- unit-тест: `FuzzyTop("кот", 3)` → 3 ближайших слова.
- unit-тест: предсказание для несловарного слова (prediction DAWG).
- integration-тест: 100 случайных слов → все найдены.
- `go test ./pkg/morphology/... -race` — зелёные.

## Ручные проверки

- `gomorphy -dict pymorphy2.dat lookup кота` → корректный разбор.
- `gomorphy -dict pymorphy2.dat lemma котам` → "кот".
- `gomorphy -dict pymorphy2.dat fuzzy кот 1` → список слов.
- Интерактивный режим: все команды работают.

## Реализация (фактический API)

Отклонения от спеки (модель opennota/morph + pymorphy2):

- `Parse` — **не** строит normal через `stem+suffix[0]` вслепую, а по форме:
  `normal = TrimPrefix(word, prefix[form])`, затем
  `TrimSuffix(..., suffix[form])`, потом `normal = prefix[0]+stem+suffix[0]`;
  для `form == 0` normal = слово. payload words.dawg — `2×uint16 BE`
  `(para_id, form_idx)` (в спеке ошибочно `(para_idx<<16)|form_idx`).
- Вероятность: ключ `word + ":" + tag` (граммемы формы), значение
  `p_t_given_w / 1e6`; сортировка `sort.SliceStable` по убыванию только при
  наличии ненулевой prob (двойники-нули сохраняют порядок словаря).
- Prediction повторяет `KnownSuffixAnalyzer` pymorphy2: значения prediction-
  DAWG — 6 байт BE `(count, para, form)`; перебор суффиксов слова (до 5 рун,
  от длинного к короткому), productive-граммемы
  (NUMR, NPRO, PRED, PREP, CONJ, PRCL, INTJ, Apro), break при totalCount>1.
- `Lemma` — через `Parse`, начальная форма = form 0 парадигмы; дедуп по
  `(Normal, Tag)`, омонимы (`кот` NOUN/VERB) сохраняются.
- `Fuzzy` — banded DP (rеферакс `pkg/dictionary/fuzzy.go`), метрика по рунам,
  дедуп по слову, сортировка `(dist, word)`. Терминал узла — по guide
  (`DAWG.HasPayloadChild`), не FollowByte-пробе: проба ловит коллизии
  double-array раскладки (ложные 0x01-ребра у префиксов в testdawg).
- `FuzzyTop` — итеративное расширение расстояния от 0 до верхней границы
  `len(query) + maxRunes` (maxRunes — обход DAG с дедупом узлов по max-глубине;
  без дедупа разделяемые суффиксы считаются экспоненциально).
- Публичный API: новый тип `morphology.Dictionary{d *internal.Dictionary}`,
  `OpenPyMorphy` возвращает `*Dictionary` (спека/этап-12 — `*internal.Dictionary`),
  добавлен `Dictionary.Language()`.
- CLI-адаптация (`import`, `-dict`, интерактивный режим) **перенесена на этап 14**:
  `-dict` открывает скомпилированный `.dat` (SaveTo/Open), которого в этапе 13
  ещё нет; mock-интеграция CLI по полному словарю — когда появится SaveTo.
- roundtrip/BinarySearch по `p_t_given_w` (prob DAWG обращается через `Find`)
  — сохранено для этапа 14 вместе с кодированием `.dat`.

Созданные файлы:
- `pkg/morphology/dictionary.go`, `open.go` (+`open_test.go`) — обёртка.
- `pkg/morphology/parse.go` (+`parse_test.go`), `lemma.go` (+`lemma_test.go`),
  `fuzzy.go` (+`fuzzy_test.go`), `fixture_test.go` (buildFixture).
- `internal`: `PayloadSeparator` (экспорт), `DAWG.ForEachChild`,
  `DAWG.HasPayloadChild`.
