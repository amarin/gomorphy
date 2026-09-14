# Этап 18. Финализация: CLI, документация, тесты

## Содержание этапа

Обновление CLI (все команды через аргументы + интерактивный режим),
финализация документации, полный прогон тестов, обновление README.

### CLI (`cmd/gomorphy/main.go`)

Все команды доступны через CLI-аргументы:

```bash
# Точный поиск
gomorphy -dict <path> lookup <word>

# Начальная форма (лемма)
gomorphy -dict <path> lemma <word>

# Нечёткий поиск
gomorphy -dict <path> fuzzy <word> <maxDist>
gomorphy -dict <path> top <word> <maxWords>

# Импорт
gomorphy import pymorphy2 <dir> -o <path>
gomorphy import opencorpora <dict.xml> -o <path>
gomorphy import unimorph <rus.tsv> -o <path>

# Интерактивный режим
gomorphy -dict <path>
# > parse кота
# > lemma котам
# > fuzzy кот 1
# > top кот 3
# > import pymorphy2 /path/to/dir
# > help
```

### Мульти-словарь на уровне CLI

```bash
# Основной словарь
gomorphy -dict pymorphy2.dat lookup кота

# Дополнительный словарь (отдельный вызов)
gomorphy -dict oc.dat lookup кота
```

Пользователь управляет словарями сам. Библиотека экземпляра `Dictionary`
не хранит глобального состояния.

### Документация

- `README.md`: обновление описания, примеры использования с новым API.
- `docs/`: все страницы актуальны, навигация корректна.
- godoc: все экспортирующие функции с комментариями.
- Примеры: `examples/` — минимальные примеры для каждого API.

### Навык «использование словаря gomorphy»

- `skills/use-dictionary/SKILL.md` — навык для агента: lookup/lemmas/fuzzy/top,
  импорт (pymorphy2 / opencorpora / unimorph), работа с несколькими `.dat`,
  интерактивный режим. Оформляется как `SKILL.md` в репозитории
  (версионируется вместе с библиотекой, копируется в конфигурацию агента).

### cmd/opencorpora_update

```bash
gomorphy update                    # загрузка + компиляция в едином формате
gomorphy update --skip-download    # только компиляция из локального файла
```

## Проверка (тесты)

- Все unit-тесты зелёные: `go test ./... -count=1`.
- Все integration-тесты зелёные.
- Race detector: `go test ./... -race`.
- Vet: `go vet ./...` без замечаний.
- Build: `go build ./...` без ошибок.
- Линтер: `golangci-lint run` без ошибок (если настроен).

## Ручные проверки

- Полный цикл pymorphy2:
  ```bash
  gomorphy import pymorphy2 /path/to/pymorphy2-dicts-ru -o pymorphy2.dat
  gomorphy -dict pymorphy2.dat lookup кота
  gomorphy -dict pymorphy2.dat lemma котам
  gomorphy -dict pymorphy2.dat fuzzy кот 1
  gomorphy -dict pymorphy2.dat top кот 3
  ```
- Полный цикл OpenCorpora:
  ```bash
  gomorphy import opencorpora dict.xml -o oc.dat
  gomorphy -dict oc.dat lookup кота
  gomorphy -dict oc.dat fuzzy кот 1
  ```
- Полный цикл UniMorph:
  ```bash
  gomorphy import unimorph rus -o ru-unimorph.dat
  gomorphy -dict ru-unimorph.dat lookup кота
  gomorphy -dict ru-unimorph.dat fuzzy кот 1
  ```
- Интерактивный режим: все команды работают.
- Tab-completion: работает в bash/zsh.
