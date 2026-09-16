# Unified gomorphy CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `cmd/gomorphy` + `cmd/gomorphy_build` with one cobra-based `gomorphy` binary: unified `-t/-i/-o/-d` flag letters, `-d/--dictionary` always resolves through `MultiDictionary` (single or many), `$GOMORPHY_DICTIONARY` fallback, `-v/-l` logging, per-command help, and `download`/`unpack`/`build`/`update`/`merge`(stub)/`split`(stub)/`version` alongside the existing `lookup`/`lemmas`/`fuzzy`/`top`/`cli`.

**Architecture:** Tasks 1-5 add new, independently-unit-testable files under `cmd/gomorphy/` (each a `new<X>Command() *cobra.Command` plus, where other code needs to call the same logic, a plain `do<X>`/`run<X>` function the command's `RunE` is a thin wrapper over) — the *existing* `cmd/gomorphy/main.go` and all of `cmd/gomorphy_build/` are untouched and still fully functional through all five tasks, so nothing regresses along the way. Task 6 is the atomic cutover: replace `main.go` with the cobra root wiring every command built in tasks 1-5, delete `cmd/gomorphy_build/`, and update the one black-box test that exercises the old CLI surface.

**Tech Stack:** Go (stdlib `os`/`path/filepath`/`strconv`), `github.com/spf13/cobra` (pulls in `github.com/spf13/pflag` as an indirect dependency — no direct `pflag` import needed anywhere in this plan's code), `golang.org/x/term` (TTY detection), `github.com/stretchr/testify`.

**Spec:** [docs/superpowers/specs/2026-09-16-cli-redesign-design.md](../specs/2026-09-16-cli-redesign-design.md)

## Global Constraints

- `cmd/gomorphy_build` is deleted only in Task 6, once `cmd/gomorphy build`/`update` fully replace it — not before.
- No backward compatibility for either old binary's flags/commands — pre-1.0.0, breaking is fine (same precedent as every other breaking change this project has made).
- Every dictionary-reading command goes through `resolveDictionaries`, which always returns `*morphology.MultiDictionary` — never a bare `*morphology.Dictionary`, not even for one resolved path.
- `-t/--type`, `-i/--input`, `-o/--output` are local flags on the commands that use them, never persistent/global. `-d/--dictionary`, `-v/--verbose`, `-l/--log` are persistent on the root only.
- No `--debug` flag anywhere.
- `version` is a command only — no `--version` flag.
- `merge`/`split` are registered commands with working `-h`, but their `RunE` returns an error saying the feature isn't implemented yet — no dictionary-merging logic in this plan.
- `build pymorphy` uses `morphology.OpenPyMorphy`, never `OpenPyMorphyDense` (`SaveTo` already hard-errors on a dense dictionary — this is not a new limitation to work around here).
- `go test ./... -race` and `go build -tags=integration ./...` must stay green after every task. `golangci-lint run ./...` must report only the 2 known pre-existing issues (`pkg/morphology/shard_test.go:67`, `pkg/morphology/importers/opencorpora/import.go:480`).

---

## Task 1: Global flags, dictionary resolution, logging, progress reporting

**Files:**
- Modify: `go.mod`, `go.sum` (add `github.com/spf13/cobra`, `golang.org/x/term`)
- Create: `cmd/gomorphy/flags.go`, `cmd/gomorphy/dict.go`, `cmd/gomorphy/logging.go`, `cmd/gomorphy/progress.go`
- Create: `cmd/gomorphy/dict_test.go`, `cmd/gomorphy/logging_test.go`, `cmd/gomorphy/progress_test.go`

**Interfaces:**
- Produces: `registerGlobalFlags(root *cobra.Command)`, `resolveDictionaries(cmd *cobra.Command) (*morphology.MultiDictionary, error)`, `configureLogging(cmd *cobra.Command) error`, `newProgressReporter(interactive bool) func(processed, total int)`, `isInteractive() bool`, and the test helper `newTestRootCmd(cmd *cobra.Command) *cobra.Command`. Every later task's command file and test file consumes these — this is the plan's shared foundation.

This is `cmd/gomorphy/main.go`'s existing `func main()` untouched — all four new files stand alone, not yet wired into the binary.

- [ ] **Step 1: Dependencies**

`github.com/spf13/cobra`, `github.com/spf13/pflag` (cobra's indirect dep), and `golang.org/x/term` are already in `go.mod`/`go.sum` (commit `a056110`, done ahead of this task by the controller — the environment's network proxy returns 403 for these packages via the normal `GOPROXY` chain, so adding them needed `GOPROXY=off` against an already-warm local module cache; see that commit's message for the full story). **Do not run `go mod tidy` yet** — with no `.go` file importing these packages so far, `tidy` will immediately strip them back out of `go.mod` as unused (this was hit and confirmed during controller setup). `go mod tidy` becomes safe and correct only after Steps 4-7 add real imports — it is not a step in this task; the next task-level `go mod tidy` opportunity is naturally covered by Step 9's full build/test/lint pass, which does not itself invoke `tidy`, so nothing here strips the requirement in the meantime either. If `go build ./...` ever reports "inconsistent vendoring" while working on this task, run `GOPROXY=off go mod vendor` (the module cache is warm; no network needed) and retry.

- [ ] **Step 2: Write the failing tests**

Create `cmd/gomorphy/dict_test.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// fixtureXML returns a minimal valid OpenCorpora dict.xml with exactly one
// NOUN lemma, word. Real, schema-correct XML (same shape already proven
// against the real xmlscan parser by
// pkg/morphology/importers/opencorpora/import_test.go's testDictXML) -
// this package cannot reach pkg/morphology/internal or that test's own
// fixture, so it builds its own via the public morphology API only.
func fixtureXML(word string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">сущ</grammeme>
  <grammeme id="nomn">им.</grammeme>
  <grammeme id="anim">од.</grammeme>
  <grammeme id="masc">м.</grammeme>
  <grammeme id="sing">ед.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text=%q>
   <l t=%q><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t=%q><g v="nomn"/></f>
  </lemma>
 </lemmata>
</dictionary>`, word, word, word)
}

// buildFixtureDat compiles xml into a real .dat file in t.TempDir(),
// through the public morphology API only.
func buildFixtureDat(t *testing.T, xml string) string {
	t.Helper()
	d, err := morphology.CompileFromXML(strings.NewReader(xml), nil)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "fixture.dat")
	require.NoError(t, d.SaveTo(path))
	return path
}

