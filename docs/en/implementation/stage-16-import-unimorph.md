# Этап 16. Импорт UniMorph

> **Статус:** черновик обновлён решениями Q&A (2026-09-16). Реализация ещё не
> начата. См. также `docs/research/0006-unimorph-import-plan.md` — полный
> анализ, объём оценки и подробные мотивации.

## Содержание этапа

Импорт словаря UniMorph из TSV-файла (`лемма<TAB>словоформа<TAB>bundle`) как
дополнительного источника данных. Аналогичен этапу 15 (Импорт OpenCorpora),
но проще: формат без XML, строки разбираются потоково. Полный анализ формата —
в [docs/unimorph.md](../unimorph.md).

Ориентир данных: `unimorph/rus` (473 482 строки, 28 069 лемм, 353 004
словоформ, лицензия CC-BY-SA 3.0).

### Ключевые решения (зафиксированы в Q&A 2026-09-16)

1. **Лемма = форма 0 парадигмы (всегда).** Если строка с `form == lemma`
   существует — её bundle становится тегом формы 0. Если нет — синтезировать
   форму 0: текст = лемма, тег = `""`. Текст леммы всегда включается во вход
   `lcp()`, что обеспечивает корректную реконструкцию нормы для
   супплетивных групп («я»/«меня», «человек»/«люди»).
2. **Теги — opaque-строки.** Bundle (например, `N;ACC;SG`) сохраняется в
   TagSet вербатим; переупорядочивание и маппинг на OpenCorpora-теги **не
   реализуются** на этапе 16 (отложено, Q2).
3. **Язык и CharPolicy.** На первом этапе — только `ru` (`Options.Language`);
   для других языков — ошибка. CharPolicy выбирается по языку: `ru` →
   `internal.RussianCharPolicy()`, иначе nil. Код строится с заделом на
   будущие языки (Q1).
4. **Битые строки ≠ 3 полей.** Ошибка выводится (логируется), но импорт
   продолжается — пропускается текущая строка (Q5). Пустой bundle (третий
   столбец) — допустим (тег `""`).
5. **Загрузчик данных** — по образцу `pkg/pymorphy`, в пакете `pkg/unimorph`
   (Q4), а не ручная загрузка.

### Импортёр (`pkg/morphology/importers/unimorph/`)

```go
func ImportFromTSV(
    r io.Reader,
    tagSet *internal.TagSet,   // nil → NewTagSet("unimorph")
    opts Options,
) (*internal.Dictionary, error)
```

`Options`:
```go
type Options struct {
    Language    string                     // "ru" по умолчанию
    CharPolicy *internal.CharPolicy        // nil → infer по Language
    OnMalformed func(lineNumber int, text string) // nil → молчаливый пропуск
    Progress    Progress                   // nil → без прогресса
}
```

#### Pipeline

1. **Считывание.** `bufio.Scanner` с буфером 1 МБ (составные/дефисные токены),
   строка — ровно 3 поля через `\t`, отрезание `\r` у 3-го поля. Строка с
   пустым леммой → лемма := словоформа. Пропуск пустых строк (без ошибки).
2. **Аккумуляция.** Данные сгруппированы в `map[лемма]*lemmaEntry`
   (28K записей для `rus` — дёшево; порядок строк по лемме не гарантирован,
   используется произвольный порядок ключей карты).
3. **Форма 0.** После полной загрузки строк для каждого lemma:
   - Найти строку с `text == lemma` → поместить её на позицию 0 форм
     (первое вхождение). Остальные строки с тем же текстом — обычные формы
     (синкретизм формы леммы).
   - Если строк `text == lemma` нет → синтезировать форму 0: `{text: lemma,
     gramm: ""}`.
4. **Стем.** Вычислить LCP (руно-безопасный) по списку форм (включая форму 0).
   Суффиксы для каждой формы — `word[len(stem):]`.
5. **Теги.** Bundle каждой формы → `tagSet.Add(bundle)` (opaque-строка;
   переупорядочивание не выполняется).
6. **Дедуп парадигм.** Суффиксы, теги и префиксы (все `""` для UniMorph)
   идентифицируются целочисленными ID. Парадигмы дедуплицируются
   (`paradigmKeyHash`); шардирование — `FillOnDemand` (лимит 65 536 суффиксов
   на шард).
