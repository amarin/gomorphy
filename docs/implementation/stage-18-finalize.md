# Этап 18. Финализация: документация, тесты

> **Обновление**: раздел «CLI» ниже описывает исходный план (флаги
> `-dict`, `import pymorphy2/opencorpora/unimorph`, tab-completion) —
> он **не реализован буквально**, но CLI полностью переработан отдельно
> (Этап 21, `gomorphy` на cobra) и закрывает суть задачи иначе: команды
> `lookup`/`lemmas`/`fuzzy`/`top`/`cli`/`download`/`unpack`/`build`/`update`,
> флаг `-d/--dictionary` (можно указывать несколько раз — объединяются в
> `MultiDictionary`). Актуальное описание CLI — [docs/cli.md](../cli.md),
> история редизайна —
> [pymorphy-source-and-cli.md](pymorphy-source-and-cli.md). Раздел «CLI»
> ниже сохранён как есть для истории планирования, не как актуальная
> спецификация. Оставшийся объём этого этапа — документация, тесты,
> метрики; см. «Итоговые метрики» ниже, обновлено по факту.

## Содержание этапа (исходный план, частично устарел — см. врезку выше)

Обновление CLI (все команды через аргументы + интерактивный режим),
финализация документации, полный прогон тестов, обновление README.

### CLI (`cmd/gomorphy/main.go`) — исходный план, см. врезку выше

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

### Мульти-словарь на уровне CLI — исходный план, см. врезку выше

Реализовано шире, чем планировалось: `-d/--dictionary` можно указывать
несколько раз в одной команде — CLI сам объединяет их в один
`MultiDictionary`, не требуя от пользователя отдельных вызовов. См.
[docs/cli.md](../cli.md) и [multi-dict.md](multi-dict.md).

### Документация — ВЫПОЛНЕНО 2026-09-17

- `README.md`: описание, Quick start, Project structure — приведены в
  соответствие текущему коду (было: старая архитектура `pkg/dictionary`,
  размер `.dat` ~300 МБ; стало: `pkg/morphology`, ~13 МБ).
- `docs/library.md`: переписан полностью под текущий `pkg/morphology`
  API (`Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`/`MultiDictionary`/`SaveTo`) —
  старая версия описывала API, которого больше не существует
  (`dictionary.Open`, `Wordform`, `Builder.AddGrammeme`).
- `docs/cli.md` — уже был актуален, без изменений.
- `docs/index.md` — добавлены ссылки на новые `implementation/*.md`
  (multi-dict, плотный алфавит, источник pymorphy2, tag-mapping),
  добавлен раздел «Использование», расширен список исследований.
- godoc: аудит экспортированных символов `pkg/morphology` и его
  подпакетов (`tagmap`, `importers/*`), `pkg/pymorphy`, `pkg/opencorpora`,
  `pkg/common` — все документированы; попутно найден и удалён мёртвый
  код без единого использования (`pkg/common/interfaces.go`).
- Примеры (`examples/`), навык для агента (`skills/use-dictionary/`) —
  **вынесены из этого этапа отдельной будущей задачей**, см.
  `docs/todo.md` — это новый контент со своим дизайном, не исправление
  документации.

### cmd/opencorpora_update — исходный план, см. врезку выше

Реализовано иначе: `gomorphy update <type>` (`opencorpora`/`pymorphy`,
без флага `--skip-download` — для «только компиляция из локального
файла» есть отдельная команда `gomorphy build <type> -i <path>`). См.
[docs/cli.md](../cli.md).

## Проверка (тесты)

Прогнано и подтверждено (дата последней проверки — см. `git log` по
этому файлу):

- `go build ./...` — без ошибок.
- `go vet ./...` — без замечаний.
- `go test -race -count=1 ./...` — 274/274 зелёных, 12 пакетов.
- `golangci-lint run ./...` — без замечаний (по факту финализации нашлись
  и исправлены 2 давних известных находки — errcheck на `defer
  opened.Close()` в тесте, staticcheck S1016 на ручной struct-литерал
  вместо конверсии типа в `pkg/morphology/importers/opencorpora/import.go`
  — обе упоминались как «pre-existing, unrelated» в спеке multi-dict,
  закрыты в рамках этого этапа).