// newTestRootCmd builds a throwaway root with the standard global flags
// registered (mirroring production's real root in main.go, once Task 6
// wires it) and cmd attached as its only child - the shape every command
// needs to run the way it really will. Shared by every later task's tests.
func newTestRootCmd(cmd *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "gomorphy"}
	registerGlobalFlags(root)
	root.AddCommand(cmd)
	return root
}

func TestResolveDictionaries_SingleFile(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", path})
	require.NoError(t, root.Execute())

	require.NotNil(t, got)
	assert.Equal(t, 1, got.Len(), "resolveDictionaries always returns a MultiDictionary, even for one path")
	assert.NotEmpty(t, got.Parse("кот"))
}

func TestResolveDictionaries_MultipleFlags(t *testing.T) {
	pathA := buildFixtureDat(t, fixtureXML("кот"))
	pathB := buildFixtureDat(t, fixtureXML("яблоко"))

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", pathA, "-d", pathB})
	require.NoError(t, root.Execute())

	require.Equal(t, 2, got.Len())
	assert.NotEmpty(t, got.Parse("кот"))
	assert.NotEmpty(t, got.Parse("яблоко"))
}

func TestResolveDictionaries_Directory(t *testing.T) {
	dir := t.TempDir()
	d, err := morphology.CompileFromXML(strings.NewReader(fixtureXML("кот")), nil)
	require.NoError(t, err)
	require.NoError(t, d.SaveTo(filepath.Join(dir, "a.dat")))
	d2, err := morphology.CompileFromXML(strings.NewReader(fixtureXML("яблоко")), nil)
	require.NoError(t, err)
	require.NoError(t, d2.SaveTo(filepath.Join(dir, "b.dat")))
	// Non-.dat file in the same directory must be ignored, not error.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o644))

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", dir})
	require.NoError(t, root.Execute())

	assert.Equal(t, 2, got.Len())
}

func TestResolveDictionaries_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		_, err := resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", dir})
	assert.Error(t, root.Execute())
}

func TestResolveDictionaries_NoFlagNoEnv(t *testing.T) {
	t.Setenv(dictionaryEnvVar, "")

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		_, err := resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe"})
	assert.Error(t, root.Execute())
}

func TestResolveDictionaries_EnvFallback(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	t.Setenv(dictionaryEnvVar, path)

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe"})
	require.NoError(t, root.Execute())

	assert.Equal(t, 1, got.Len())
	assert.NotEmpty(t, got.Parse("кот"))
}

func TestResolveDictionaries_FlagTakesPriorityOverEnv(t *testing.T) {
	envPath := buildFixtureDat(t, fixtureXML("груша"))
	flagPath := buildFixtureDat(t, fixtureXML("кот"))
	t.Setenv(dictionaryEnvVar, envPath)

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", flagPath})
	require.NoError(t, root.Execute())

	assert.Equal(t, 1, got.Len())
	assert.NotEmpty(t, got.Parse("кот"))
	assert.Empty(t, got.Parse("груша"), "env path must not be used when -d was given")
}
```

Create `cmd/gomorphy/logging_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureLogging_DefaultsToStderrWarn(t *testing.T) {
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe"})
	assert.NoError(t, root.Execute())
}

func TestConfigureLogging_Verbose(t *testing.T) {
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-v"})
	assert.NoError(t, root.Execute())
}

func TestConfigureLogging_LogFileExistingDir(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "gomorphy.log")

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-l", logPath})
	require.NoError(t, root.Execute())
}

func TestConfigureLogging_LogFileMissingParentDir(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "does-not-exist", "gomorphy.log")

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-l", logPath})
	assert.Error(t, root.Execute())

	_, statErr := os.Stat(logPath)
	assert.True(t, os.IsNotExist(statErr), "must not create the log file when its parent dir is missing")
}
```

Create `cmd/gomorphy/progress_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProgressReporter_NonInteractive_NoCarriageReturn(t *testing.T) {
	var buf strings.Builder
	report := newProgressReporterTo(&buf, false)
	report(1, 10)
	report(10, 10)

	out := buf.String()
	assert.NotContains(t, out, "\r")
	assert.Equal(t, 2, strings.Count(out, "\n"))
}

