# Этап 1. Примитивы формата (internal/format) — ВЫПОЛНЕН

## Содержание этапа

Инкремент: пакет кодирования и секционного контейнера.

- varint/delta кодеки (`[]uint32`, `[]uint64`) — encode/decode roundtrip.
- Заголовок: magic `"GMRF"`, version, xxh3-чексумма; каталог секций
  (имя, offset, size); writer (секции дописываются, каталог в конце)
  и reader (проверки magic/version/чексуммы, доступ к секции по имени).

## Проверка (выполнена)

- unit-тесты roundtrip varint/delta на краевых случаях (0, 1, max u32/u64,
  монотонные/немонотонные последовательности) — `TestAppendDelta*Roundtrip`.
- unit-тесты writer→reader: несколько секций (в т.ч. пустых, streaming),
  чтение по имени, ошибки при битой чексумме/магии/версии/обрезке.
- `go build ./...`, `go vet ./...`, `go test ./internal/format -race` — зелёные.

## Ручная проверка

Hexdump тестового файла — magic `"GMRF"`, version=1, indexOffset=0x1C,
каталог из 2 секций, trailer xxh3 — читаемы и корректны.