7. **DAWG.** Ключ — `prefix+stem+suffix` (= словоформа для UniMorph),
   значение — `paraID<<16|formIdx`. Построение через `BuildDAWGWithValuesProgress`.
8. **Сборка.** `internal.NewDictionary(language, tagSet, suffixes, prefixes,
   paradigms, dawg, charPolicy)` → `SaveTo`.

#### Обработка данных UniMorph

| Случай | Обработка |
|---|---|
| Синкретизм (текст с разными bundles) | Нативно: несколько чтений словоформы |
| Словоформа == лемма | Форма 0 с её bundle; другие — обычные формы |
| Словоформа == лемма (несколько строк) | Первая → форма 0; остальные — обычные формы (омонимы лемм. формы) |
| Пустая лемма | Лемма := словоформа (форма 0 = словоформа, тег её bundle) |
| Многословные/дефисные токены | Библиотека работает с произвольными байтами |
| `LGSPEC *`/неизвестные признаки | opaque-граммема, не ошибка |
| Супплетивизм («я»/«меня», «человек»/«люди») | Корректно: стем LCP по формам+лемма (форма 0 включена), норма = lemma |
| Нет строки `form == lemma` | Синтез формы 0: текст=lemma, тег=`""`; добавляет ~28K ключей в DAWG |
| Повтор `(form, bundle)` внутри леммы | Дедуп парадигм (пары интернируются) |
| Битые строки (≠3 поля) | OnMalformed + пропуск строки; импорт продолжается |
| Пустой bundle | Тег `""` в TagSet — допустим |
| 1 шард для `rus` (353K форм) | Механизм шардирования используется для страховки; переключится для крупных языков |

### Публичный API (`pkg/morphology/`)

```go
type UniMorphOptions = unimorph.Options

func CompileFromUniMorph(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFile(path string, opts UniMorphOptions) (*Dictionary, error)
```

Обёртки над `unimorph.CompileFromTSV`; Language из opts — язык словаря
(пока только `"ru"`).

### CLI

```bash
gomorphy download unimorph             # загрузить rus в .data/unimorph/
gomorphy import unimorph <file> -lang ru -o ru-unimorph.dat
gomorphy -dict ru-unimorph.dat lookup кота  # → N;ACC;SG
```

Флаг `-lang` (default `ru`). CLI `download/unpack/build/unpack` для типа
`unimorph` аналогичны pymorphy/opencorpora.

`CompileFromUniMorphFile` читает TSV-файл, строит Dictionary.
Обёртка над `unimorph.ImportFromTSV`.

### CLI

```bash
gomorphy import unimorph <rus.tsv> -o ru-unimorph.dat
gomorphy -dict ru-unimorph.dat lookup кота
```

## Проверка (тесты)

- unit-тест: мини-TSV (5–10 лемм) → корректные леммы и парадигмы.
- unit-тест: синтетическая форма 0 (лемма не встречается как форма) → `Parse(лемма)`,
  `Lemma(form)` возвращает лемму, тег `""`.
- unit-тест: roundtrip ImportFromTSV → SaveTo → Open → Parse идентичен.
- unit-тест: синкретизм — одна словоформа с несколькими bundles → все чтения.
- unit-тест: супплетивизм («я»/«меня», «человек»/«люди») → норма = лемма.
- unit-тест: пустая лемма → лемма == словоформа.
- unit-тест: битые строки (≠3 полей) → OnMalformed вызывается, импорт продолжается.
- unit-тест: дедуп `(form, bundle)` и парадигм.
- integration-тест: весь `rus` → `Lookup`/`Lemmas` на выборочных словах
  (`кота` → `N;ACC;SG`, лемма `кот`).
- integration-тест: парадигмы < лемм (для `rus`: 28 068 < 28 069).
- integration-тест: кросс-сверка общих слов с OpenCorpora-словарём.
- `go test ./pkg/morphology/... -race` — зелёные.

## Ручные проверки

- `gomorphy download unimorph` → `rus`-файл в `.data/unimorph/`.
- `gomorphy import unimorph <file> -lang ru -o ru-unimorph.dat` → файл создан.
- `gomorphy -dict ru-unimorph.dat lookup кота` → `N;ACC;SG`.
- Другой язык (например, `eng`): `gomorphy -dict en.dat lookup cats` — на этапе
  16 `-lang` принимает только `ru` (остальные — ошибка), задел под другие
  языки оставлен в сигнатурах (Q1).
- Сверка по словам, общим с OpenCorpora-словарём.