func TestNewProgressReporter_Interactive_UsesCarriageReturn(t *testing.T) {
	var buf strings.Builder
	report := newProgressReporterTo(&buf, true)
	report(1, 10)
	report(10, 10)

	out := buf.String()
	assert.Contains(t, out, "\r")
	assert.Equal(t, 1, strings.Count(out, "\n"), "interactive mode prints exactly one trailing newline, once done")
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./cmd/gomorphy/... -v`
Expected: FAIL to compile — none of `registerGlobalFlags`, `resolveDictionaries`, `dictionaryEnvVar`, `configureLogging`, `newProgressReporterTo` exist yet.

- [ ] **Step 4: Implement `flags.go`**

```go
package main

import "github.com/spf13/cobra"

// registerGlobalFlags attaches the flags every dictionary-aware and
// logging-aware command shares (-d/--dictionary, -v/--verbose, -l/--log)
// as persistent flags on root, so every subcommand inherits them without
// redeclaring anything.
func registerGlobalFlags(root *cobra.Command) {
	root.PersistentFlags().StringArrayP("dictionary", "d", nil, "path to a .dat file or a directory of .dat files (repeatable)")
	root.PersistentFlags().BoolP("verbose", "v", false, "verbose logging and diagnostics")
	root.PersistentFlags().StringP("log", "l", "", "log to this file instead of stderr")
}
```

- [ ] **Step 5: Implement `dict.go`**

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
// for a single resolved path - every dictionary-reading command goes
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

- [ ] **Step 6: Implement `logging.go`**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amarin/logging"
	"github.com/spf13/cobra"
)

// configureLogging reads -v/--verbose and -l/--log and initializes the
// process-wide logger. Must be called at the top of every command's RunE,
// before any other work.
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

- [ ] **Step 7: Implement `progress.go`**

```go
package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// isInteractive reports whether stdout is a terminal - used to choose
// between newProgressReporter's redraw-in-place and one-line-per-step
// output modes.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// newProgressReporter returns a progress callback (the shape already used
// by opencorpora.CompileFromXML's progress parameter) shaped for the
// caller's output mode: interactive redraws one line in place;
// non-interactive prints one line per call, no control characters - safe
// for a log file or a pipe.
func newProgressReporter(interactive bool) func(processed, total int) {
	return newProgressReporterTo(os.Stdout, interactive)
}

// newProgressReporterTo is newProgressReporter with an explicit writer,
// for testing without touching the real os.Stdout.
func newProgressReporterTo(w io.Writer, interactive bool) func(processed, total int) {
	if !interactive {
		return func(processed, total int) {
			fmt.Fprintf(w, "%d/%d\n", processed, total)
		}
	}
	return func(processed, total int) {
		fmt.Fprintf(w, "\r%d/%d", processed, total)
		if total > 0 && processed >= total {
			fmt.Fprintln(w)
		}
	}
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./cmd/gomorphy/... -v`
Expected: PASS (all tests in `dict_test.go`, `logging_test.go`, `progress_test.go`).

- [ ] **Step 9: Full test + integration build + lint**

Run: `go test ./... -race`
Run: `go build -tags=integration ./...`
Run: `golangci-lint run ./...`
Expected: all green; lint shows only the 2 known pre-existing issues.

- [ ] **Step 10: Commit**

```bash
git add go.mod go.sum cmd/gomorphy/flags.go cmd/gomorphy/dict.go cmd/gomorphy/logging.go cmd/gomorphy/progress.go cmd/gomorphy/dict_test.go cmd/gomorphy/logging_test.go cmd/gomorphy/progress_test.go
git commit -m "feat(cmd/gomorphy): add cobra global flags, dictionary resolution, logging, progress reporting

New files only - cmd/gomorphy/main.go and cmd/gomorphy_build are
untouched and still fully functional. resolveDictionaries always returns
*MultiDictionary, even for a single resolved dictionary."
```

---

## Task 2: `lookup`/`lemmas`/`fuzzy`/`top` commands

**Files:**
- Create: `cmd/gomorphy/lookup.go`, `cmd/gomorphy/lemmas.go`, `cmd/gomorphy/fuzzy.go`, `cmd/gomorphy/top.go`
- Create: `cmd/gomorphy/lookup_test.go`, `cmd/gomorphy/lemmas_test.go`, `cmd/gomorphy/fuzzy_test.go`, `cmd/gomorphy/top_test.go`

**Interfaces:**
- Consumes: `resolveDictionaries`, `configureLogging`, `newTestRootCmd`, `fixtureXML`, `buildFixtureDat` (Task 1).
- Produces: `doLookup(w io.Writer, m *morphology.MultiDictionary, word string) error`, `doLemmas(w io.Writer, m *morphology.MultiDictionary, word string) error`, `doFuzzy(w io.Writer, m *morphology.MultiDictionary, word string, maxDist int) error`, `doTop(w io.Writer, m *morphology.MultiDictionary, word string, n int) error`, plus `newLookupCommand`/`newLemmasCommand`/`newFuzzyCommand`/`newTopCommand() *cobra.Command`. Task 5's `cli.go` consumes the four `do<X>` functions directly (not the cobra commands) to avoid a duplicate implementation in the interactive console.

- [ ] **Step 1: Write the failing tests**

Create `cmd/gomorphy/lookup_test.go`:

```go
package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoLookup_Found(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doLookup(&buf, m, "кот"))
	out := buf.String()
	assert.Contains(t, out, "кот")
	assert.Contains(t, out, "NOUN")
	assert.Contains(t, out, "para#0/") // Dict=0, always-MultiDictionary
}

func TestDoLookup_NotFound(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	assert.Error(t, doLookup(&buf, m, "несуществующееслово"))
}

func TestLookupCommand_EndToEnd(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newLookupCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"lookup", "-d", path, "кот"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "кот")
}
```

Add the shared `mustResolveOne` test helper to `cmd/gomorphy/dict_test.go` (append, do not create a new file — it belongs next to `resolveDictionaries`'s other test helpers):

```go
// mustResolveOne wraps a single fixture .dat path into a *MultiDictionary
// via the real resolveDictionaries code path (not a direct
// NewMultiDictionary call) - every command test in this package needs
// exactly this, and going through resolveDictionaries is what proves the
// "always MultiDictionary, even for one path" contract at every call site
// that uses this helper.
func mustResolveOne(t *testing.T, path string) *morphology.MultiDictionary {
	t.Helper()
	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", path})
	require.NoError(t, root.Execute())
	return got
}
```

Create `cmd/gomorphy/lemmas_test.go`:

```go
package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoLemmas_Found(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doLemmas(&buf, m, "кот"))
	assert.Contains(t, buf.String(), "кот")
}

func TestDoLemmas_NotFound(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	assert.Error(t, doLemmas(&buf, m, "несуществующееслово"))
}
```

Create `cmd/gomorphy/fuzzy_test.go`:

```go
package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoFuzzy_ExactMatch(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doFuzzy(&buf, m, "кот", 0))
	out := buf.String()
	assert.Contains(t, out, "кот")
	assert.Contains(t, out, "dict#0")
}

