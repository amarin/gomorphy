# Документация gomorphy

> Russian version (installation/CLI/library basics only): [docs/ru/index.md](../ru/index.md).

## Требования

- [Требования к библиотеке](requirements.md)
- [План работ](todo.md)

## Реализация

- [Реализация (as-is + план)](implementation.md)
- [Глоссарий](glossary.md)

## Анализ источников данных

- [UniMorph: схема и формат данных](unimorph.md)

## Этапы исходной реализации (выполнены)

- [Этап 0. Анализ и подготовка репозитория](implementation/stage-0-analysis.md)
- [Этап 1. Примитивы формата](implementation/stage-1-format.md)
- [Этап 2. Интернирование строк](implementation/stage-2-intern.md)
- [Этап 3. Сканер dict.xml](implementation/stage-3-xmlscan.md)
- [Этап 4. Builder и CSR-структуры](implementation/stage-4-builder-csr.md)
- [Этап 5. Компилятор и загрузчик файла](implementation/stage-5-compiler-loader.md)
- [Этап 6. Публичный фасад pkg/dictionary](implementation/stage-6-facade.md)
- [Этап 7. Интеграция с OpenCorpora end-to-end](implementation/stage-7-opencorpora.md)
- [Этап 8. Поиск лемм FT5](implementation/stage-8-lemmas.md)
- [Этап 9. Нечёткий поиск FT6](implementation/stage-9-fuzzy.md)
- [Этап 10. Финализация](implementation/stage-10-finalize.md)

## Этапы новой реализации (paradigm + DAWG; 11-15 выполнены, 17 частично, 16/18 — план)

- [Обоснование редизайна хранилища](implementation/redesign-rationale.md)
- [Этап 11. Внутренний формат: TagSet + Paradigm + DAWG reader](implementation/stage-11-internal-format.md)
- [Этап 12. Импорт PyMorphy2](implementation/stage-12-import-pymorphy2.md)
- [Этап 13. Публичный API: Parse, Lemma, Fuzzy](implementation/stage-13-public-api.md)
- [Этап 14. Сериализация: единый формат на диске](implementation/stage-14-serialization.md)
- [Этап 15. Импорт OpenCorpora](implementation/stage-15-import-opencorpora.md)
- [Этап 16. Импорт UniMorph (план)](implementation/stage-16-import-unimorph.md)
- [Ускорение сборки DAWG (free-list)](implementation/dawg-freelist-optimization.md)
- [Исправление минимизации DAWG (баг в chainSig)](implementation/dawg-minimization-fix.md)
- [Этап 17. Сужение типов ID + формат-задел под сжатие](implementation/stage-17-optimize.md)
- [Секция info: метаданные сборки словаря](implementation/info-section.md)
- [Ревью кода перед 1.0.0: разбор находок + оба критических бага](implementation/code-review-pre-1.0-triage.md)
- [Multi-dict: `morphology.MultiDictionary`](implementation/multi-dict.md)
- [Плотный 1-байтовый DAWG-алфавит для pymorphy2](implementation/pymorphy2-dense-alphabet.md)
- [Источник pymorphy2 (`pkg/pymorphy`) и интеграция в CLI `gomorphy`](implementation/pymorphy-source-and-cli.md)
- [Универсальный маппинг тегов между словарями (`pkg/morphology/tagmap`)](implementation/tag-mapping.md)
- [Этап 18. Финализация: документация, тесты (план)](implementation/stage-18-finalize.md)
- [Этап 19. Тематические словари: TSV-импорт, CLI-батчи, навыки, решение по MCP](todo.md)
- [Этап 20. База синонимов: группы, теги, sidecar-файл](todo.md)

## Использование

- [CLI: gomorphy](cli.md)
- [Программное использование библиотеки](library.md)
- [Установка](installation.md)

## Ревью и находки

- [Ревью кода перед 1.0.0 (полный отчёт)](code-review-pre-1.0.md)

## Исследования

- [Плотность упаковки DAWG при плотном алфавите меток](research/0001-dawg-alphabet-density.md)
- [Бинарное кодирование парадигм и списка тегов вместо текста](research/0002-paradigm-tagset-binary-encoding.md)
- [Сравнительные парадигмы не мержились (Cmp2/«по-»)](research/0003-comparative-paradigms-not-merging.md)
- [Плотный DAWG-алфавит с payload](research/0004-dawg-dense-alphabet-with-payload.md)
- [Стоимость полного обхода pymorphy2 words.dawg](research/0005-pymorphy2-full-dawg-walk-cost.md)
- [План импорта UniMorph](research/0006-unimorph-import-plan.md)
- [План импорта Universal Dependencies](research/0007-universal-dependencies-import-plan.md)
- [Оценка реализуемости экспорта словарей (pymorphy2/OpenCorpora)](research/0008-dictionary-export-feasibility.md)

## Решения

- [Почему нет встроенного MCP-сервера](mcp.md)
