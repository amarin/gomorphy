# Установка

## Требования

- Go 1.27.1+
- ОС: Linux, macOS (amd64/arm64); Windows не поддерживается — библиотека
  использует mmap через пакет `syscall` напрямую (см.
  `docs/code-review-pre-1.0.md`)

## Установка CLI-утилит

```bash
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
go install github.com/amarin/gomorphy/cmd/gomorphy_build@latest
```

Утилиты будут доступны в `$GOPATH/bin` (убедитесь, что директория добавлена в `$PATH`).

- `gomorphy` — морфологический анализ по скомпилированному словарю
  (`lookup`/`lemmas`/`fuzzy`/`top`, интерактивная консоль, `import`).
- `gomorphy_build` — загрузка, распаковка и компиляция словаря OpenCorpora
  (`update`/`compile`).

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
make build     # скомпилирует CLI-утилиты в ./deploy/
```

Доступные цели Makefile:

| Цель | Описание |
|------|----------|
| `make build` | Сборка CLI-утилит (`gomorphy`, `gomorphy_build`) в `./deploy/` |
| `make update` | Собрать `gomorphy_build` и выполнить полный цикл: скачать + распаковать + скомпилировать словарь OpenCorpora |
| `make compile` | Собрать `gomorphy_build` и скомпилировать уже распакованный `dict.xml` |
| `make test` | Запуск unit-тестов с детектором гонок |
| `make test-integration` | Интеграционные тесты (`-tags integration`; часть требует сеть и реальные данные OpenCorpora в `.data/`) |
| `make lint` | golangci-lint |
| `make clean` | Удаление артефактов сборки (`./deploy/`) |

## Загрузка словаря OpenCorpora

После сборки (`make build`) загрузите и скомпилируйте словарь:

```bash
./deploy/gomorphy_build update
```

Результат: файл `.data/opencorpora/opencorpora.dat` (десятки МБ, зависит
от версии `dict.xml`).

Подробнее о работе со словарём через CLI → [cli.md](cli.md).