func TestFuzzyCommand_DefaultMaxDist(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newFuzzyCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"fuzzy", "-d", path, "кот"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "кот")
}

func TestFuzzyCommand_InvalidMaxDist(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newFuzzyCommand())
	root.SetArgs([]string{"fuzzy", "-d", path, "кот", "not-a-number"})
	assert.Error(t, root.Execute())
}
```

Create `cmd/gomorphy/top_test.go`:

```go
package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoTop_ReturnsUpToN(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doTop(&buf, m, "кот", 1))
	assert.Contains(t, buf.String(), "кот")
}

func TestTopCommand_InvalidN(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newTopCommand())
	root.SetArgs([]string{"top", "-d", path, "кот", "0"})
	assert.Error(t, root.Execute())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/gomorphy/... -v`
Expected: FAIL to compile — none of `doLookup`, `doLemmas`, `doFuzzy`, `doTop`, `newLookupCommand`, `newLemmasCommand`, `newFuzzyCommand`, `newTopCommand`, `mustResolveOne` exist yet.

- [ ] **Step 3: Implement `lookup.go`**

```go
package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doLookup writes every reading of word to w, tab-separated: Word, Normal,
// Tag, and a para#Dict/Shard/Para composite (Dict is always 0 for a
// single-dictionary resolution, since resolveDictionaries always produces
// a MultiDictionary).
func doLookup(w io.Writer, m *morphology.MultiDictionary, word string) error {
	readings := m.Parse(word)
	if len(readings) == 0 {
		return fmt.Errorf("lookup %q: no readings", word)
	}
	for _, r := range readings {
		fmt.Fprintf(w, "%s\t%s\t%s\tpara#%d/%d/%d\n", r.Word, r.Normal, r.Tag, r.Dict, r.Shard, r.Para)
	}
	return nil
}

func newLookupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "lookup <word>",
		Short: "exact dictionary lookup (all readings)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return doLookup(cmd.OutOrStdout(), m, args[0])
		},
	}
}
```

- [ ] **Step 4: Implement `lemmas.go`**

```go
package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doLemmas writes every initial form of word to w, tab-separated: Normal,
// Tag. Unlike doLookup, no Dict component - the pre-redesign lemmas
// output never carried a Shard/Para composite either, and this spec did
// not decide to add one now.
func doLemmas(w io.Writer, m *morphology.MultiDictionary, word string) error {
	refs := m.Lemma(word)
	if len(refs) == 0 {
		return fmt.Errorf("lemmas %q: no lemmas", word)
	}
	for _, r := range refs {
		fmt.Fprintf(w, "%s\t%s\n", r.Normal, r.Tag)
	}
	return nil
}

func newLemmasCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "lemmas <word>",
		Short: "find initial forms (lemmas)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return doLemmas(cmd.OutOrStdout(), m, args[0])
		},
	}
}
```

- [ ] **Step 5: Implement `fuzzy.go`**

```go
package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doFuzzy writes every fuzzy match within maxDist of word to w,
// tab-separated: Distance, Word, dict#Dict.
func doFuzzy(w io.Writer, m *morphology.MultiDictionary, word string, maxDist int) error {
	matches := m.Fuzzy(word, maxDist)
	for _, mt := range matches {
		fmt.Fprintf(w, "%d\t%s\tdict#%d\n", mt.Distance, mt.Word, mt.Dict)
	}
	return nil
}

func newFuzzyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "fuzzy <word> [maxDist]",
		Short: "fuzzy search within edit distance (default 2)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			maxDist := 2
			if len(args) == 2 {
				n, err := strconv.Atoi(args[1])
				if err != nil || n < 0 {
					return fmt.Errorf("invalid maxDist %q", args[1])
				}
				maxDist = n
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return doFuzzy(cmd.OutOrStdout(), m, args[0], maxDist)
		},
	}
}
```

- [ ] **Step 6: Implement `top.go`**

```go
package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doTop writes up to n nearest matches of word to w, tab-separated:
// Distance, Word, dict#Dict.
func doTop(w io.Writer, m *morphology.MultiDictionary, word string, n int) error {
	matches := m.FuzzyTop(word, n)
	for _, mt := range matches {
		fmt.Fprintf(w, "%d\t%s\tdict#%d\n", mt.Distance, mt.Word, mt.Dict)
	}
	return nil
}

func newTopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "top <word> <N>",
		Short: "N nearest words by edit distance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			n, err := strconv.Atoi(args[1])
			if err != nil || n <= 0 {
				return fmt.Errorf("invalid N %q", args[1])
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return doTop(cmd.OutOrStdout(), m, args[0], n)
		},
	}
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./cmd/gomorphy/... -v`
Expected: PASS (all tests, including the appended `mustResolveOne` helper's compile).

- [ ] **Step 8: Full test + integration build + lint**

Run: `go test ./... -race`
Run: `go build -tags=integration ./...`
Run: `golangci-lint run ./...`
Expected: all green; lint shows only the 2 known pre-existing issues.

- [ ] **Step 9: Commit**

```bash
git add cmd/gomorphy/lookup.go cmd/gomorphy/lemmas.go cmd/gomorphy/fuzzy.go cmd/gomorphy/top.go cmd/gomorphy/lookup_test.go cmd/gomorphy/lemmas_test.go cmd/gomorphy/fuzzy_test.go cmd/gomorphy/top_test.go cmd/gomorphy/dict_test.go
git commit -m "feat(cmd/gomorphy): add lookup/lemmas/fuzzy/top commands

