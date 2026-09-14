# Этап 17. Сужение типов ID и zstd

## Содержание этапа

Оптимизация размера файла за счёт:
1. Сужения типов: uint16 для paradigm/suffix/tag/prefix IDs.
2. zstd-сжатия холодных секций (suffixes, prefixes, tagset, paradigms).

### Сужение типов

Текущие ограничения (измерены на OpenCorpora):
- Парадигмы: ~3K → uint16 (65K макс)
- Суффиксы: ~5K → uint16 (65K макс)
- Префиксы: ~3 → uint16
- Теги: ~1K → uint16 (65K макс)

Paradigm плоский массив: `[]uint16` вместо `[]uint32`.
Экономия: ~40% на paradigms секции.

### zstd-сжатие

Холодные секции (содержатся при загрузке, декомпрессируются один раз):
- `suffixes` — строки, хорошо сжимаются
- `prefixes` — строки
- `tagset` — JSON
- `paradigms` — uint16 array (частичная дедупликация)

Горячие секции (используются при каждом lookup):
- `words.dawg` — **не сжимается** (mmap zero-copy)

Флаг сжатия в каталоге секций: `flags & flag_compressed != 0`.
Декомпрессия при загрузке: `zstd.NewReader` → bytes.Buffer.

### Ожидаемый эффект

| Секция | До сжатия | После сжатия |
|---|---|---|
| suffixes | ~0.5 МБ | ~0.3 МБ |
| prefixes | ~0.1 МБ | ~0.05 МБ |
| tagset | ~0.1 МБ | ~0.05 МБ |
| paradigms | ~3–4 МБ | ~1–2 МБ |
| **Итого** | ~4–5 МБ | ~2–3 МБ |

Экономия: ~2 МБ. Не критично для pymorphy2 (~15 МБ), но значимо
для OpenCorpora (~305 МБ → ~20–25 МБ).

## Проверка (тесты)

- unit-тест: roundtrip сжатый → несжатый → данные идентичны.
- unit-тест: сжатая секция меньше несжатой (assert size < threshold).
- unit-тест: roundtrip несжатый → данные идентичны (backward compat).
- benchmark: Parse до и после (ожидается 0% regression на words.dawg).
- `go test ./pkg/morphology/... -race` — зелёные.

## Ручные проверки

- `make compile`: сравнить размер `.dat` до и после.
- `gomorphy -dict pymorphy2.dat lookup кота` → идентично.
- Загрузка: `time gomorphy -dict pymorphy2.dat lookup кота` → без регрессии.
