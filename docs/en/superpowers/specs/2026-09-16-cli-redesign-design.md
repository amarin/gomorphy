# CLI redesign: unified `gomorphy` binary

## Context

Two separate binaries exist today: `cmd/gomorphy` (interactive console +
`lookup`/`lemmas`/`fuzzy`/`top`/`import`, single `-dict <path>`, stdlib
`flag`) and `cmd/gomorphy_build` (`update`/`compile`, OpenCorpora only,
stdlib `flag`, `-l`/`-d`/`-o`/`-version`). Neither supports multiple
dictionaries, long/short flag pairing, or per-command help; `pkg/pymorphy`
(download+unpack, shipped 2026-09-16) has no CLI at all yet.
`docs/todo.md`'s path-to-1.0.0 lists "CLI grooming" as the last gate before
Этап 18/release, with the note "user has specific ideas, not yet discussed
with Claude" — this spec is that discussion's outcome.

This session shipped two library prerequisites the CLI redesign depends on
directly: `morphology.MultiDictionary` (`pkg/morphology/multidict.go`,
`Parse`/`Lemma`/`Close`/`DictInfo`/`Len`) and, as prep for this spec
specifically, `MultiDictionary.Fuzzy`/`FuzzyTop` (added in the same
session, `FuzzyMatch` gained a `Dict int` field mirroring
`Reading.Dict`/`LemmaRef.Dict`). `BuildInfo.SourceVersion` is now populated
by both importers (also shipped this session) — relevant here because
`DictInfo(i).Source`/`.SourceVersion` is available for any future verbose
dictionary-identification output.

`docs/todo.md`'s Этап 21 ("gomorphy_build CLI редизайн + pymorphy2
source") — previously scoped 2026-09-15 as post-1.0 backlog — is
explicitly superseded by this spec, per the user's decision in this
session's brainstorming: this redesign absorbs and replaces it rather than
being a separate, later increment.

## Decision

**One binary, `cmd/gomorphy`; `cmd/gomorphy_build` is deleted.** Both
existing binaries' logic moves under one command tree. Pre-1.0.0, no
backward compatibility is preserved for either binary's old flag/command
shapes (same precedent as every other breaking change this project has
made pre-release).

**cobra + pflag**, not hand-rolled flag parsing. Gives POSIX long/short
flag pairing, automatic global-and-per-command `-h/--help`, and idiomatic
subcommand structure "for free" — matches the ambition of the flag
contract below (stable short letters, per-command help) without a bespoke
parsing layer to maintain.

**Command surface:**

| Command | Replaces | Purpose |
|---|---|---|
| `cli` | `gomorphy` (no args) | interactive console |
| `lookup <word>` | `gomorphy lookup` | exact dictionary lookup |
| `lemmas <word>` | `gomorphy lemmas` | initial forms |
| `fuzzy <word> [maxDist]` | `gomorphy fuzzy` | fuzzy search |
| `top <word> <N>` | `gomorphy top` | N nearest words |
| `download <type>` | `gomorphy_build update` (partial) | fetch source archive |
| `unpack <type>` | `gomorphy_build update` (partial) | extract source archive |
| `build <type>` | `gomorphy_build compile`, `gomorphy import` | compile source → `.dat` |
| `update <type>` | `gomorphy_build update` (full) | download + unpack + build |
| `merge` | — (new) | **structure only** — registered command, `-h` works, body returns "not yet implemented"; no library logic in this increment |
| `split` | — (new) | same as `merge` |
| `version` | `-version` flag on both binaries | print version, command only, no flag |

`type` for `download`/`unpack`/`build`/`update`: `opencorpora`, `pymorphy`
(matches the sources `pkg/opencorpora`/`pkg/pymorphy` already support;
`unimorph` stays a documented-but-unimplemented future value, consistent
with Этап 16 being unstarted).

**Dictionary resolution is one shared code path for every command that
reads a dictionary (`lookup`/`lemmas`/`fuzzy`/`top`/`cli`), always
producing a `*morphology.MultiDictionary` — never `*morphology.Dictionary`
directly, even for exactly one dictionary.** This was the one point
revised after the initial design pass: the user correctly flagged that
branching CLI code on "one dict vs. many" is pure waste when
`MultiDictionary` already exists — the fix was extending
`MultiDictionary` with `Fuzzy`/`FuzzyTop` (done, see Context) specifically
so no command needs two code paths. Resolution order:

