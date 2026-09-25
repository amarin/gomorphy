# Установка

## Требования

- Go 1.25+ (директива `go` следует политике «текущий релиз Go минус два минорных версии» и повышается при выходе новой минорной версии Go)
- ОС: только Unix (Linux, macOS, BSD) — `Open` загружает словарь через
  `syscall.Mmap` (см. [code-review-pre-1.0.md](../en/code-review-pre-1.0.md),
  англ.). Сборка под Windows компилируется, но `Open` возвращает ошибку во
  время выполнения, поэтому CLI `gomorphy` там тоже не может открыть
  словарь. `OpenBytes` работает на Windows (без mmap) — это текущий
  обходной путь для загрузки словаря там (например, встроить `.dat`-файл
  через `//go:embed` и передать его байты) — см.
  [todo.md](../en/todo.md#windows-native-mmap--planned-post-10-backlog),
  «Windows: native mmap».

## Установка CLI-утилиты

```bash
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
```

Утилита будет доступна в `$GOPATH/bin` (убедитесь, что директория добавлена в `$PATH`).

`gomorphy` — единый бинарь: морфологический анализ по скомпилированному
словарю (`lookup`/`lemmas`/`fuzzy`/`top`, интерактивная консоль `cli`), а
также загрузка, распаковка и компиляция исходных словарей
(`download`/`unpack`/`build`/`update`), импорт TSV словоформ
(`import tsv`) и объединение словарей (`merge`).

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
| `make update` | Собрать `gomorphy` и выполнить полный цикл: скачать + распаковать + собрать словарь OpenCorpora (`gomorphy update opencorpora`; opencorpora.org сейчас недоступен — см. ниже) |
| `make compile` | Собрать `gomorphy` и скомпилировать уже распакованный `dict.xml` (`gomorphy build opencorpora`) |
| `make test` | Запуск unit-тестов с детектором гонок |
| `make test-integration` | Интеграционные тесты (`-tags integration`; часть требует сеть и реальные данные OpenCorpora в `.data/`) |
| `make lint` | golangci-lint |
| `make clean` | Удаление артефактов сборки (`./deploy/`) |

## Загрузка словаря

После сборки (`make build`) загрузите и соберите словарь pymorphy2
(`pymorphy2-dicts-ru`, с PyPI) — рекомендуемый источник:

```bash
./deploy/gomorphy update pymorphy
```

Результат: файл `.data/pymorphy/pymorphy.dat` (около 16 МБ).

OpenCorpora (`dict.opcorpora.xml.bz2` с opencorpora.org) по-прежнему
поддерживается как альтернатива, но opencorpora.org сейчас недоступен, и
его дальнейшая доступность не гарантируется:

```bash
./deploy/gomorphy update opencorpora
```

Результат: файл `.data/opencorpora/opencorpora.dat` (около 10 МБ, зависит
от версии `dict.xml`).

Подробнее о работе со словарём через CLI → [cli.md](cli.md).
