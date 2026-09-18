# The pymorphy2 source (`pkg/pymorphy`) and its integration into the unified `gomorphy` CLI

## Original plan (Stage 21, drafted 2026-09-15)

The originally drafted task was to reformat `cmd/gomorphy_build` from a
"one dictionary per command" tool into a universal pipeline
`gomorphy_build <command> <dict_type> [options]`
(`download`/`unpack`/`compile`/`update` x `opencorpora`/`pymorphy`/
`unimorph`) and add a second dictionary data source — pre-built
pymorphy2 dictionaries from PyPI (the `pymorphy2-dicts-ru` package).

**Update 2026-09-16**: this plan was absorbed into a broader CLI
redesign — instead of reformatting `cmd/gomorphy_build` into a
separate pipeline, both binaries (`cmd/gomorphy`, `cmd/gomorphy_build`)
were replaced by one cobra-based `gomorphy`. The literal interface
`gomorphy_build <command> <dict_type>` as a separate binary was not
implemented — but its *substance* (command, then dictionary type, then
options; a single download->unpack->compile pipeline for several
types) is implemented via `gomorphy`'s subcommands. Design:
[2026-09-16-cli-redesign-design.md](../superpowers/specs/2026-09-16-cli-redesign-design.md).
Plan: [2026-09-16-cli-redesign.md](../superpowers/plans/2026-09-16-cli-redesign.md).

## What's implemented

### `pkg/pymorphy` — a loader for the pre-built dictionary (modeled on `pkg/opencorpora`)

- `const.go`: `DomainName = "pymorphy"`, `PyPIPackageName =
  "pymorphy2-dicts-ru"`, `PyPIJSONURL` (JSON: `releases`/`urls`, wheel =
  ZIP, subtree `pymorphy2_dicts_ru/data/`).
- `loader.go`: `Loader` — methods modeled on `opencorpora.Loader`:
  `Sync(skipDownload)` = download+unpack, `DownloadUpdate()` (PyPI JSON
  API -> wheel URL -> `.data/pymorphy/pymorphy2-dicts-ru.whl` +
  `version.txt`), `UnpackUpdate()` (zip -> `.data/pymorphy/data/`).
- `pypi.go` — parsing the PyPI JSON API.

One deliberate departure from the original Stage 21 plan's literal
text: the downloaded file's name is fixed (`pymorphy2-dicts-ru.whl`,
with no version in the name), and the PyPI version is stored
separately in `version.txt` — simpler than parsing the version out of
a filename on every update check.

### `cmd/gomorphy` — integration into the unified CLI

Verified by reading the code (accurate as of writing):
- `gomorphy download pymorphy` (`cmd/gomorphy/download.go`) — downloads
  the wheel via `pymorphy.NewLoader("")`.
- `gomorphy unpack pymorphy` (`cmd/gomorphy/unpack.go`) — unpacks it
  into `.data/pymorphy/data/`.
- `gomorphy build pymorphy [-i <dir>] [-o <path>]`
  (`cmd/gomorphy/build.go`) — compiles via `morphology.OpenPyMorphy`
  + `SaveTo` into a single `.dat` (by default
  `.data/pymorphy/pymorphy.dat`). Compilation is **inherently optional**
  for pymorphy2 (the unpack result already loads directly via
  `morphology.OpenPyMorphy` with no compilation — the DAWG/prediction
  data is already built in the source package), but the `build` command
  unifies it with OpenCorpora into a single `.dat` file (mmap, one file).
- `gomorphy update pymorphy` (`cmd/gomorphy/update.go`) — download +
  unpack + build in one call.

Bottom line: both "`compile pymorphy`" and "CLI redesign" — the two
items the original Stage 21 plan left open — are implemented, just
under different command syntax (`gomorphy <command> <type>` instead of
`gomorphy_build <command> <dict_type>`).

## Automated checks

Covered by existing tests: `pkg/pymorphy/loader_test.go`,
`pkg/pymorphy/pypi_test.go` (PyPI JSON parsing, versions, unpacking a
test zip), `cmd/gomorphy/{download,unpack,build,update}_test.go`.

## Not part of this increment

- `unimorph` as a dictionary type in `download`/`unpack`/`build` — named
  in the original Stage 21 plan as "groundwork for the future," not
  implemented; the UniMorph import itself is Stage 16, separate and not
  started (see [todo.md](../todo.md)).
