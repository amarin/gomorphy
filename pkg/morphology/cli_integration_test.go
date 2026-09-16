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
