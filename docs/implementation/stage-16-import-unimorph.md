# Этап 16. Импорт UniMorph

## Содержание этапа

Импорт словаря UniMorph из TSV-файла (`лемма<TAB>словоформа<TAB>bundle`) как
дополнительного источника данных. Аналогичен этапу 15 (Импорт OpenCorpora),
но проще: формат без XML, строки разбираются потоково. Полный анализ формата —
в [docs/unimorph.md](../unimorph.md).

Ориентир данных: `unimorph/rus` (473 482 строки, 28 069 лемм, 353 004
словоформы, лицензия CC-BY-SA 3.0).

### Импортёр (`pkg/morphology/importers/unimorph/`)

Функция `ImportFromTSV(r io.Reader, opts Options) (*Dictionary, error)`:

#### Pipeline

1. **Считывание.** Потоковое чтение (`bufio.Scanner`, лимит 1 МБ на строку —
   строки длинные для составных токенов). Разбиение строки по `\t` ровно на
   3 поля: лемма, словоформа, bundle.
2. **Лемма.** Каждая новая лемма → `AddLemma(лемма, baseGrammemes...)`.
   Base-граммемы — bundle строки, где словоформа совпадает с леммой; если
   такой строки нет — пустой набор. Если столбец леммы пуст — леммой считать
   саму словоформу.
3. **Формы.** Каждая строка → `AddForm(lemmaID, словоформа, features...)`,
   где `features = strings.Split(bundle, ";")`. Порядок признаков сохраняется
   (анкоды в gomorphy упорядочены).
4. **Теги (по умолчанию opaque).** Признаки UniMorph (`N;ACC;SG` и т.п.)
   интернируются как есть — как независимый TagSet. Опция `Options.Mapping`
   позволяет спроецировать на другой набор (OpenCorpora-теги) — маппинг
   табличный, неполные соответствия либо игнорируются, либо сохраняются как
   opaque.
5. **Компиляция.** `Compile()` → иммутабельный `Dictionary` → `SaveTo` —
   единый `.dat`-формат, общий для всех источников.

#### Обработка(и данных) UniMorph

| Случай | Обработка |
|---|---|
| Синкретизм (текст с разными bundles) | Нативно: несколько чтений словоформы |
| Словоформа == лемма | Привязка к лемме, base-граммемы из её bundle |
| Пустая лемма | Лемма := словоформа |
| Многословные/дефисные токены | Библиотека работает с произвольными байтами |
| `LGSPEC *`/неизвестные признаки | opaque-граммема, не ошибка |
| Повтор `(form, bundle)` внутри леммы | Дедуплицируется Builder (пары интернируются) |

### Публичный API

```go
// В pkg/morphology/
func CompileFromUniMorph(r io.Reader) (*Dictionary, error)
func CompileFromUniMorphFile(path string) (*Dictionary, error)
```

`CompileFromUniMorphFile` читает TSV-файл, строит Dictionary.
Обёртка над `unimorph.ImportFromTSV`.

### CLI

```bash
gomorphy import unimorph <rus.tsv> -o ru-unimorph.dat
gomorphy -dict ru-unimorph.dat lookup кота
```

## Проверка (тесты)

- unit-тест: мини-TSV (5–10 лемм) → корректные леммы и парадигмы.
- unit-тест: roundtrip ImportFromTSV → SaveTo → Open → Parse идентичен.
- unit-тест: синкретизм — одна словоформа с несколькими bundles → все чтения.
- unit-тест: пустая лемма → лемма == словоформа.
- unit-тест: маппинг тегов (opaque и `Mapping`) — оба пути.
- integration-тест: весь `rus` → `Lookup`/`Lemmas` на выборочных словах
  (`кота` → `N;ACC;SG`, лемма `кот`).
- integration-тест: парадигмы < лемм (для `rus`: 28 068 < 28 069).
- `go test ./pkg/morphology/... -race` — зелёные.

## Ручные проверки

- `gomorphy import unimorph rus -o ru-unimorph.dat` → файл создан.
- `gomorphy -dict ru-unimorph.dat lookup кота` → `N;ACC;SG`.
- Другой язык (например, `eng`): `gomorphy -dict en.dat lookup cats` —
  импортёр работает без изменений (FT10).
- Сверка по словам, общим с OpenCorpora-словарём.