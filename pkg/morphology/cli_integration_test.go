package morphology_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	run, err := exec.Command(bin, "import", "pymorphy2", dir, "-o", out).CombinedOutput()
	require.NoError(t, err, "import: %s", run)
	require.Contains(t, string(run), "saved")
	_, err = os.Stat(out)
	require.NoError(t, err, ".dat файл создан")

	lookup, err := exec.Command(bin, "-dict", out, "lookup", "кот").CombinedOutput()
	require.NoError(t, err, "lookup: %s", lookup)
	require.Contains(t, string(lookup), "кот")
	require.Contains(t, string(lookup), "NOUN,anim,masc,sing,nomn")

	lemmas, err := exec.Command(bin, "-dict", out, "lemmas", "кота").CombinedOutput()
	require.NoError(t, err, "lemmas: %s", lemmas)
	require.Contains(t, string(lemmas), "кот")

	fuzzy, err := exec.Command(bin, "-dict", out, "fuzzy", "код", "1").CombinedOutput()
	require.NoError(t, err, "fuzzy: %s", fuzzy)
	require.Contains(t, string(fuzzy), "кот")

	if !strings.Contains(string(lookup), "VERB,impf,trans") {
		t.Fatalf("ожидался омоним VERB в lookup: %q", lookup)
	}
}
