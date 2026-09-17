# Источник pymorphy2 (`pkg/pymorphy`) и его интеграция в единый CLI `gomorphy`

## Исходный план (Этап 21, сформулирован 2026-09-15)

Первоначально сформулированная задача — переформатировать
`cmd/gomorphy_build` с «команда одного словаря» на универсальный
конвейер `gomorphy_build <команда> <тип_словаря> [опции]`
(`download`/`unpack`/`compile`/`update` × `opencorpora`/`pymorphy`/
`unimorph`) и добавить второй источник словарных данных — готовые
словари pymorphy2 из PyPI (пакет `pymorphy2-dicts-ru`).

**Обновление 2026-09-16**: этот план поглощён более широким
CLI-редизайном — вместо переформатирования `cmd/gomorphy_build` в
отдельный конвейер оба бинаря (`cmd/gomorphy`, `cmd/gomorphy_build`)
заменены одним cobra-based `gomorphy`. Конкретный интерфейс
`gomorphy_build <command> <dict_type>` как отдельный бинарь не
реализован буквально — но его *суть* (команда, затем тип словаря,
затем опции; единый конвейер download→unpack→compile для нескольких
типов) реализована через подкоманды `gomorphy`. Дизайн:
[2026-09-16-cli-redesign-design.md](../superpowers/specs/2026-09-16-cli-redesign-design.md).
План: [2026-09-16-cli-redesign.md](../superpowers/plans/2026-09-16-cli-redesign.md).

## Что реализовано

### `pkg/pymorphy` — загрузчик готового словаря (по образцу `pkg/opencorpora`)

- `const.go`: `DomainName = "pymorphy"`, `PyPIPackageName =
  "pymorphy2-dicts-ru"`, `PyPIJSONURL` (JSON: `releases`/`urls`, wheel =
  ZIP, поддерево `pymorphy2_dicts_ru/data/`).
- `loader.go`: `Loader` — методы по образцу `opencorpora.Loader`:
  `Sync(skipDownload)` = download+unpack, `DownloadUpdate()` (PyPI JSON
  API → wheel-URL → `.data/pymorphy/pymorphy2-dicts-ru.whl` +
  `version.txt`), `UnpackUpdate()` (zip → `.data/pymorphy/data/`).
- `pypi.go` — разбор PyPI JSON API.

Один сознательный отход от буквального текста исходного плана Этапа 21:
имя скачанного файла фиксировано (`pymorphy2-dicts-ru.whl`, без версии
в имени), версия PyPI хранится отдельно в `version.txt` — проще, чем
парсить версию из имени файла на каждой проверке обновления.

### `cmd/gomorphy` — интеграция в единый CLI

Проверено чтением кода (актуально на момент написания):
- `gomorphy download pymorphy` (`cmd/gomorphy/download.go`) — скачивает
  wheel через `pymorphy.NewLoader("")`.
- `gomorphy unpack pymorphy` (`cmd/gomorphy/unpack.go`) — распаковывает
  в `.data/pymorphy/data/`.
- `gomorphy build pymorphy [-i <dir>] [-o <path>]`
  (`cmd/gomorphy/build.go`) — компилирует через `morphology.OpenPyMorphy`
  + `SaveTo` в единый `.dat` (по умолчанию
  `.data/pymorphy/pymorphy.dat`). Компиляция для pymorphy2
  **опциональна по своей природе** (unpack-результат уже загружается
  напрямую через `morphology.OpenPyMorphy` без компиляции — DAWG/
  prediction уже собраны в исходном пакете), но команда `build`
  унифицирует его с OpenCorpora в единый `.dat`-файл (mmap, один файл).
- `gomorphy update pymorphy` (`cmd/gomorphy/update.go`) — download +
  unpack + build за один вызов.

Итог: и «`compile pymorphy`», и «редизайн CLI» — оба пункта, которые
исходный план Этапа 21 оставлял открытыми, реализованы, просто под
другим синтаксисом команд (`gomorphy <command> <type>` вместо
`gomorphy_build <command> <dict_type>`).

## Автоматические проверки

Покрыто существующими тестами: `pkg/pymorphy/loader_test.go`,
`pkg/pymorphy/pypi_test.go` (разбор PyPI JSON, версии, unpack тестового
zip), `cmd/gomorphy/{download,unpack,build,update}_test.go`.

## Не входит в этот инкремент

- `unimorph` как тип словаря в `download`/`unpack`/`build` — заявлено в
  исходном плане Этапа 21 как «задел на будущее», не реализовано; сам
  импорт UniMorph — отдельный, не начатый Этап 16 (см. [todo.md](../todo.md)).
