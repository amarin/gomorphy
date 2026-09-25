# Installation

## Requirements

- Go 1.25+ (the `go` directive follows the policy "current Go release minus two minor versions" and is raised when a new Go minor version ships)
- OS: Linux, macOS (amd64/arm64); Windows builds compile, but `Open`
  returns an error at runtime — dictionary loading uses mmap directly
  through the `syscall` package (see `docs/code-review-pre-1.0.md`).
  `OpenBytes` works on Windows (no mmap); it is today's workaround for
  loading a dictionary there (e.g. `//go:embed` the `.dat` file and pass
  its bytes) — see `docs/en/todo.md`, "Windows: native mmap".

## Installing the CLI utility

```bash
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
```

The utility will be available in `$GOPATH/bin` (make sure that directory is on your `$PATH`).

`gomorphy` is a single binary: morphological analysis against a compiled
dictionary (`lookup`/`lemmas`/`fuzzy`/`top`, interactive `cli` console), as
well as downloading, unpacking, and compiling source dictionaries
(`download`/`unpack`/`build`/`update`).

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
| `make update` | Build `gomorphy` and run the full cycle: download + unpack + build the OpenCorpora dictionary (`gomorphy update opencorpora`) |
| `make compile` | Build `gomorphy` and compile an already-unpacked `dict.xml` (`gomorphy build opencorpora`) |
| `make test` | Run unit tests with the race detector |
| `make test-integration` | Integration tests (`-tags integration`; some require network access and real OpenCorpora data in `.data/`) |
| `make lint` | golangci-lint |
| `make clean` | Remove build artifacts (`./deploy/`) |

## Downloading the OpenCorpora dictionary

After building (`make build`), download and build the dictionary:

```bash
./deploy/gomorphy update opencorpora
```

Result: the file `.data/opencorpora/opencorpora.dat` (tens of MB, depending
on the `dict.xml` version).

More details on working with the dictionary via the CLI -> [cli.md](cli.md).