Each is a thin cobra RunE over a plain do<X>(w, m, ...) function, so
Task 5's interactive console can call the same logic directly instead of
duplicating it. lookup/fuzzy/top output gains a Dict component; lemmas'
output shape is unchanged from before this redesign."
```

---

## Task 3: `download`/`unpack` commands

**Files:**
- Create: `cmd/gomorphy/download.go`, `cmd/gomorphy/unpack.go`
- Create: `cmd/gomorphy/download_test.go`, `cmd/gomorphy/unpack_test.go`

**Interfaces:**
- Consumes: `configureLogging`, `newTestRootCmd` (Task 1).
- Produces: `runDownload(cmd *cobra.Command, typ string) error`, `runUnpack(cmd *cobra.Command, typ string) error`, `newDownloadCommand`/`newUnpackCommand() *cobra.Command`. Task 4's `update.go` consumes `runDownload`/`runUnpack` directly (not the cobra commands) so `update` doesn't duplicate the type-dispatch switch.

- [ ] **Step 1: Write the failing tests**

Create `cmd/gomorphy/download_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newDownloadCommand())
	root.SetArgs([]string{"download", "unimorph"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}

func TestDownloadCommand_RequiresExactlyOneArg(t *testing.T) {
	root := newTestRootCmd(newDownloadCommand())
	root.SetArgs([]string{"download"})
	assert.Error(t, root.Execute())

	root = newTestRootCmd(newDownloadCommand())
	root.SetArgs([]string{"download", "opencorpora", "pymorphy"})
	assert.Error(t, root.Execute())
}
```

Create `cmd/gomorphy/unpack_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnpackCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newUnpackCommand())
	root.SetArgs([]string{"unpack", "unimorph"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}
```

Real network-touching download/unpack behavior is covered by `pkg/opencorpora`'s and `pkg/pymorphy`'s own existing integration tests (`Loader.DownloadUpdate`/`UnpackUpdate`) — this task's commands are thin dispatchers over those already-tested loaders, so no new integration test is added here; only the dispatch/argument-validation logic (which has no network dependency) is unit-tested.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/gomorphy/... -v`
Expected: FAIL to compile — `newDownloadCommand`/`newUnpackCommand` don't exist yet.

- [ ] **Step 3: Implement `download.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
)

// runDownload downloads the source archive for typ ("opencorpora" or
// "pymorphy"), reporting the result on cmd's output writer.
func runDownload(cmd *cobra.Command, typ string) error {
	switch typ {
	case "opencorpora":
		loader := opencorpora.NewLoader("")
		updated, err := loader.DownloadUpdate()
		if err != nil {
			return fmt.Errorf("download opencorpora: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "opencorpora: updated=%v, %s\n", updated, loader.DataPath())
	case "pymorphy":
		loader := pymorphy.NewLoader("")
		updated, err := loader.DownloadUpdate()
		if err != nil {
			return fmt.Errorf("download pymorphy: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "pymorphy: updated=%v, %s\n", updated, loader.DataPath())
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora or pymorphy)", typ)
	}
	return nil
}

func newDownloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "download <type>",
		Short: "download the source archive for a dictionary type (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			return runDownload(cmd, args[0])
		},
	}
}
```

- [ ] **Step 4: Implement `unpack.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
)

// runUnpack extracts the already-downloaded source archive for typ.
func runUnpack(cmd *cobra.Command, typ string) error {
	switch typ {
	case "opencorpora":
		loader := opencorpora.NewLoader("")
		if err := loader.UnpackUpdate(); err != nil {
			return fmt.Errorf("unpack opencorpora: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "opencorpora: unpacked to %s\n", loader.UnpackedFilePath())
	case "pymorphy":
		loader := pymorphy.NewLoader("")
		if err := loader.UnpackUpdate(); err != nil {
			return fmt.Errorf("unpack pymorphy: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "pymorphy: unpacked to %s\n", loader.UnpackedDirPath())
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora or pymorphy)", typ)
	}
	return nil
}

func newUnpackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unpack <type>",
		Short: "extract the downloaded source archive for a dictionary type (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			return runUnpack(cmd, args[0])
		},
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/gomorphy/... -v`
Expected: PASS.

- [ ] **Step 6: Full test + integration build + lint**

Run: `go test ./... -race`
Run: `go build -tags=integration ./...`
Run: `golangci-lint run ./...`
Expected: all green; lint shows only the 2 known pre-existing issues.

- [ ] **Step 7: Commit**

```bash
git add cmd/gomorphy/download.go cmd/gomorphy/unpack.go cmd/gomorphy/download_test.go cmd/gomorphy/unpack_test.go
git commit -m "feat(cmd/gomorphy): add download/unpack commands

Thin type-dispatch wrappers over the already-tested
pkg/opencorpora.Loader/pkg/pymorphy.Loader. runDownload/runUnpack are
plain functions (not just cobra RunE closures) so Task 4's update
command can call them directly without duplicating the dispatch."
```

---

## Task 4: `build`/`update` commands

**Files:**
- Create: `cmd/gomorphy/build.go`, `cmd/gomorphy/update.go`
- Create: `cmd/gomorphy/build_test.go`, `cmd/gomorphy/update_test.go`

**Interfaces:**
- Consumes: `configureLogging`, `newTestRootCmd`, `fixtureXML` (Task 1); `runDownload`, `runUnpack` (Task 3).
- Produces: `runBuild(cmd *cobra.Command, typ, input, output string) error`, `newBuildCommand`/`newUpdateCommand() *cobra.Command`.

- [ ] **Step 1: Write the failing tests**

Create `cmd/gomorphy/build_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func TestBuildCommand_OpenCorpora_WithInput(t *testing.T) {
	xmlPath := filepath.Join(t.TempDir(), "dict.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(fixtureXML("кот")), 0o644))
	outPath := filepath.Join(t.TempDir(), "out.dat")

	root := newTestRootCmd(newBuildCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"build", "opencorpora", "-i", xmlPath, "-o", outPath})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "saved")

	d, err := morphology.Open(outPath)
	require.NoError(t, err)
	defer func() { _ = d.Close() }()
	assert.NotEmpty(t, d.Parse("кот"))
}

func TestBuildCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "unimorph"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}

func TestBuildCommand_MissingInputFile(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "opencorpora", "-i", "/no/such/file.xml", "-o", filepath.Join(t.TempDir(), "out.dat")})
	assert.Error(t, root.Execute())
}
```

Create `cmd/gomorphy/update_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newUpdateCommand())
	root.SetArgs([]string{"update", "unimorph"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}
```

`update`'s real download+unpack+build sequence needs network access (via `runDownload`) to test end-to-end, matching `pkg/opencorpora`/`pkg/pymorphy`'s own convention of leaving that to their existing integration-tagged tests — this task only unit-tests `update`'s argument validation, which fails before `runDownload` is ever called.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/gomorphy/... -v`
Expected: FAIL to compile — `newBuildCommand`/`newUpdateCommand` don't exist yet.

- [ ] **Step 3: Implement `build.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/common"
	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
)

// runBuild compiles typ's source into a .dat file. input, if non-empty,
// is compiled directly (skipping the loader's own unpacked-file/dir
// path); output, if empty, defaults to .data/<type>/<type>.dat.
func runBuild(cmd *cobra.Command, typ, input, output string) error {
	progress := newProgressReporter(isInteractive())

	var d *morphology.Dictionary
	var err error
	var defaultOut string

	switch typ {
	case "opencorpora":
		xmlPath := input
		if xmlPath == "" {
			xmlPath = opencorpora.NewLoader("").UnpackedFilePath()
		}
		f, openErr := os.Open(xmlPath)
		if openErr != nil {
			return fmt.Errorf("build opencorpora: %w", openErr)
		}
		defer func() { _ = f.Close() }()
		d, err = morphology.CompileFromXML(f, progress)
		defaultOut = common.DomainFilePath(opencorpora.DomainName, "opencorpora.dat")
	case "pymorphy":
		dir := input
		if dir == "" {
			dir = pymorphy.NewLoader("").UnpackedDirPath()
		}
		d, err = morphology.OpenPyMorphy(dir)
		defaultOut = common.DomainFilePath(pymorphy.DomainName, "pymorphy.dat")
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora or pymorphy)", typ)
	}
	if err != nil {
		return fmt.Errorf("build %s: %w", typ, err)
	}

	if output == "" {
		output = defaultOut
	}
	if err := d.SaveTo(output); err != nil {
		return fmt.Errorf("save %s: %w", output, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "saved %s\n", output)
	return nil
}

func newBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build <type>",
		Short: "compile a dictionary source into a .dat file (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringP("input", "i", "", "compile this source path directly, skipping the loader")
	cmd.Flags().StringP("output", "o", "", "output .dat path (default: .data/<type>/<type>.dat)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		return runBuild(cmd, args[0], input, output)
	}
	return cmd
}
```

- [ ] **Step 4: Implement `update.go`**

```go
package main

import (
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <type>",
		Short: "download + unpack + build a dictionary in one step (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringP("output", "o", "", "output .dat path (default: .data/<type>/<type>.dat)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		typ := args[0]
		if err := runDownload(cmd, typ); err != nil {
			return err
		}
		if err := runUnpack(cmd, typ); err != nil {
			return err
		}
		output, _ := cmd.Flags().GetString("output")
		return runBuild(cmd, typ, "", output)
	}
	return cmd
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/gomorphy/... -v`
Expected: PASS.

- [ ] **Step 6: Full test + integration build + lint**

Run: `go test ./... -race`
Run: `go build -tags=integration ./...`
Run: `golangci-lint run ./...`
Expected: all green; lint shows only the 2 known pre-existing issues.

- [ ] **Step 7: Commit**

```bash
git add cmd/gomorphy/build.go cmd/gomorphy/update.go cmd/gomorphy/build_test.go cmd/gomorphy/update_test.go
git commit -m "feat(cmd/gomorphy): add build/update commands

build's -i/--input compiles a given source directly, bypassing the
loader - required for testing without a real network download, and for
compiling an already-unpacked or hand-built source in real use. update
composes runDownload+runUnpack+runBuild, unchanged from Task 3's
functions."
```

---

## Task 5: `version`/`merge`/`split` stubs, interactive `cli` command

**Files:**
- Create: `cmd/gomorphy/version.go`, `cmd/gomorphy/merge.go`, `cmd/gomorphy/split.go`, `cmd/gomorphy/cli.go`
- Create: `cmd/gomorphy/version_test.go`, `cmd/gomorphy/merge_test.go`, `cmd/gomorphy/split_test.go`
- Modify: `cmd/gomorphy/dict.go` (add `parseNonNegativeInt`, a small parsing helper `cli.go`'s console dispatch shares with nothing yet in earlier tasks — `fuzzy.go`/`top.go`'s own inline `strconv.Atoi` calls are left as they are, not retrofitted onto this helper, since that's out of this task's scope)

**Interfaces:**
- Consumes: `resolveDictionaries`, `configureLogging` (Task 1); `doLookup`, `doLemmas`, `doFuzzy`, `doTop` (Task 2); `morphology.Version` (existing, `pkg/morphology`).
- Produces: `newVersionCommand`, `newMergeCommand`, `newSplitCommand`, `newCLICommand() *cobra.Command`, `parseNonNegativeInt(s string) (int, error)`.

No dedicated `cli_test.go` — the interactive console is a readline loop over `os.Stdin`, not practically unit-testable in isolation; it is covered by Task 6's end-to-end smoke test instead (documented there, not skipped silently).

- [ ] **Step 1: Write the failing tests**

Create `cmd/gomorphy/version_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func TestVersionCommand(t *testing.T) {
	root := newTestRootCmd(newVersionCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"version"})
	require.NoError(t, root.Execute())
	assert.Equal(t, morphology.Version, strings.TrimSpace(buf.String()))
}
```

Create `cmd/gomorphy/merge_test.go`:

```go
package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeCommand_Help(t *testing.T) {
	root := newTestRootCmd(newMergeCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"merge", "-h"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "merge")
}

func TestMergeCommand_NotImplemented(t *testing.T) {
	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge"})
	err := root.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}
```

Create `cmd/gomorphy/split_test.go`:

```go
package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitCommand_Help(t *testing.T) {
	root := newTestRootCmd(newSplitCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"split", "-h"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "split")
}

func TestSplitCommand_NotImplemented(t *testing.T) {
	root := newTestRootCmd(newSplitCommand())
	root.SetArgs([]string{"split"})
	err := root.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/gomorphy/... -v`
Expected: FAIL to compile — `newVersionCommand`/`newMergeCommand`/`newSplitCommand` don't exist yet.

- [ ] **Step 3: Implement `version.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print gomorphy version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), morphology.Version)
			return nil
		},
	}
}
```

- [ ] **Step 4: Implement `merge.go` and `split.go`**

```go
package main

import (
	"errors"

	"github.com/spf13/cobra"
)

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

Create `cmd/gomorphy/split.go`:

```go
package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newSplitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "split",
		Short: "split one dictionary into several .dat files (not yet implemented)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("split: not yet implemented")
		},
	}
}
```

- [ ] **Step 5: Implement `cli.go`**

```go
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

