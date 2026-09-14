# CLI: gomorphy

Утилита морфологического анализа слов на основе скомпилированного словаря
GMOR (`.dat`, единый формат для источников pymorphy2/OpenCorpora/UniMorph
— см. [library.md](library.md)).

## Общая форма вызова

```bash
gomorphy [flags] <command> [args]
gomorphy [flags]                    # интерактивная консоль (без команды)
gomorphy import <source> <path> -o <out.dat>
```

### Флаги

| Флаг | Описание |
|------|----------|
| `-dict <path>` | путь к скомпилированному словарю `.dat` (обязателен для всех команд, кроме `import`) |
| `-o <path>` | выходной файл для `import` |

## Команды

### `lookup` — точный поиск словоформы

Возвращает все грамматические разборы заданного слова.

```bash
gomorphy -dict opencorpora.dat lookup кота
```

Вывод (по одной строке на разбор, поля через TAB):

```
кота	кот	sing,nomn	para#33
```

Формат: `<слово>\t<лемма>\t<тег>\tpara#<id>`.

### `lemmas` — поиск начальных форм

Находит начальную форму (лемму) для заданного слова.

```bash
gomorphy -dict opencorpora.dat lemmas кота
```

Вывод: `<лемма>\t<тег начальной формы>` — по одной строке на омоним
(разные части речи/значения одного текста).

### `fuzzy` — нечёткий поиск

Слова словаря в пределах расстояния Левенштейна `maxDist` (по умолчанию 2).

```bash
gomorphy -dict opencorpora.dat fuzzy кот 1
```

Вывод (отсортирован по расстоянию, затем по слову):

```
0	кот
1	бот
1	вот
1	гот
...
```

Формат: `<расстояние>\t<слово>`.

### `top` — N ближайших слов

Как `fuzzy`, но расстояние расширяется итеративно, пока не набрано `N` слов.

```bash
gomorphy -dict opencorpora.dat top кот 5
```

```
0	кот
1	бот
1	вот
1	гот
1	дот
```

## Интерактивная консоль

Запустите без команды (с `-dict`) для входа в интерактивный режим:

```bash
gomorphy -dict opencorpora.dat
```

```
gomorphy> lookup кота
кота	кот	sing,nomn	para#33
gomorphy> exit
```

TAB — автодополнение команд (`lookup`, `lemmas`, `fuzzy`, `top`, `import`),
`exit`/`quit` — выход.

## Импорт словарей — `gomorphy import`

```bash
gomorphy import pymorphy2 <dir> -o out.dat
gomorphy import opencorpora <dict.xml> -o out.dat
```

`-o` должен идти после `<source> <path>` (или в любом месте — CLI сам
достаёт `-o`/`-o=...` из позиционных аргументов).

## CLI: gomorphy_build

Утилита для полного цикла работы со словарём OpenCorpora: загрузка,
распаковка, компиляция (для одноразовых/CI-сценариев; для программного
импорта нескольких источников используйте `gomorphy import`, см. выше).

```bash
gomorphy_build <command> [flags]
```

### Команды

| Команда | Описание |
|---|---|
| `update` | Скачать (если нужно) + распаковать + скомпилировать |
| `compile` | Скомпилировать уже распакованный `dict.xml` (без сети) |

### Флаги

Важно: флаги `flag`-пакета Go должны идти **до** команды, не после
(`gomorphy_build -o out.dat compile`, не `gomorphy_build compile -o out.dat`).

| Флаг | Описание |
|------|----------|
| `-l` | не скачивать, использовать локальный `dict.xml` (только для `update`) |
| `-d` | отладочное логирование |
| `-o <path>` | путь выходного `.dat`-файла (по умолчанию — `.data/opencorpora/opencorpora.dat`) |

### Примеры

```bash
# Полный цикл: скачивание + распаковка + компиляция
gomorphy_build update

# Пропустить скачивание, только компиляция уже распакованного dict.xml
gomorphy_build update -l

# Компиляция dict.xml без обращения к сети/загрузчику
gomorphy_build compile

# Свой путь для результата
gomorphy_build -o /tmp/oc.dat compile
```

По умолчанию результат сохраняется в `.data/opencorpora/opencorpora.dat`.

Подробнее о программном использовании библиотеки → [library.md](library.md).