- Попутно найден и удалён мёртвый код: `pkg/common/interfaces.go`
  (`Dictionary`/`DomainDataLoader` — экспортированные интерфейсы без
  единого использования в кодовой базе, наследие до-редизайна).
- Integration-тесты (`-tags=integration`, нужны реальные словарные
  данные) — не гоняются в CI/по умолчанию; см. `docs/todo.md` про
  переменные окружения (`GOMORPHY_DICT_XML`, `GOMORPHY_PYMORPHY2_DIR`).

## Ручные проверки

Актуальные команды — см. [docs/cli.md](../cli.md). Полный цикл на
реальных данных:

```bash
gomorphy update pymorphy
gomorphy -d .data/pymorphy/pymorphy.dat lookup кота
gomorphy -d .data/pymorphy/pymorphy.dat lemmas кота
gomorphy -d .data/pymorphy/pymorphy.dat fuzzy кот 1
gomorphy -d .data/pymorphy/pymorphy.dat top кот 3

gomorphy update opencorpora
gomorphy -d .data/opencorpora/opencorpora.dat lookup кота

# Несколько словарей одним индексом
gomorphy -d .data/opencorpora/opencorpora.dat -d .data/pymorphy/pymorphy.dat lookup кота

gomorphy cli -d .data/opencorpora/opencorpora.dat   # интерактивная консоль
```

UniMorph-цикл (`gomorphy import unimorph .../update unimorph`) —
недоступен, Этап 16 не начат.

Полный shell-автокомплит (bash/zsh для самой команды `gomorphy`) —
**не реализован**; есть только автодополнение имён команд внутри
интерактивной консоли `gomorphy cli` (TAB — `lookup`/`lemmas`/`fuzzy`/
`top`/`exit`/`quit`, см. [docs/cli.md](../cli.md)).

## Итоговые метрики

> Обновление 2026-09-17: таблица ниже заменена на реально измеренные
> факты, где измерение возможно; расходные ожидания-без-измерения
> (`ожидаемые на момент планирования`) убраны как недостоверные, а не
> оставлены рядом с фактом. Латентность операций (Parse/Lemma/Fuzzy) в
> кодовой базе **не бенчмаркается** — ни одной функции `Benchmark*`
> нет; строки ниже честно помечены как неизмеренные, а не заполнены
> оценкой на глаз.

| Показатель | Значение | Как измерено |
|---|---|---|
| Размер `.dat` (OpenCorpora, полный словарь) | ~13.3 МБ | `ls -la .data/opencorpora/opencorpora.dat`, 2026-09-17 |
| Размер `.dat` (pymorphy2, полный словарь) | ~14.0 МБ | `ls -la .data/pymorphy/pymorphy.dat`, 2026-09-17 |
| Исходный `dict.xml` (OpenCorpora) | ~401 МБ | для сравнения — во сколько раз компактнее `.dat` |
| Размер `.dat` с zstd | — | не реализовано, см. `docs/todo.md`, Этап 17 остаток |
| Загрузка (`Open`) | mmap, без полного чтения в память | архитектурно (mmap-backed секции), не бенчмаркалось в мкс/мс |
| Parse/Lemma/Fuzzy latency | — | **не бенчмаркается** — нет `Benchmark*` в кодовой базе |
| Поддержка pymorphy2 | да | `OpenPyMorphy`/`OpenPyMorphyDense` |
| Поддержка OpenCorpora | да | `CompileFromXML(File)` |
| Поддержка UniMorph (TSV) | нет | Этап 16 не начат, см. `docs/todo.md` |
| Множественные словари | да, из библиотеки (`MultiDictionary`) | не «на уровне приложения» — теперь часть API |
| Тесты | 274/274, `-race`, 12 пакетов | `go test -race -count=1 ./...`, 2026-09-17 |
| Линтер | без замечаний | `golangci-lint run ./...`, 2026-09-17 |
