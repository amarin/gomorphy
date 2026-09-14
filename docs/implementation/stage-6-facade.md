# Этап 6. Публичный фасад pkg/dictionary (FT7–FT9) — ВЫПОЛНЕН

## Содержание этапа

Инкремент: библиотечный API.

- `Open`/`NewEmpty`/`Lookup`/`Lemmas`/`Fuzzy`(заглушка до этапа 9)/`SaveTo`/`Builder`
  (Builder — фасад над `build.Builder`: `NewBuilder` + `AddGrammeme`/`AddLemma`/
  `AddForm`/`Compile`; `Dictionary.Builder()` реконструирует наполнение из снимка).
- Конкурентность: чтение из N горутин; два независимых экземпляра одновременно.

## Проверка (выполнена)

- `go test ./pkg/dictionary -race`: конкурентное чтение, параллельные экземпляры.
- пример использования в `example_test.go` (godoc-пример с Output).
- отсутствие глобальных переменных: только immutable-snapshot поля,
  закрытие через `atomic.Bool` (`ErrClosed`), пакетного состояния нет.
