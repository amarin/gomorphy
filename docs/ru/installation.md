# Установка

## Требования

- Go 1.25+ (директива `go` следует политике «текущий релиз Go минус два минорных версии» и повышается при выходе новой минорной версии Go)
- ОС: Linux, macOS (amd64/arm64); Windows не поддерживается — библиотека
  использует mmap через пакет `syscall` напрямую (см.
  `docs/code-review-pre-1.0.md`)

## Установка CLI-утилиты

```bash
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
```

Утилита будет доступна в `$GOPATH/bin` (убедитесь, что директория добавлена в `$PATH`).

`gomorphy` — единый бинарь: морфологический анализ по скомпилированному
словарю (`lookup`/`lemmas`/`fuzzy`/`top`, интерактивная консоль `cli`), а
также загрузка, распаковка и компиляция исходных словарей
(`download`/`unpack`/`build`/`update`).

Подробнее → [cli.md](cli.md).

## Установка как Go-библиотеки

```bash
go get github.com/amarin/gomorphy/pkg/morphology
```

Импорт в коде:

```go
import "github.com/amarin/gomorphy/pkg/morphology"
```

Подробнее о публичном API → [library.md](library.md).

## Сборка из исходников

```bash
git clone https://github.com/amarin/gomorphy.git
cd gomorphy
make build     # скомпилирует CLI-утилиту в ./deploy/
```

Доступные цели Makefile:

| Цель | Описание |
|------|----------|
| `make build` | Сборка CLI-утилиты (`gomorphy`) в `./deploy/` |
| `make update` | Собрать `gomorphy` и выполнить полный цикл: скачать + распаковать + собрать словарь OpenCorpora (`gomorphy update opencorpora`) |
| `make compile` | Собрать `gomorphy` и скомпилировать уже распакованный `dict.xml` (`gomorphy build opencorpora`) |
| `make test` | Запуск unit-тестов с детектором гонок |
| `make test-integration` | Интеграционные тесты (`-tags integration`; часть требует сеть и реальные данные OpenCorpora в `.data/`) |
| `make lint` | golangci-lint |
| `make clean` | Удаление артефактов сборки (`./deploy/`) |

## Загрузка словаря OpenCorpora

После сборки (`make build`) загрузите и соберите словарь:

```bash
./deploy/gomorphy update opencorpora
```

Результат: файл `.data/opencorpora/opencorpora.dat` (десятки МБ, зависит
от версии `dict.xml`).

Подробнее о работе со словарём через CLI → [cli.md](cli.md).
