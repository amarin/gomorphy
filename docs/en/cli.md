# CLI: gomorphy

A utility for morphological analysis of words based on a compiled GMOR
dictionary (`.dat`, a unified format for the pymorphy2/OpenCorpora/UniMorph
sources — see [library.md](library.md)), plus a utility for fetching and
building source dictionaries.

## General invocation form

```bash
gomorphy <command> [flags] [args]
gomorphy cli [flags]        # interactive console
```

### Global flags (common to all lookup commands)

| Flag | Description |
|------|----------|
| `-d, --dictionary <path>` | path to a `.dat` file or a directory of `.dat` files (can be given multiple times — dictionaries are merged into a single index, preserving order) |
| `-v, --verbose` | verbose logging |
| `-l, --log <path>` | write the log to a file instead of stderr |

If `-d/--dictionary` is not set, the path is taken from the
`GOMORPHY_DICTIONARY` environment variable.

## Lookup commands

### `lookup` — exact wordform lookup

Returns all grammatical readings for a given word.

```bash
gomorphy lookup -d opencorpora.dat кота
```

Output (one line per reading, tab-separated fields):

```
кота	кот	sing,nomn	para#0/33/1
```

Format: `<word>\t<lemma>\t<tag>\tpara#<dict>/<shard>/<para>` — the `dict`
component indicates which merged dictionary (0-based) the reading came
from, when multiple `-d` flags were given.

### `lemmas` — lemma lookup

Finds the lemma (base form) for a given word.

```bash
gomorphy lemmas -d opencorpora.dat кота
```

Output: `<lemma>\t<lemma's tag>` — one line per homonym (different parts
of speech/meanings of the same text).

### `fuzzy` — fuzzy search

Dictionary words within Levenshtein distance `maxDist` (default 2).

```bash
gomorphy fuzzy -d opencorpora.dat кот 1
```

Output (sorted by distance, then by word):

```
0	кот	dict#0
1	бот	dict#0
1	вот	dict#0
1	гот	dict#0
...
```

Format: `<distance>\t<word>\tdict#<dict>`.

### `top` — N nearest words

Like `fuzzy`, but the distance expands iteratively until `N` words have
been collected.

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

### `cli` — interactive console

```bash
gomorphy cli -d opencorpora.dat
```

```
gomorphy> lookup кота
кота	кот	sing,nomn	para#0/33/1
gomorphy> exit
```

TAB — command autocompletion (`lookup`, `lemmas`, `fuzzy`, `top`, `exit`,
`quit`), `exit`/`quit` — exit.

## Fetching and building dictionaries

### `download` — download the source archive

```bash
gomorphy download opencorpora
gomorphy download pymorphy
gomorphy download unimorph
```

### `unpack` — unpack an already-downloaded archive

```bash
gomorphy unpack opencorpora
gomorphy unpack pymorphy
gomorphy unpack unimorph
```

`unpack unimorph` is a no-op that just confirms the download exists —
UniMorph's downloaded file is already the usable TSV, there's no
archive to extract. Kept only so all three source types go through the
same `download`/`unpack`/`build`/`update` shape.

### `build` — compile a source into `.dat`

By default it uses an already-unpacked source; `-i/--input` lets you
specify the path explicitly (e.g. to compile `dict.xml` directly, without
network access).

```bash
gomorphy build opencorpora -i dict.xml -o out.dat
gomorphy build pymorphy -i /path/to/unpacked/dir -o out.dat
gomorphy build unimorph -i rus.tsv -o out.dat
```

Both `build pymorphy` and `build opencorpora` always recompile
`words.dawg` under a dense 1-byte alphabet — there's no flag to opt out.
This is an agreed default (see
[implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md)),
and `build unimorph` follows the same default (see
[implementation/stage-16-import-unimorph.md](implementation/stage-16-import-unimorph.md)).
Embedding a raw (non-dense) dictionary is Go-API-only: call
`morphology.OpenPyMorphy`, `morphology.CompileFromXML`/
`CompileFromXMLFile`, or `morphology.CompileFromUniMorph`/
`CompileFromUniMorphFile` directly instead of going through the CLI.

Flags:

| Flag | Description |
|------|----------|
| `-i, --input <path>` | compile this path directly, bypassing the loader |
| `-o, --output <path>` | path to the output `.dat` file (default `.data/<type>/<type>.dat`) |
| `--lang <code>` | `unimorph` only; the language to build (default `ru`, currently the only accepted value) |

### `update` — download + unpack + build in one command

```bash
gomorphy update opencorpora
gomorphy update pymorphy -o /tmp/pymorphy.dat
gomorphy update unimorph --lang ru
```

Flags: `-o, --output <path>`, `--lang <code>` — same as `build`.

### `merge` / `split` — not yet implemented

Placeholder commands for future `.dat` dictionary merging/splitting;
currently they fail with a `not yet implemented` error.

### `version`

```bash
gomorphy version
```

Prints the library version.

---

More details on programmatic library usage -> [library.md](library.md).