1. Every `-d/--dictionary` value (flag is repeatable) — each value is
   either a file (opened directly) or a directory (glob `*.dat` inside
   it; **not** recursive, **not** raw source directories like an unpacked
   pymorphy2 `data/` — those need `build` first).
2. If no `-d` was given at all: `GOMORPHY_DICTIONARY` env var — exactly
   one path (file or directory, same file-vs-dir handling as above). Not
   a list; combining the env var with multiple `-d` values, or making the
   env var itself a path list, was explicitly decided against for
   simplicity.
3. If the resolved set is still empty: hard error, command does not run.

**Global flags**, all cobra persistent flags on the root command (inherited
by every subcommand):

- `-v/--verbose` — log level `warning`→`info`; also unlocks extra
  diagnostic output at call sites that already have something informative
  to say (e.g. "resolved dictionary path from `$GOMORPHY_DICTIONARY`:
  `<path>`").
- `-l/--log <path>` — redirect the logger's target from `stderr` (default)
  to `<path>`. Always requires a value if present (no bare `-l`). Error if
  `filepath.Dir(path)` does not exist — this check is the CLI's
  responsibility, `amarin/logging`'s own `Target` validation accepts any
  non-empty string as a file path candidate and does not check existence.
- `-h/--help` — cobra-native, works both globally (`gomorphy -h`) and per
  command (`gomorphy lookup -h`).

No `--debug` flag (dropped — `-v` already covers "more logging"; keeping
it would have collided with `-d/--dictionary`'s letter anyway).

**Interactive vs. scripted output.** Detected once via a TTY check on
`os.Stdout` (`golang.org/x/term.IsTerminal` or equivalent — a new,
lightweight dependency, not something to hand-roll). Only the
progress-reporting commands (`download`/`build`/`update`) branch on it: TTY
→ a single self-overwriting line (carriage-return redraw); non-TTY → one
plain line per progress step, matching how a piped/logged run should look
(no control characters in a log file). `lookup`/`lemmas`/`fuzzy`/`top`
output is already line-oriented and unaffected either way.

## Design

### Package layout

```
cmd/gomorphy/
  main.go       — cobra root command, global flag registration, entrypoint
  dict.go       — resolveDictionaries(cmd *cobra.Command) (*morphology.MultiDictionary, error)
  logging.go    — configureLogging(cmd *cobra.Command) error (reads -v/-l, calls logging.Init)
  progress.go   — newProgressReporter(interactive bool) func(processed, total int)
  cli.go        — `cli` command (interactive console, adapted from today's runConsole)
  lookup.go     — `lookup` command
  lemmas.go     — `lemmas` command
  fuzzy.go      — `fuzzy` command
  top.go        — `top` command
  download.go   — `download <type>` command
  unpack.go     — `unpack <type>` command
  build.go      — `build <type>` command
  update.go     — `update <type>` command (calls the same helpers as download+unpack+build)
  merge.go      — `merge` command (stub)
  split.go      — `split` command (stub)
  version.go    — `version` command
cmd/gomorphy_build/  — DELETED
```

One file per command matches cobra's own idiom and keeps each command's
review surface small and independent — consistent with this project's
existing file-per-concern convention (e.g. `pkg/morphology`'s
`parse.go`/`lemma.go`/`fuzzy.go`/`multidict.go` split).

### Root command and global flags (`main.go`)

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	root := &cobra.Command{
		Use:   "gomorphy",
		Short: "gomorphy — морфологический анализ слов по словарям в едином формате (GMOR)",
	}

	root.PersistentFlags().StringArrayP("dictionary", "d", nil, "path to a .dat file or a directory of .dat files (repeatable)")
	root.PersistentFlags().BoolP("verbose", "v", false, "verbose logging and diagnostics")
	root.PersistentFlags().StringP("log", "l", "", "log to this file instead of stderr")

	root.AddCommand(
		newCLICommand(), newLookupCommand(), newLemmasCommand(), newFuzzyCommand(), newTopCommand(),
		newDownloadCommand(), newUnpackCommand(), newBuildCommand(), newUpdateCommand(),
		newMergeCommand(), newSplitCommand(), newVersionCommand(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

Each `new<X>Command()` constructor returns a `*cobra.Command` whose `RunE`
calls `configureLogging(cmd)` first, then (for dictionary-reading commands)
`resolveDictionaries(cmd)`, then does the command's own work. `-i/--input`,
`-o/--output` are declared per-command (not global — only
`download`/`unpack`/`build`/`update` and a future `merge`/`split` use
`-i`/`-o`), keeping the flag contract's letters stable in *meaning*
without forcing every command to expose flags it has no use for. The
source type (`opencorpora`|`pymorphy`) is a required positional argument
(`<type>`) on `download`/`unpack`/`build`/`update`, not a flag — there is
only ever one type per invocation, so a flag would add ceremony without
adding meaning.

### Dictionary resolution (`dict.go`)

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

const dictionaryEnvVar = "GOMORPHY_DICTIONARY"

// resolveDictionaries opens every dictionary named by -d/--dictionary (a
// file, or a directory globbed for *.dat), falling back to
// $GOMORPHY_DICTIONARY (exactly one path, same file-or-directory handling)
// when -d was not given at all. Always returns a *MultiDictionary, even
// for a single resolved path — every dictionary-reading command goes
// through the same code path regardless of count.
func resolveDictionaries(cmd *cobra.Command) (*morphology.MultiDictionary, error) {
	paths, err := cmd.Flags().GetStringArray("dictionary")
	if err != nil {
		return nil, err
	}

	fromEnv := false
	if len(paths) == 0 {
		if envPath := os.Getenv(dictionaryEnvVar); envPath != "" {
			paths = []string{envPath}
			fromEnv = true
		}
	}

	var files []string
	for _, p := range paths {
		expanded, err := expandDictionaryPath(p)
		if err != nil {
			return nil, err
		}
		files = append(files, expanded...)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no dictionary specified: use -d/--dictionary or set $%s", dictionaryEnvVar)
	}

	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose && fromEnv {
		fmt.Fprintf(cmd.ErrOrStderr(), "resolved dictionary from $%s: %s\n", dictionaryEnvVar, paths[0])
	}

	dicts := make([]*morphology.Dictionary, 0, len(files))
	for _, f := range files {
		d, err := morphology.Open(f)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f, err)
		}
		dicts = append(dicts, d)
	}
	return morphology.NewMultiDictionary(dicts...), nil
}

// expandDictionaryPath resolves one -d value: a file path as-is, or a
// directory into every *.dat file directly inside it (non-recursive).
func expandDictionaryPath(p string) ([]string, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("dictionary path %s: %w", p, err)
	}
	if !info.IsDir() {
		return []string{p}, nil
	}
	matches, err := filepath.Glob(filepath.Join(p, "*.dat"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .dat files found in directory %s", p)
	}
	return matches, nil
}
```

### Logging (`logging.go`)

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amarin/logging"
	"github.com/spf13/cobra"
)

func configureLogging(cmd *cobra.Command) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	logPath, _ := cmd.Flags().GetString("log")

	level := logging.LevelWarn
	if verbose {
		level = logging.LevelInfo
	}

	target := logging.StdErr
	if logPath != "" {
		dir := filepath.Dir(logPath)
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("log file directory %s: %w", dir, err)
		}
		target = logging.Target(logPath)
	}

	return logging.Init(logging.WithFormat(logging.FormatText), logging.WithTarget(target), logging.WithLevel(level))
}
```

### Progress reporting (`progress.go`)

```go
package main

import "fmt"

// newProgressReporter returns a progress callback (the shape already used
// by opencorpora.CompileFromXML's progress parameter and import.go's
// shardProgress) shaped for the caller's output mode: interactive
// redraws one line in place; non-interactive prints one line per call, no
// control characters — safe for a log file or a pipe.
func newProgressReporter(interactive bool) func(processed, total int) {
	if !interactive {
		return func(processed, total int) {
			fmt.Printf("%d/%d\n", processed, total)
		}
	}
	return func(processed, total int) {
		fmt.Printf("\r%d/%d", processed, total)
		if processed >= total {
			fmt.Println()
		}
	}
}
```

Interactivity itself is detected once, in `main.go`, via
`term.IsTerminal(int(os.Stdout.Fd()))` (`golang.org/x/term` — a new
dependency, standard for this exact check, not hand-rolled) and threaded
into whichever commands need it (`download`/`build`/`update`).

### `download`/`unpack`/`build`/`update` — source dispatch

Each takes a required positional `<type>` argument (`opencorpora`|`pymorphy`), `-o/--output` (for
`build`/`update`; ignored/rejected for `download`/`unpack`, which write to
the loader's own fixed layout under `.data/<type>/`). `build` additionally
takes an **optional** `-i/--input <path>`: when given, `build` compiles
`<path>` directly (a `dict.xml` file for `opencorpora`, a source directory
for `pymorphy`) instead of going through the loader's
`UnpackedFilePath()`/`UnpackedDirPath()`. This is not a deferred nicety —
it is required by this spec's own testing strategy (see Testing below,
which needs to compile a synthetic fixture directory without a real
network download, exactly the way today's `cli_integration_test.go`'s
`gomorphy import pymorphy2 <fixtureDir> -o out.dat` already does) and by
the same real-world need `import` served before this redesign: compiling
an already-downloaded or hand-built source without re-running
`download`/`unpack` against it first. `download`/`unpack` do not take
`-i` — they always go through the loader, there is nothing else for them
to point at. Dispatch table, one function per `(command, type)` pair built
on already-existing loaders/importers:

| type | download | unpack | build (no `-i`) | build (`-i <path>`) |
|---|---|---|---|---|
| `opencorpora` | `opencorpora.NewLoader("").DownloadUpdate()` | `.UnpackUpdate()` | `morphology.CompileFromXMLFile(loader.UnpackedFilePath(), progress)` → `.SaveTo(out)` | `morphology.CompileFromXMLFile(path, progress)` → `.SaveTo(out)` |
| `pymorphy` | `pymorphy.NewLoader("").DownloadUpdate()` | `.UnpackUpdate()` | `morphology.OpenPyMorphy(loader.UnpackedDirPath())` → `.SaveTo(out)` | `morphology.OpenPyMorphy(path)` → `.SaveTo(out)` |

`build pymorphy` uses `OpenPyMorphy`, not `OpenPyMorphyDense` —
`SaveTo` already hard-errors on a dictionary with a non-nil `Alphabet`
(shipped this session as part of the pymorphy2 dense-recompile work's own
final review fix), so there is no working dense path through `SaveTo` to
offer here yet; this is not a new limitation introduced by this spec.
`update <type>` calls download → unpack → build in sequence, matching
`gomorphy_build update`'s existing full-cycle behavior today, generalized
to both types.

### `merge`/`split` — stubs

```go
func newMergeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "merge",
		Short: "merge several dictionaries into one .dat (not yet implemented)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("merge: not yet implemented")
		},
	}
}
```

`split` mirrors this exactly. Both are registered (appear in `gomorphy -h`
and have their own `gomorphy merge -h`), so the command surface's final
shape is visible now even though the library-level logic is future work.

### `version`

```go
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print gomorphy version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(morphology.Version)
			return nil
		},
	}
}
```

Command only — no `--version` flag anywhere (decided explicitly: one way
to ask, not two).

### `lookup`/`lemmas`/`fuzzy`/`top` — output format

Reuse today's tab-separated shapes, extended with the dictionary index
where a `MultiDictionary`-sourced field now carries one. `lookup`'s
existing format is `Word\tNormal\tTag\tpara#Shard/Para`
(`cmd/gomorphy/main.go:253`, pre-redesign) — becomes
`Word\tNormal\tTag\tpara#Dict/Shard/Para`, reading `Reading.Dict` the same
way `Shard`/`Para` are already read off the struct. `fuzzy`/`top` gain the
same `Dict` component on their `Distance\tWord` line. Exact column layout
is a plan-level detail (this spec fixes that a `Dict` component belongs in
the existing composite field, not a new column, matching the project's
own established style for `Shard`).

### `cli` — interactive console

Same `chzyer/readline`-based loop as today, but the resolved
`*morphology.MultiDictionary` (from `resolveDictionaries`, called once
before entering the read loop) replaces the single `*morphology.Dictionary`
`runConsole` currently closes over. Command dispatch inside the console
(`lookup`/`lemmas`/`fuzzy`/`top` typed at the `gomorphy>` prompt) calls the
same shared per-command logic the top-level `lookup`/`lemmas`/`fuzzy`/`top`
commands use, not a duplicate implementation.

## Non-goals

- `merge`/`split` library logic — command structure only, per the
  brainstorming decision. A real merge needs to reconcile `Paradigms`,
  `Suffixes`, `Prefixes`, `TagSet`, and the DAWGs themselves across
  dictionaries — a separate future design, not scoped here.
- `unimorph` as a real `-t` value — Этап 16 (UniMorph import) is
  unstarted; `download`/`unpack`/`build -t unimorph` are not required to
  work, only to not be silently confused with a typo (an unrecognized
  `-t` value is a hard error either way).
- `--mute`/`-m` (silent mode) — explicitly deferred by the user to a
  separate future item, not part of this redesign.
- Any change to the dense-alphabet pymorphy2 path beyond what already
  shipped (`build pymorphy` stays on `OpenPyMorphy`, not
  `OpenPyMorphyDense` — see Design above).
- `.dat` format changes, `Prediction`/`Probability` DAWG handling,
  `fuzzy.go`'s dense-alphabet limitation — all separately tracked
  non-goals from the pymorphy2 dense-recompile work, untouched here.

## Testing

- **`resolveDictionaries`** (`cmd/gomorphy/dict_test.go`): file path
  resolves to one dictionary; directory path globs every `*.dat` inside
  (non-recursively — a `.dat` in a nested subdirectory must NOT be
  picked up); multiple `-d` values combine; empty directory (no `.dat`
  files) errors; nonexistent path errors; no `-d` + `$GOMORPHY_DICTIONARY`
  set falls back to it (both the file and the directory case); no `-d`
  and no env var errors; **always returns `*MultiDictionary`** — assert
  this directly even for the single-file/single-dictionary case (the
  point of this design decision is exactly that there is no special case
  to accidentally regress back into).
- **`configureLogging`**: default (no flags) → `LevelWarn`/`StdErr`;
  `-v` → `LevelInfo`; `-l <path>` with an existing parent directory
  succeeds; `-l <path>` with a missing parent directory errors, and does
  not partially reconfigure logging first.
- **`newProgressReporter`**: interactive mode's output contains `\r` and
  ends with a trailing newline only once `processed >= total`;
  non-interactive mode's output never contains `\r`, one line per call.
- **Command wiring** (per-command tests, table-driven where the shape
  repeats — e.g. `download`/`unpack`/`build`/`update` all dispatch on the
  same `-t` values): unknown `-t` value errors clearly; `lookup`/`lemmas`/
  `fuzzy`/`top` produce output whose `Dict` component matches which
  fixture dictionary actually held the looked-up word, using the same
  two-dictionary-fixture pattern `pkg/morphology/multidict_test.go`
  already established (reuse or closely mirror it, not reinvent it).
- **`merge`/`split`**: `gomorphy merge -h`/`gomorphy split -h` succeed and
  print help; running either without `-h` returns the "not yet
  implemented" error, not a panic or a silent no-op.
- **`version`**: prints exactly `morphology.Version`, nothing else, exit
  code 0.
- End-to-end smoke test in the spirit of the existing
  `cmd/gomorphy/cli_integration_test.go` (`TestCLIEndToEnd`): build the
  new single binary, run `update -t pymorphy` (or a fixture-based
  equivalent, avoiding a real network call in the default test run —
  match whatever pattern `cli_integration_test.go` already uses), then
  `lookup` against the result.
- `go test ./... -race`, `go build -tags=integration ./...`,
  `golangci-lint run ./...` green throughout (2 pre-existing unrelated
  issues elsewhere not this work's concern, same as every other spec this
  session).
