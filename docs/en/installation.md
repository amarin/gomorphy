# Installation

## Requirements

- Go 1.25+ (the `go` directive follows the policy "current Go release minus two minor versions" and is raised when a new Go minor version ships)
- OS: Unix only (Linux, macOS, BSD) — `Open` loads a dictionary via
  `syscall.Mmap` (see [code-review-pre-1.0.md](code-review-pre-1.0.md)).
  Windows builds compile, but `Open` returns an error at runtime, so the
  `gomorphy` CLI can't open dictionaries there either. `OpenBytes` works
  on Windows (no mmap); it is today's workaround for loading a dictionary
  there (e.g. `//go:embed` the `.dat` file and pass its bytes) — see
  [todo.md](todo.md#windows-native-mmap--planned-post-10-backlog),
  "Windows: native mmap".

## Installing the CLI utility

```bash
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
```

The utility will be available in `$GOPATH/bin` (make sure that directory is on your `$PATH`).

`gomorphy` is a single binary: morphological analysis against a compiled
dictionary (`lookup`/`lemmas`/`fuzzy`/`top`, interactive `cli` console), as
well as downloading, unpacking, and compiling source dictionaries
(`download`/`unpack`/`build`/`update`), importing a wordform TSV
(`import tsv`) and merging dictionaries (`merge`).

More details -> [cli.md](cli.md).

## Installing as a Go library

```bash
go get github.com/amarin/gomorphy/pkg/morphology
```

Import in code:

```go
import "github.com/amarin/gomorphy/pkg/morphology"
```

More details on the public API -> [library.md](library.md).

## Building from source

```bash
git clone https://github.com/amarin/gomorphy.git
cd gomorphy
make build     # builds the CLI utility into ./deploy/
```

Available Makefile targets:

| Target | Description |
|------|----------|
| `make build` | Build the CLI utility (`gomorphy`) into `./deploy/` |
| `make update` | Build `gomorphy` and run the full cycle: download + unpack + build the OpenCorpora dictionary (`gomorphy update opencorpora`; opencorpora.org is currently unreachable — see below) |
| `make compile` | Build `gomorphy` and compile an already-unpacked `dict.xml` (`gomorphy build opencorpora`) |
| `make test` | Run unit tests with the race detector |
| `make test-integration` | Integration tests (`-tags integration`; some require network access and real OpenCorpora data in `.data/`) |
| `make lint` | golangci-lint |
| `make clean` | Remove build artifacts (`./deploy/`) |

## Downloading a dictionary

After building (`make build`), download and build the pymorphy2 dictionary
(`pymorphy2-dicts-ru`, from PyPI) — the recommended source:

```bash
./deploy/gomorphy update pymorphy
```

Result: the file `.data/pymorphy/pymorphy.dat` (about 16 MB).

OpenCorpora (`dict.opcorpora.xml.bz2` from opencorpora.org) remains
supported as an alternative, but opencorpora.org is currently unreachable
and its continued availability is not guaranteed:

```bash
./deploy/gomorphy update opencorpora
```

Result: the file `.data/opencorpora/opencorpora.dat` (about 10 MB, depending
on the `dict.xml` version).

More details on working with the dictionary via the CLI -> [cli.md](cli.md).
