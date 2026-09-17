# Этап 12. Импорт PyMorphy2

## Содержание этапа

Загрузка словаря PyMorphy2 из директории (words.dawg + paradigms.array +
suffixes.json + gramtab-*.json) → иммутабельный `Dictionary`.

### Импортёр (`pkg/morphology/importers/pymorphy2/`)

Функция `ImportFromDir(dir string) (*Dictionary, error)`:

1. **Чтение `gramtab-opencorpora-int.json`**: JSON-массив строк → TagSet.
   Каждая строка — тег вида `NOUN,anim,masc,sing,nomn`.

2. **Чтение `suffixes.json`**: JSON-массив строк → `[]string` (пул суффиксов).
   Индекс в массиве = suffix_id в парадигмах.

3. **Чтение `paradigm-prefixes.json`** (опционально): JSON-массив строк →
   `[]string` (пул префиксов). Если файл отсутствует — `["", "по", "наи"]`.

4. **Чтение `paradigms.array`**:
   - uint16 count — количество парадигм
   - Для каждой парадигмы: uint16 len + []uint16 data (len × 3 значений:
     суффиксы + теги + префиксы)

5. **Чтение `words.dawg`**: dictionary uint32[] + guide byte[] → DAWG.
   Формат: uint32 size → size × uint32 dictionary → uint32 guide_size →
   guide_size × 2 байта guide.

6. **Опционально: `prediction-suffixes-N.dawg`**: DAWG предсказаний.
   Количество = количество префиксов (len(prefixes)).

7. **Опционально: `p_t_given_w.intdawg`**: DAWG вероятностей.

### Публичный API

```go
// В pkg/morphology/
func OpenPyMorphy(dir string) (*Dictionary, error)
```

Обёртка над `pymorphy2.ImportFromDir`. Устанавливает `CharPolicy` для русского
(е→ё) по умолчанию.

## Проверка (тесты)

- unit-тест: ImportFromDir с тестовой директорией (маленький DAWG + парадигмы).
- unit-тест: roundtrip — ImportFromDir → SaveTo → Open → данные идентичны.
- integration-тест: полный словарь pymorphy2-dicts-ru:
  - Parse("все") → ≥4 разбора
  - Parse("кота") → разбор с NOUN,anim,masc,sing,gent
  - Parse("кот") → разбор с NOUN,anim,masc,sing,nomn
- `go test ./pkg/morphology/... -race` — зелёные.

## Ручные проверки

- `pip install pymorphy2-dicts-ru` → `gomorphy import pymorphy2 $(python -c "...") -o pymorphy2.dat`
- `gomorphy -dict pymorphy2.dat lookup кота` → корректный разбор
- Сравнить с выводом `python -c "import pymorphy2; print(pymorphy2骆d骆骆().parse('кота'))"`

## Реализация (фактический API)

Пакет `pkg/morphology/importers/pymorphy2` (package `pymorphy2`) + обёртка
`pkg/morphology.OpenPyMorphy`. Референс форматов — `opennota/morph`
morph.go/dict.go/guide.go.

- `pymorphy2.ImportFromDir(dir) (*internal.Dictionary, error)` — прямое чтение:
  - `gramtab-opencorpora-int.json` → JSON-массив строк → `TagSet` (пакет
    `opencorpora-int`);
  - `paradigm-prefixes.json` → `Prefixes`; при отсутствии — `["", "по", "наи"]`
    (как в opennota);
  - `suffixes.json` → `Suffixes`;
  - `paradigms.array` — `uint16 count` + для каждой `uint16 len` + `len×uint16`
    LE → `NewParadigmFromData` (длина обязана делиться на 3);
  - `words.dawg` → `ReadDAWG` (dictionary+guide);
  - опц. `p_t_given_w.intdawg` → `Probability`; опц. `prediction-suffixes-{i}.dawg`
    для `i < len(prefixes)` → `Prediction` (пропуск недостающих файлов);
  - `Language="ru"`, `CharPolicy=RussianCharPolicy()`.
- Внутренние примитивы: `internal.NewParadigmFromData(data []uint16)` —
  приём сырых данных pymorphy2 без копирования.
- Тест-инфраструктура: билдер DAWG-фикстур вынесен из internal-тестов в
  `pkg/morphology/internal/testdawg` (`Build`, `Marshal`) — переиспользуется
  unit-тестами импортёра и public-обёртки. Фикстуры пишут реальные файлы
  директории pymorphy2 (включая слова с `0x01` payload, base64-значения).
- Примечания:
  - roundtrip `SaveTo`/`Open` отложен до этапа 14 (сериализация) — в todo.md.
  - integration-тест (`//go:build integration`) полного словаря требует
    `GOMORPHY_PYMORPHY2_DIR`; без него — `t.Skip`. Проверяет структуру
    (~3K парадигм, ~5K суффиксов, ~1K тегов, 3 prediction, probability)
    и ≥4 разбора «все» через `Words.SimilarItems`.
