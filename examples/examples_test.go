package examples_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestExamplesOutput runs every example whose main.go ends with an
// "// Output:" block — the ones that need no external data — and compares
// its stdout with that block, the way go test checks ExampleXxx functions.
// Examples reading a downloaded dictionary (open, pymorphy) carry no such
// block and are skipped.
func TestExamplesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs every example")
	}

	dirs, err := filepath.Glob("*/main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, mainGo := range dirs {
		want, ok := expectedOutput(t, mainGo)
		if !ok {
			continue
		}
		name := filepath.Dir(mainGo)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("go", "run", "./examples/"+name)
			cmd.Dir = ".." // the repository root, where ./examples/<name> resolves
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			got, err := cmd.Output()
			if err != nil {
				t.Fatalf("go run ./examples/%s: %v\n%s", name, err, stderr.String())
			}
			if strings.TrimSpace(string(got)) != want {
				t.Errorf("output mismatch\n got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// expectedOutput returns the text of main.go's "// Output:" comment block.
func expectedOutput(t *testing.T, mainGo string) (string, bool) {
	t.Helper()
	src, err := os.ReadFile(mainGo)
	if err != nil {
		t.Fatal(err)
	}
	_, block, ok := strings.Cut(string(src), "// Output:")
	if !ok {
		return "", false
	}
	// Like go test, accept both "// Output: text" and a block below it.
	first, rest, _ := strings.Cut(block, "\n")
	var lines []string
	if first = strings.TrimSpace(first); first != "" {
		lines = append(lines, first)
	}
	for line := range strings.Lines(rest) {
		line = strings.TrimSpace(line)
		text, isComment := strings.CutPrefix(line, "//")
		if !isComment {
			break
		}
		lines = append(lines, strings.TrimPrefix(text, " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), true
}
