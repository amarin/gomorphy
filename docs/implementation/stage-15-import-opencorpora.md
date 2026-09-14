# Этап 15. Импорт OpenCorpora — ВЫПОЛНЕН

## Содержание этапа

Импорт словаря OpenCorpora (`dict.xml`) через извлечение парадигм из лемм.
Переиспользование `internal/xmlscan` для чтения XML. Результат — тот же
внутренний формат (paradigm + DAWG с payload).

### Импортёр (`pkg/morphology/importers/opencorpora/`)

Функция `ImportFromXML(r io.Reader, tagSet *TagSet) (*Dictionary, error)`:

#### Pipeline

1. **Сканирование XML** через `xmlscan`:
   - Событие `Grammeme`: сбор имён граммем (опционально, используется для <grammeme> с <name>).
   - Событие `Lemma`: начало леммы (id, text).
   - Событие `Form`: словоформа (text из атрибута `t`, грамммы из `<g>` тегов).

2. **Извлечение парадигм** (core algorithm):
   ```
   Для каждой леммы:
     Собрать все формы: forms = [{text, gramm}, ...]
     Вычислить стем: stem = LCP(forms.map(_.text))
     Для каждой формы:
       suffix = form.text[len(stem):]
       tag_id = tagSet.ID(form.gramm)  // комбинированная строка тегов
       парадигма.add(suffix_id, tag_id, 0)  // prefix = 0 (пустой)
     Дедуплицировать парадигму (hash → paradigmID)
     Сохранить пару (stem, paradigmID)
   ```

3. **Построение DAWG**:
   ```
   Для каждого (stem, paradigmID):
     Для каждой формы парадигмы (form_idx):
       key = stem + suffix[form_idx] + "\x01" + base64(para_id << 16 | form_idx)
   BuildDAWGWithValues(keys, values)
   ```

4. **Возврат Dictionary** с наполненными Suffixes, Prefixes (nil), Paradigms, Words.

#### Дедупликация парадигм

Парадигма = упорядоченный список `(suffix_id, tag_id)` (prefix = 0 для
OpenCorpora). Хэш вычисляется как сериализация байтов suffixes + tags.
Результат: из ~391K лемм → ~3K уникальных парадигм (для полного dict.xml).

#### TagSet

Теги собираются из `<g v="...">` тегов в каждой форме. Каждый уникальный
`gramm` (комбинированная строка типа `"NOUN,anim,masc,sing,nomn"`) добавляется
в TagSet через `Add()`.

### Публичный API

```go
// В pkg/morphology/
func CompileFromXML(r io.Reader) (*Dictionary, error)
func CompileFromXMLFile(path string) (*Dictionary, error)
```

`CompileFromXMLFile` открывает `dict.xml`, парсит граммемы, строит
Dictionary. `CompileFromXML` — обёртка над `opencorpora.ImportFromXML`.

### CLI

```bash
gomorphy import opencorpora dict.xml -o oc.dat
gomorphy -dict oc.dat lookup кота
```

## Проверка (выполнена)

- unit-тест: маленький XML (3 леммы) → парадигмы извлечены корректно.
- unit-тест: число парадигм ≤ число лемм (dedup работает).
- unit-тест: стем = LCP всех форм (проверка суффикса "" = index 0).
- unit-тест: DAWG через SimilarItems находит все словоформы.
- unit-тест: roundtrip — значения из DAWG имеют правильную структуру (4 байта).
- unit-тест: леммы без словоформ игнорируются.
- `go test ./pkg/morphology/importers/opencorpora/... -count=1` — зелёные.

## Ручные проверки

- `gomorphy import opencorpora dict.xml -o oc.dat` → файл создан.
- `gomorphy -dict oc.dat lookup кота` → корректный разбор.

## Реализация (итог)

- `pkg/morphology/importers/opencorpora/import.go` — полный импортёр:
  xmlscan Handler → LCP-stem → suffix ID map → paradigm dedup → DAWG.
- `internal.BuildDAWGWithValues` — расширение dawgdic builder для хранения
  значений как payload (ключ = `word\x01<base64_value>`).
- `pkg/morphology/open.go` — добавлены `CompileFromXML` и `CompileFromXMLFile`.
- `cmd/gomorphy/main.go` — подкоманда `import opencorpora <xml> -o <out.dat>`.

### Отклонения от плана

- **Suffixes vs paradigms**: в оригинальном плане suffixes — отдельный пул,
  но фактическая реализация хранит suffix texts в `Dictionary.Suffixes` как
  ordered list из `suffixTexts` map. Это совместимо с pymorphy2 importer.
- **Prefixes**: OpenCorpora не хранит префиксы в парадигмах (леммы уже
  содержат полный текст), поэтому `Prefixes = nil`.

