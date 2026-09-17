# CLI: gomorphy

Утилита морфологического анализа слов на основе скомпилированного словаря
GMOR (`.dat`, единый формат для источников pymorphy2/OpenCorpora/UniMorph
— см. [library.md](library.md)), а также утилита получения и сборки
исходных словарей.

## Общая форма вызова

```bash
gomorphy <command> [flags] [args]
gomorphy cli [flags]        # интерактивная консоль
```

### Глобальные флаги (общие для всех команд поиска)

| Флаг | Описание |
|------|----------|
| `-d, --dictionary <path>` | путь к `.dat`-файлу или каталогу с `.dat`-файлами (можно указывать несколько раз — словари объединяются в один индекс с сохранением порядка) |
| `-v, --verbose` | подробное логирование |
| `-l, --log <path>` | писать лог в файл вместо stderr |

Если `-d/--dictionary` не задан, используется путь из переменной окружения
`GOMORPHY_DICTIONARY`.

## Команды поиска

### `lookup` — точный поиск словоформы

Возвращает все грамматические разборы заданного слова.

```bash
gomorphy lookup -d opencorpora.dat кота
```

Вывод (по одной строке на разбор, поля через TAB):

```
кота	кот	sing,nomn	para#0/33/1
```

Формат: `<слово>\t<лемма>\t<тег>\tpara#<dict>/<shard>/<para>` — компонент
`dict` указывает, из какого по счёту (начиная с 0) объединённого словаря
пришёл разбор, если указано несколько `-d`.

### `lemmas` — поиск начальных форм

Находит начальную форму (лемму) для заданного слова.

```bash
gomorphy lemmas -d opencorpora.dat кота
```

Вывод: `<лемма>\t<тег начальной формы>` — по одной строке на омоним
(разные части речи/значения одного текста).

### `fuzzy` — нечёткий поиск

Слова словаря в пределах расстояния Левенштейна `maxDist` (по умолчанию 2).

```bash
gomorphy fuzzy -d opencorpora.dat кот 1
```

Вывод (отсортирован по расстоянию, затем по слову):

```
0	кот	dict#0
1	бот	dict#0
1	вот	dict#0
1	гот	dict#0
...
```

Формат: `<расстояние>\t<слово>\tdict#<dict>`.

### `top` — N ближайших слов

Как `fuzzy`, но расстояние расширяется итеративно, пока не набрано `N` слов.

```bash
gomorphy top -d opencorpora.dat кот 5
```

```
0	кот	dict#0
1	бот	dict#0
1	вот	dict#0
1	гот	dict#0
1	дот	dict#0
```

### `cli` — интерактивная консоль

```bash
gomorphy cli -d opencorpora.dat
```

```
gomorphy> lookup кота
кота	кот	sing,nomn	para#0/33/1
gomorphy> exit
```

TAB — автодополнение команд (`lookup`, `lemmas`, `fuzzy`, `top`, `exit`,
`quit`), `exit`/`quit` — выход.

## Получение и сборка словарей

### `download` — скачать исходный архив

```bash
gomorphy download opencorpora
gomorphy download pymorphy
```

### `unpack` — распаковать уже скачанный архив

```bash
gomorphy unpack opencorpora
gomorphy unpack pymorphy
```

### `build` — скомпилировать источник в `.dat`

По умолчанию берёт уже распакованный источник; `-i/--input` позволяет
указать путь явно (например, чтобы скомпилировать `dict.xml` напрямую,
без сети).

```bash
gomorphy build opencorpora -i dict.xml -o out.dat
gomorphy build pymorphy -i /path/to/unpacked/dir -o out.dat
```

Флаги:

| Флаг | Описание |
|------|----------|
| `-i, --input <path>` | скомпилировать этот путь напрямую, минуя загрузчик |
| `-o, --output <path>` | путь выходного `.dat`-файла (по умолчанию `.data/<type>/<type>.dat`) |

### `update` — download + unpack + build одной командой

```bash
gomorphy update opencorpora
gomorphy update pymorphy -o /tmp/pymorphy.dat
```

Флаги: `-o, --output <path>` — как у `build`.

### `merge` / `split` — пока не реализованы

Команды-заглушки для будущего объединения/разбиения `.dat`-словарей;
сейчас завершаются ошибкой `not yet implemented`.

### `version`

```bash
gomorphy version
```

Печатает версию библиотеки.

---

Подробнее о программном использовании библиотеки → [library.md](library.md).