var consoleCommands = []string{"lookup", "lemmas", "fuzzy", "top", "exit", "quit"}

type commandCompleter struct{}

func (c *commandCompleter) Do(line []rune, pos int) ([][]rune, int) {
	input := string(line[:pos])
	parts := strings.Fields(input)

	completeCmd := len(parts) == 0 || (len(parts) == 1 && !strings.HasSuffix(input, " "))
	if !completeCmd {
		return nil, 0
	}

	prefix := ""
	if len(parts) == 1 {
		prefix = parts[0]
	}

	var result [][]rune
	for _, cmd := range consoleCommands {
		if strings.HasPrefix(cmd, prefix) {
			suffix := cmd[len(prefix):]
			result = append(result, []rune(suffix))
		}
	}
	return result, len(prefix)
}

func newCLICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cli",
		Short: "interactive console",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return runConsole(cmd.OutOrStdout(), m)
		},
	}
}

func runConsole(out io.Writer, m *morphology.MultiDictionary) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "gomorphy> ",
		AutoComplete:    &commandCompleter{},
		HistoryFile:     "",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("init console: %w", err)
	}
	defer func() { _ = rl.Close() }()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			if err == io.EOF {
				fmt.Fprintln(out)
				return nil
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}

		fields := strings.Fields(line)
		cmd, cmdArgs := fields[0], fields[1:]

		var runErr error
		switch cmd {
		case "lookup":
			runErr = requireOneArg(cmdArgs, func(word string) error { return doLookup(out, m, word) })
		case "lemmas":
			runErr = requireOneArg(cmdArgs, func(word string) error { return doLemmas(out, m, word) })
		case "fuzzy":
			runErr = runConsoleFuzzy(out, m, cmdArgs)
		case "top":
			runErr = runConsoleTop(out, m, cmdArgs)
		default:
			runErr = fmt.Errorf("unknown command %q", cmd)
		}
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", runErr)
		}
	}
}

