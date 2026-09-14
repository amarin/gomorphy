# Документация gomorphy

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

## Этапы новой реализации (план)

- [Этап 11. Внутренний формат: TagSet + Paradigm + DAWG reader](implementation/stage-11-internal-format.md)
- [Этап 12. Импорт PyMorphy2](implementation/stage-12-import-pymorphy2.md)
- [Этап 13. Публичный API: Parse, Lemma, Fuzzy](implementation/stage-13-public-api.md)
- [Этап 14. Сериализация: единый формат на диске](implementation/stage-14-serialization.md)
- [Этап 15. Импорт OpenCorpora](implementation/stage-15-import-opencorpora.md)
- [Этап 16. Импорт UniMorph](implementation/stage-16-import-unimorph.md)
- [Этап 17. Сужение типов ID и zstd](implementation/stage-17-optimize.md)
- [Этап 18. Финализация: CLI, документация, тесты](implementation/stage-18-finalize.md)
- [Этап 19. Тематические словари: TSV-импорт, CLI-батчи, навыки, решение по MCP](todo.md)
- [Этап 20. База синонимов: группы, теги, sidecar-файл](todo.md)

## Решения

- [Почему нет встроенного MCP-сервера](mcp.md)
