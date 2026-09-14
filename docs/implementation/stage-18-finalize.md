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

## Итоговые метрики (ожидаемые на момент планирования этапа)

> Обновление 2026-09-14: строка «Размер .dat (opencorpora)» устарела —
> после фикса минимизации DAWG (см.
> [dawg-minimization-fix.md](dawg-minimization-fix.md)) реальный размер
> уже 14.6 МБ, то есть цель по размеру уже перевыполнена без zstd.
> Остальные строки — как планировались изначально, не проверены на
> практике.

| Показатель | До редизайна | После этапа 18 (план) |
|---|---|---|
| Размер .dat (pymorphy2) | — | ~15–20 МБ |
| Размер .dat (opencorpora) | 305 МБ | ~~~20–30 МБ~~ уже 14.6 МБ (2026-09-14) |
| Размер .dat с zstd | — | ~10–15 МБ |
| Загрузка | mmap, мс | mmap, мс |
| Parse (точный) | < 10 мкс | < 10 мкс |
| Parse (предсказание) | нет | < 50 мкс |
| Lemmas | < 10 мкс | < 10 мкс |
| Fuzzy k≤2 | доли сек | доли сек |
| Поддержка pymorphy2 | нет | да |
| Поддержка OpenCorpora | да | да (новый формат) |
| Поддержка UniMorph (TSV) | нет | да (169 языков) |
| Множественные словари | на уровне приложения | на уровне приложения |
