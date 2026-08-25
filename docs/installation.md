# Установка

## Требования

- Go 1.22+
-操作系统: Linux, macOS, Windows (amd64/arm64)

## Установка CLI-утилит

```bash
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
go install github.com/amarin/gomorphy/cmd/opencorpora_update@latest
```

Утилиты будут доступны в `$GOPATH/bin` (убедитесь, что директория добавлена в `$PATH`).

## Установка как Go-библиотеки

```bash
go get github.com/amarin/gomorphy/pkg/dictionary
```

Импорт в коде:

```go
import "github.com/amarin/gomorphy/pkg/dictionary"
```

## Сборка из исходников

```bash
git clone https://github.com/amarin/gomorphy.git
cd gomorphy
make build     # скомпилирует CLI-утилиты в ./deploy/
```

Доступные цели Makefile:

| Цель | Описание |
|------|----------|
| `make build` | Сборка CLI-утилит |
| `make test` | Запуск unit-тестов с детектором гонок |
| `make test-integration` | Интеграционные тесты (требуется скомпилированный словарь) |
| `make lint` | golangci-lint |
| `make clean` | Удаление артефактов сборки |

## Загрузка словаря OpenCorpora

После установки CLI-утилит загрузите и скомпилируйте словарь:

```bash
opencorpora_update -l   # загрузка + распаковка + компиляция
```

Результат: файл `.data/opencorpora/opencorpora.dict` (~300 МБ).

Подробнее о работе со словарём через CLI → [cli.md](cli.md).