func requireOneArg(args []string, fn func(string) error) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: <word>")
	}
	return fn(args[0])
}

func runConsoleFuzzy(out io.Writer, m *morphology.MultiDictionary, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: fuzzy <word> [maxDist]")
	}
	maxDist := 2
	if len(args) == 2 {
		n, err := parseNonNegativeInt(args[1])
		if err != nil {
			return err
		}
		maxDist = n
	}
	return doFuzzy(out, m, args[0], maxDist)
}

func runConsoleTop(out io.Writer, m *morphology.MultiDictionary, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: top <word> <N>")
	}
	n, err := parseNonNegativeInt(args[1])
	if err != nil || n == 0 {
		return fmt.Errorf("invalid N %q", args[1])
	}
	return doTop(out, m, args[0], n)
}
```

Add the small shared helper `parseNonNegativeInt` to `cmd/gomorphy/dict.go` (it belongs with the other small shared utilities, not duplicated between `fuzzy.go`'s/`top.go`'s cobra `Args`-validation-by-hand and `cli.go`'s console parsing):

```go
// parseNonNegativeInt parses s as a non-negative int, for the CLI flags
// and console commands that take a distance or count.
func parseNonNegativeInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	return n, nil
}
```

(This adds `"strconv"` to `dict.go`'s imports.)

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./cmd/gomorphy/... -v`
Expected: PASS.

- [ ] **Step 7: Full test + integration build + lint**

Run: `go test ./... -race`
Run: `go build -tags=integration ./...`
Run: `golangci-lint run ./...`
Expected: all green; lint shows only the 2 known pre-existing issues.

- [ ] **Step 8: Commit**

```bash
git add cmd/gomorphy/version.go cmd/gomorphy/merge.go cmd/gomorphy/split.go cmd/gomorphy/cli.go cmd/gomorphy/dict.go cmd/gomorphy/version_test.go cmd/gomorphy/merge_test.go cmd/gomorphy/split_test.go
git commit -m "feat(cmd/gomorphy): add version/merge(stub)/split(stub)/cli commands

cli's console dispatch calls doLookup/doLemmas/doFuzzy/doTop directly -
same logic as the top-level commands, not a second implementation."
```

---

## Task 6: Cutover — wire `main.go`, delete `gomorphy_build`, update the black-box test

**Files:**
- Modify: `cmd/gomorphy/main.go` (full rewrite)
- Delete: `cmd/gomorphy_build/` (entire directory)
- Modify: `pkg/morphology/cli_integration_test.go`

**Interfaces:**
- Consumes: every `new<X>Command()` from Tasks 1-5.

