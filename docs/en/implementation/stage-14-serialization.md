# Этап 14. Сериализация: единый формат на диске

## Содержание этапа

Определение и реализация единого формата файла для хранения словаря.
Формат совместим с pymorphy2 (words.dawg + paradigms.array + строки)
или является его расширением с дополнительными секциями.

### Формат файла (`pkg/morphology/internal/format.go`)

```
┌──────────────────────────────────┐
│  Заголовок                       │
│  magic  "GMOR"  4 байта          │
│  version    u32                  │
│  checksum   xxh3-64  8 байт      │
├──────────────────────────────────┤
│  Каталог секций                  │
│  count      u16                  │
│  entries:                          │
│    name     [16]byte             │
│    offset   u64                  │
│    size     u64                  │
│    flags    u8  (сжатие, etc.)   │
├──────────────────────────────────┤
│  Секции данных                   │
│  meta, tagset, suffixes,         │
│  prefixes, paradigms,            │
│  words.dawg, prediction-N,       │
│  probability                      │
└──────────────────────────────────┘
```

### Секции

- **meta**: язык, количество лемм/форм/парадигм, версия формата.
- **tagset**: JSON-массив строк имён граммем.
- **suffixes**: varint-length-prefixed строки (пул суффиксов).
- **prefixes**: varint-length-prefixed строки (пул префиксов).
- **paradigms**: uint16 array (N суффиксов + N тегов + N префиксов на парадигму).
- **words.dawg**: dictionary uint32[] + guide byte[] (как в pymorphy2).
- **prediction-N**: prediction DAWGs (опционально).
- **probability**: probability DAWG (опционально).

### Запись (`pkg/morphology/save.go`)

```go
func (d *Dictionary) SaveTo(path string) error
```

1. Сериализация компонентов:
   - tagset → JSON
   - suffixes → varint-length-prefixed строки
   - prefixes → varint-length-prefixed строки
   - paradigms → uint16 array
   - words.dawg → raw bytes
2. Запись заголовка + каталога + секций.
3. Опционально: zstd-сжатие холодных секций (suffixes, prefixes, tagset, paradigms).
   Флаг `flag_compressed` в каталоге.

### Чтение (`pkg/morphology/open.go`)

```go
func Open(path string) (*Dictionary, error)
```

1. mmap файла, чтение заголовка и каталога.
2. Для каждой секции:
   - Если флаг compressed — декомпрессия (zstd) в память.
   - Если не compressed — алиас mmap-региона (zero-copy).
3. Парсинг компонентов: tagset из JSON, suffixes/prefixes из varint-encoded,
   paradigms из uint16 array, DAWG из dictionary+guide.
4. Возврат иммутабельного `Dictionary`.

### Формат совместимости с pymorphy2

Для чтения напрямую из директории pymorphy2 (без конвертации) —
`OpenPyMorphy(dir)` читает файлы формата pymorphy2 напрямую
(этап 12). Формат `GMOR` — расширение с additional секциями
(meta, prediction, probability).

## Проверка (тесты)

- unit-тест: roundtrip ImportFromDir → SaveTo → Open → Parse идентичны.
- unit-тест: формат-версия корректно записывается и читается.
- unit-тест: повреждённый файл → осмысленная ошибка (magic mismatch, checksum).
- unit-тест: сжатая секция декомпрессируется корректно.
- unit-тест: mmap-алиас для words.dawg (zero-copy).
- `go test ./pkg/morphology/... -race` — зелёные.

## Ручные проверки

- Сравнить размер `.dat` с размером директории pymorphy2.
- `gomorphy -dict pymorphy2.dat lookup кота` → идентично прямой загрузке.
- Загрузка через mmap: `time gomorphy -dict pymorphy2.dat lookup кота`.

## Реализация (итог)

- `pkg/morphology/internal/format.go` — контейнер GMOR: header
  (magic `GMOR` | version u32 | checksum xxh3-64), каталог
  (count u16; entry: name[16] | offset u64 | size u64 | flags u8),
  секции. Смещения секций выравниваются по 8 байтам (0 по модулю 8),
  поэтому массив единиц words.dawg (offset+4) 4-байтово выровнен и
  алиасится из mmap без копирования. Проверки: magic, version, checksum,
  каталог (bounds, дубли имён, усечение).
- Кодеки секций: `EncodeMeta`/`DecodeMeta` (язык + CharPolicy),
  `EncodeTagSet`/`DecodeTagSet` (JSON {name,tags}),
  `EncodeStrings`/`DecodeStrings` (uvarint-length-prefixed),
  `EncodeParadigms`/`DecodeParadigms` (u32 count; len + u16[] на парадигму).
- `internal.DAWG.Bytes()` / `internal.ParseDAWG` — потоковый формат
  words.dawg; `ParseDAWG` алиасит dictionary/guide при выравнивании
  (zero-copy mmap), иначе копирует. `Paradigm.Data()` для сериализации.
- `pkg/morphology.SaveTo` и `Open` (+ `Dictionary.Close`, жизненный цикл
  mmap-региона). `Open` держит mmap до `Close`.
- CLI (`cmd/gomorphy`) переведён на `pkg/morphology`:
  `import pymorphy2 <dir> -o <file.dat>`, `-dict <file.dat>` + lookup/lemmas/
  fuzzy/top; флаг `-o` после подкоманды разбирается вручную (flag-пакет
  останавливается на первом нефлаге). End-to-end тест CLI строит бинарь
  и прогоняет import→lookup→lemmas→fuzzy.

### Отклонения от плана

- **zstd не подключён** в этом этапе: флаг `FlagCompressed` в каталоге
  предусмотрен, чтение сжатой секции возвращает осмысленную ошибку.
  Реальное сжатие — в этапе 17 (сужение типов + zstd), где для этого
  добавится кодек (klauspost/compress).
