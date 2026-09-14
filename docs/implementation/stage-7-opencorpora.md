# Этап 7. Интеграция с OpenCorpora end-to-end — ВЫПОЛНЕН

## Содержание этапа

Инкремент: полный цикл FT7.

- `dictionary.CompileFromXML(path)`: xmlscan → Builder → снимок
  (+ `SaveToAtomic`: temp-файл + rename).
- `cmd/opencorpora_update`: флаги `-l` (download), `-skip-compile`, `-v`;
  компиляция выполняется автоматически после unpack, итог с временем/памятью.
- `pkg/opencorpora.Update()` снова вызывает компиляцию (`loader.Compile()`).

## Проверка (выполнена)

- ручной прогон `go run ./cmd/opencorpora_update -l` на реальном `dict.xml`:
  `opencorpora.dict` 303 MB создан за 13.7s (scan+build+save), peak heap ~4 GB.
- smoke-тест (`pkg/dictionary/smoke_integration_test.go`): 50 частотных слов
  резолвятся; контрольные слова («кота»→кот sing,gent; «домами»; омонимы
  «стекла», «пила», «бежал» ≥2 леммы) сверены по фактическим данным словаря.

## Замечание для этапа 8

В `dict.xml` формы несут только собственные `<g>`-теги
(падеж/число); POS и константные теги живут на лемме — при FT2/FT5 выдаче
полную грамматическую характеристику нужно собирать из леммы + формы.
