# Этап 2. Интернирование строк (internal/intern, internal/stringsx) — ВЫПОЛНЕН

## Содержание этапа

Инкремент: арена строк + таблица интернирования без аллокаций на токен.

- `stringsx.Arena`: append байт, границы по uint32-оффсетам, `Get(i) []byte`.
- `intern.Table`: хеш `[]byte` → id, open addressing, pre-sized, rehash;
  метод `Intern(b []byte) (id uint32, existed bool)`, `Get(id) []byte`.

## Проверка (выполнена)

- unit-тесты Arena/Intern: дубликаты дают один id; уникальные — новые;
  корректность Get после роста таблицы (rehash на 50k вставок при ожидании 4);
  похожие слова дают разные id.
- benchmark Intern на 1M строк: повторные вставки — 63 ns/op,
  **0 allocs/op** (требование FT1 выполнено);
  полная сборка уникального словаря 1M строк — 71 мс, 23 аллокации
  (амортизированный рост арены и rehash).
- `go test ./internal/intern ./internal/stringsx -bench .` — зелёные,
  полный `-race` прогон всех пакетов зелёный.