This is the one task that touches the actually-running binary. Everything it wires already has its own passing unit tests from Tasks 1-5 — this task's own job is the wiring itself and the one black-box regression test, not re-proving each command's internal logic.

- [ ] **Step 1: Read the current black-box test to know exactly what it asserts today**

Read `pkg/morphology/cli_integration_test.go` in full (it's short) before changing it — it currently drives the *old* CLI (`gomorphy import pymorphy2 <dir> -o <out>`, `gomorphy -dict <out> lookup <word>` etc.) via `exec.Command` on a binary built from `github.com/amarin/gomorphy/cmd/gomorphy`. Every assertion in it needs an equivalent under the new command syntax before this task is done — none should simply disappear.

- [ ] **Step 2: Rewrite `cmd/gomorphy/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "gomorphy",
		Short: "gomorphy — морфологический анализ слов по словарям в едином формате (GMOR)",
	}

	registerGlobalFlags(root)

	root.AddCommand(
		newCLICommand(),
		newLookupCommand(), newLemmasCommand(), newFuzzyCommand(), newTopCommand(),
		newDownloadCommand(), newUnpackCommand(), newBuildCommand(), newUpdateCommand(),
		newMergeCommand(), newSplitCommand(),
		newVersionCommand(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

Delete every other top-level declaration that used to live in the old `main.go` (the old `usage` const, `commands`/`commandCompleter` at package scope, `runImport`/`importAndSave`/`runConsole`/`runLookup`/`runLemmas`/`runFuzzy`/`runTop` and their flag-based argument parsing) — all of it has a Task 1-5 equivalent now. Note: Task 5's `cli.go` had to rename its own versions to `consoleCompleter`/`runInteractiveConsole` (not `commandCompleter`/`runConsole` as originally sketched here) specifically to avoid colliding with these still-present old `main.go` declarations while Tasks 1-5 keep the old CLI working — once this step deletes the old `main.go` declarations, the naming collision that forced the rename no longer exists, but `cli.go` itself is not touched by this step and keeps its renamed identifiers as-is (do not rename them back; nothing in this file's own new content references `commandCompleter`/`runConsole` by name — only `newCLICommand()`, which is unaffected either way).

- [ ] **Step 3: Delete `cmd/gomorphy_build/`**

```bash
git rm -r cmd/gomorphy_build
```

- [ ] **Step 4: Rewrite `pkg/morphology/cli_integration_test.go` for the new command syntax**

```go
package morphology_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gomorphy")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/amarin/gomorphy/cmd/gomorphy")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build CLI: %s", out)
	return bin
}

func TestCLIEndToEnd(t *testing.T) {
	words := make(map[string]uint32)
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)
	bin := buildCLI(t)
	out := filepath.Join(t.TempDir(), "pymorphy2.dat")

	build, err := exec.Command(bin, "build", "pymorphy", "-i", dir, "-o", out).CombinedOutput()
	require.NoError(t, err, "build: %s", build)
	require.Contains(t, string(build), "saved")
	_, err = os.Stat(out)
	require.NoError(t, err, ".dat файл создан")

	// кот has two paradigms in stdWords (NOUN and VERB, a homonym) - both
	// must surface in one lookup, same as before this redesign.
	lookup, err := exec.Command(bin, "lookup", "-d", out, "кот").CombinedOutput()
	require.NoError(t, err, "lookup: %s", lookup)
	require.Contains(t, string(lookup), "кот")
	require.Contains(t, string(lookup), "NOUN,anim,masc,sing,nomn")
	require.Contains(t, string(lookup), "VERB")

	lemmas, err := exec.Command(bin, "lemmas", "-d", out, "кота").CombinedOutput()
	require.NoError(t, err, "lemmas: %s", lemmas)
	require.Contains(t, string(lemmas), "кот")

	fuzzy, err := exec.Command(bin, "fuzzy", "-d", out, "код").CombinedOutput()
	require.NoError(t, err, "fuzzy: %s", fuzzy)
	require.Contains(t, string(fuzzy), "dict#0")
}
```

- [ ] **Step 5: Run tests to verify everything passes**

Run: `go build ./...` and `go build -tags=integration ./...`
Run: `go test ./... -race`
Expected: all green, including the rewritten `TestCLIEndToEnd`.

- [ ] **Step 6: Lint**

Run: `golangci-lint run ./...`
Expected: only the 2 known pre-existing issues (now that `cmd/gomorphy_build` is deleted, confirm neither of those 2 issues was inside it — they are not: both are in `pkg/morphology/shard_test.go` and `pkg/morphology/importers/opencorpora/import.go`).

- [ ] **Step 7: Manual sanity check**

```bash
go build -o /tmp/gomorphy ./cmd/gomorphy
/tmp/gomorphy -h
/tmp/gomorphy lookup -h
/tmp/gomorphy version
/tmp/gomorphy merge
```
Expected: root help lists every command; `lookup -h` shows its own flags; `version` prints the version string; `merge` prints the "not yet implemented" error and exits non-zero.

- [ ] **Step 8: Commit**

```bash
git add cmd/gomorphy/main.go pkg/morphology/cli_integration_test.go
git commit -m "feat(cmd/gomorphy): wire the cobra command tree into main, delete gomorphy_build

Atomic cutover: main.go now builds the root command from every
new<X>Command() added in Tasks 1-5; cmd/gomorphy_build is gone
(superseded by build/update); the black-box CLI test now drives the new
command syntax end to end (build -i, lookup/lemmas/fuzzy with -d)."
```

---

## Final Verification

- [ ] `go test ./... -race` — all green.
- [ ] `go build -tags=integration ./...` — compiles clean.
- [ ] `golangci-lint run ./...` — only the 2 known pre-existing issues.
- [ ] `go vet ./...` — clean.
- [ ] `gomorphy -h`, every `gomorphy <command> -h`, and `gomorphy version` all work when run manually against a real built binary.
