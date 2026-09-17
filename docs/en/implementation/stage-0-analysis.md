# Этап 0. Анализ и подготовка репозитория — ВЫПОЛНЕН

## Содержание этапа

- Анализ схемы `dict.xml`, сбор метрик (см. [requirements.md](../requirements.md)).
- Проектирование структуры хранения ([implementation.md](../implementation.md)).
- Фиксация требований FT1–FT9 ([requirements.md](../requirements.md)).
- Удаление устаревшего кода из `internal/` и `pkg/` (объектная модель индекса,
  binutils-сериализация, generic XML-парсер). Сохранены: `pkg/common`,
  `pkg/opencorpora` (только загрузка/распаковка), `cmd/opencorpora_update`
  (загрузка+распаковка).

## Проверка

- `go build ./...`, `go vet ./...`, `go test ./...` — зелёные.
- `go mod tidy && go mod vendor` без лишних зависимостей.
