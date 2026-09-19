# Programmatic Library Usage

Import:

```go
import "github.com/amarin/gomorphy/pkg/morphology"
```

The entire public API lives in the `morphology` package — the nested
`morphology/internal` package is not exported and is not meant for direct
use.

Runnable, self-contained programs for each entry point below live in
[examples/](../../examples/README.md) — `go run ./examples/<name>`.
Each entry point also has a matching `ExampleXxx` function in
`pkg/morphology/example_test.go`, which pkg.go.dev/godoc renders
inline on that function's own doc page.

## Opening a dictionary

Entry points, all returning `*morphology.Dictionary`:

```go
func Open(path string) (*Dictionary, error)
func OpenPyMorphy(dir string) (*Dictionary, error)
func OpenPyMorphyDense(dir string) (*Dictionary, error)
func CompileFromXML(r io.Reader, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromXMLFile(path string, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromXMLDense(r io.Reader, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromXMLFileDense(path string, progress opencorpora.Progress) (*Dictionary, error)
func CompileFromUniMorph(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFile(path string, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphDense(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFileDense(path string, opts UniMorphOptions) (*Dictionary, error)
```

- **`Open(path)`** — loads an already-compiled unified format
  (`.dat`, sections + mmap). Hot sections (`words.dawg`) are mapped
  without copying; the result must be closed with `Close()`.
- **`OpenPyMorphy(dir)`** — reads a pymorphy2 dictionary directly from
  a directory of source files (`words.dawg`, `paradigms.array`,
  ...), without pre-compiling into `.dat`.
- **`OpenPyMorphyDense(dir)`** — like `OpenPyMorphy`, but rebuilds
  `words.dawg` with a dense 1-byte alphabet (see
  [implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md)).
  Produces the same readings, `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop` results,
  as `OpenPyMorphy`, but faster and more compact in memory. `SaveTo`/
  `Open` round-trip it like any other dictionary. This is what
  `gomorphy build pymorphy` uses by default (see [cli.md](cli.md)) — call
  `OpenPyMorphy` directly instead if you specifically want the raw,
  non-dense variant.
- **`CompileFromXML`/`CompileFromXMLFile`** — compiles an OpenCorpora
  dictionary from `dict.xml`. `progress` is an optional callback
  `func(processed, total int)` for tracking compilation progress (`nil`
  can be passed).
- **`CompileFromXMLDense`/`CompileFromXMLFileDense`** — like
  `CompileFromXML`/`CompileFromXMLFile`, but rebuilds every shard's
  `words.dawg` under one dense 1-byte alphabet shared across the whole
  dictionary (OpenCorpora imports can have several shards; pymorphy2's
  never do). Same guarantees as `OpenPyMorphyDense`: identical readings,
  round-trips through `SaveTo`/`Open`. This is what `gomorphy build
  opencorpora` uses by default.
- **`CompileFromUniMorph`/`CompileFromUniMorphFile`** — compiles a
  UniMorph dictionary from a `lemma<TAB>wordform<TAB>bundle` TSV file
  (e.g. `unimorph/rus`). `opts` (`UniMorphOptions`, an alias for
  `unimorph.Options`) requires `Language: "ru"` — currently the only
  accepted value. See
  [implementation/stage-16-import-unimorph.md](implementation/stage-16-import-unimorph.md).
- **`CompileFromUniMorphDense`/`CompileFromUniMorphFileDense`** — like
  `CompileFromUniMorph`/`CompileFromUniMorphFile`, but rebuilds every
  shard's `words.dawg` under one dense 1-byte alphabet shared across
  the whole dictionary. Same guarantees as `CompileFromXMLDense`:
  identical readings, round-trips through `SaveTo`/`Open`. This is
  what `gomorphy build unimorph` uses by default.

```go
d, err := morphology.Open(".data/opencorpora/opencorpora.dat")
if err != nil {
    log.Fatal(err)
}
defer d.Close()
```

