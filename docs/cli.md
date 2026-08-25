# CLI: gomorphy

Интерактивная утилита морфологического анализа русских слов на основе скомпилированного словаря OpenCorpora.

## Общая форма вызова

```bash
gomorphy [flags] <command> [args]
gomorphy [flags]                    # интерактивная консоль
```

### Флаги

| Флаг | Описание |
|------|----------|
| `-dict <path>` | путь к скомпилированному словарю (обязательный) |

## Команды

### `lookup` — точный поиск словоформы

Возвращает все грамматические разборы заданного слова.

```bash
gomorphy -dict .data/opencorpora/opencorpora.dict lookup кота
```

Вывод:

```
кота  NOUN,anim,masc,sing,gent  lemma#140411
кота  NOUN,anim,masc,sing,accs  lemma#140411
```

Формат: `<слово>  <грамматика>  lemma#<id>`

### `lemmas` — поиск начальных форм

Находит начальную форму (лемму) для заданного слова.

```bash
gomorphy -dict .data/opencorpora/opencorpora.dict lemmas кота
```

Вывод:

```
#140411  кот  NOUN,anim,masc
```

Формат: `#<id>  <лемма>  <базовые граммемы>`

### `fuzzy` — нечёткий поиск

Поиск слов с расстоянием Левенштейна не более `maxDist` (по умолчанию 2).

```bash
gomorphy -dict .data/opencorpora/opencorpora.dict fuzzy кот 1
```

Вывод (результаты сгруппированы по расстоянию):

```
0  кот
1  код
1  крот
```

Формат: `<расстояние>  <слово>`

### `top` — ближайшие N слов

Находит N ближайших слов по расстоянию Левенштейна.

```bash
gomorphy -dict .data/opencorpora/opencorpora.dict top кот 5
```

Вывод:

```
0  кот
1  бот
1  вот
1  гот
1  дот
```

## Интерактивная консоль

Запустите без команды для входа в интерактивный режим:

```bash
gomorphy -dict .data/opencorpora/opencorpora.dict
```

```
gomorphy> lookup кота
кота  NOUN,anim,masc,sing,gent  lemma#140411
кота  NOUN,anim,masc,sing,accs  lemma#140411

gomorphy> exit
```

Поддерживается TAB-автодополнение команд.

## CLI: opencorpora_update

Утилита загрузки, распаковки и компиляции словаря OpenCorpora.

```bash
opencorpora_update [flags]
```

### Флаги

| Флаг | Описание |
|------|----------|
| `-l` | использовать локальный файл, пропустить скачивание |
| `-skip-compile` | только загрузить и распаковать, не компилировать |
| `-v` | включить отладочное логирование |
| `-h` | показать справку |

### Примеры

```bash
# Полный цикл: скачивание + распаковка + компиляция
opencorpora_update

# Только компиляция из уже скачанного dict.xml
opencorpora_update -l

# Только загрузка без компиляции
opencorpora_update -skip-compile
```

Скомпилированный словарь сохраняется в `.data/opencorpora/opencorpora.dict`.

Подробнее о программном использовании библиотеки → [library.md](library.md).