`Dictionary.Close()` is a no-op for dictionaries not opened via `Open`
(`OpenPyMorphy*`/`CompileFromXML*` don't use mmap), so it's safe to call
unconditionally.

### Fetching source data

For the CLI utility (`gomorphy download`/`unpack`/`update`, see
[cli.md](cli.md)), fetching sources is done by `pkg/opencorpora.Loader`,
`pkg/pymorphy.Loader`, and `pkg/unimorph.Loader` — the same API is also
available from the library:

```go
loader := pymorphy.NewLoader("") // "" — default path, .data/pymorphy
if err := loader.Sync(false); err != nil { // false — don't skip downloading
    log.Fatal(err)
}
d, err := morphology.OpenPyMorphy(loader.UnpackedDirPath())
```

`opencorpora.Loader` — the same for `dict.xml`
(`loader.UnpackedFilePath()` instead of `UnpackedDirPath()`, then
`morphology.CompileFromXMLFile`).

`unimorph.Loader` — the same for a UniMorph TSV, but `NewLoader`
returns an error (unsupported `language` is rejected up front rather
than surfacing later from `Sync`):

```go
loader, err := unimorph.NewLoader("ru", "") // "" — default path, .data/unimorph/ru
if err != nil {
    log.Fatal(err)
}
if err := loader.Sync(false); err != nil {
    log.Fatal(err)
}
d, err := morphology.CompileFromUniMorphFile(loader.UnpackedFilePath(), morphology.UniMorphOptions{Language: "ru"})
```

## Exact wordform lookup

`Parse` returns all grammatical readings of a word, sorted by probability
(descending), or `nil` if the word is not found either exactly or via
ending-based prediction:

```go
readings := d.Parse("кота")
for _, r := range readings {
    fmt.Printf("%s -> %s (%s)\n", r.Word, r.Normal, r.Tag)
}
// кота -> кот (NOUN,anim,masc,sing,gent)
// кота -> кот (NOUN,anim,masc,sing,accs)
```

The `Reading` type:

```go
type Reading struct {
    Word   string  // wordform as stored in the dictionary (with "ё")
    Normal string  // lemma (base form)
    Tag    string  // grammeme tag, e.g. "NOUN,anim,masc,sing,nomn"
    Para   uint16  // paradigm id — unique only together with Shard
    Form   uint16  // form index within the paradigm
    Shard  int     // dictionary shard index; always 0 for unsharded dictionaries
    Dict   int     // dictionary index within MultiDictionary; always 0 for a direct Dictionary.Parse call
    Prob   float64 // reading probability (0 if probability data is unavailable)
}
```

The `Tag` format depends on the dictionary source: `TagSet.Name`
distinguishes `"opencorpora"` (comma-joined, from `dict.xml`) and
`"opencorpora-int"` (pymorphy2, its own syntax) — both describe the same
set of grammemes but serialize them differently. For comparing tags
between dictionaries of different origin, see
[implementation/tag-mapping.md](implementation/tag-mapping.md)
(`pkg/morphology/tagmap`).

Input is automatically lowercased.

## Lemmas (base forms)

`Lemma` returns the lemma (base form) and its own tag for each homonym of
a word:

```go
lemmas := d.Lemma("кота")
for _, l := range lemmas {
    fmt.Printf("%s (%s)\n", l.Normal, l.Tag)
}
// кот (NOUN,anim,masc,sing,nomn)
```

The `LemmaRef` type:

```go
type LemmaRef struct {
    Normal string // base form
    Tag    string // base form's tag (form 0 of the paradigm)
    Para   uint16
    Shard  int
    Dict   int
}
```

## Fuzzy search

`Fuzzy` returns all dictionary words within Levenshtein distance
`maxDist` (measured in runes; "е"/"ё" count as a single substitution).
`FuzzyTop` returns the `maxWords` nearest words, expanding the distance
iteratively. Both return a `[]FuzzyMatch` sorted by (distance, word), with
no duplicate words:

```go
matches := d.Fuzzy("кот", 1)
top := d.FuzzyTop("кот", 5)
```

```go
type FuzzyMatch struct {
    Word     string
    Distance int
    Dict     int
}
```

Works the same for dictionaries with a dense alphabet
(`OpenPyMorphyDense`) — same matches as the raw dictionary, see "Opening
a dictionary" above.

## Diagnostic metadata

`Info()` returns the file's `info` section (when and with what the
dictionary was built), or `nil` if it's absent (dictionaries not saved via
`SaveTo`, or built before the section was introduced):

```go
type BuildInfo struct {
    BuiltAt        time.Time
    LibraryVersion string
    Source         string // "pymorphy2" / "opencorpora"
    SourceVersion  string
    Author         string
    Description    string
    SourceURL      string
}
```

## Multiple dictionaries at once: `MultiDictionary`

`MultiDictionary` aggregates `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`/`Close`
across an arbitrary set of already-open dictionaries — it does not open
anything itself. Full design:
[implementation/multi-dict.md](implementation/multi-dict.md).

```go
oc, _ := morphology.Open("opencorpora.dat")
pm, _ := morphology.OpenPyMorphy(".data/pymorphy/data")
m := morphology.NewMultiDictionary(oc, pm)
defer m.Close()

for _, r := range m.Parse("кота") {
    fmt.Printf("dict#%d: %s -> %s (%s)\n", r.Dict, r.Word, r.Normal, r.Tag)
}
```

- `Reading.Dict`/`LemmaRef.Dict`/`FuzzyMatch.Dict` — the dictionary's index
  in the order passed to `NewMultiDictionary` (0-based).
- `Parse`/`Lemma`/`Fuzzy` — the concatenation of results from every
  dictionary where the word was found, in registration order, with no
  cross-dictionary prioritization or deduplication (each dictionary
  already deduplicates itself).
- `FuzzyTop(word, maxWords)` — the only method where `maxWords` limits the
  **overall** result rather than the per-dictionary result: it takes the
  top `maxWords` from each dictionary, then sorts and truncates the
  combined set again.
- `DictInfo(i)` — the `BuildInfo` of the dictionary at index `i` (`nil` for
  an out-of-range index or a dictionary with no `info` section).
- `Close()` closes every dictionary in the set, joining errors via
  `errors.Join`.

## Concurrent use

Reading from an open `*Dictionary` is thread-safe — multiple goroutines
can call `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop` concurrently:

```go
var wg sync.WaitGroup
for _, word := range words {
    wg.Add(1)
    go func(w string) {
        defer wg.Done()
        readings := d.Parse(w)
        // process readings
    }(word)
}
wg.Wait()
```

## Saving to disk

`SaveTo` saves a dictionary (including one compiled from XML, loaded via
`OpenPyMorphy`, or rebuilt via `OpenPyMorphyDense`) into the unified
format:

```go
d, err := morphology.CompileFromXMLFile("dict.xml", nil)
if err != nil {
    log.Fatal(err)
}
if err := d.SaveTo("opencorpora.dat"); err != nil {
    log.Fatal(err)
}
```

A dictionary with a dense alphabet (`OpenPyMorphyDense`) round-trips
through `SaveTo`/`Open` like any other — the alphabet codec is written
into its own `.dat` section and reconstructed on `Open`.

## Errors

The package does not define its own exported errors (`Err...`) — every
function returns wrapped (`fmt.Errorf("...: %w", err)`) errors from the
underlying layer (filesystem, format parsing, etc.). Check `err != nil`
and, if you need to distinguish a specific cause, unwrap via
`errors.Is`/`errors.As` against known standard-library errors (e.g.
`os.ErrNotExist`).
