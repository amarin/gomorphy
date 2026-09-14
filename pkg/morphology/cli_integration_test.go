package morphology_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/require"
)

// buildFixtureDir собирает директорию pymorphy2 в t.TempDir() (как buildFixture,
// но возвращает путь). Переиспользует помощники фикстур из fixture_test.go.
func buildFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeParadigms(t, dir, [][]uint16{
		{0, 1, 0, 1, 0, 0}, // п.0: кот NOUN — ""(nomn) / "а"(gent)
		{0, 2, 0},          // п.1: кот VERB — ""(VERB)
		{0, 1, 0, 1, 0, 0}, // п.2: мышь NOUN
	})
	writeFile(t, dir, "suffixes.json", []byte(`["","а"]`))
	writeFile(t, dir, "paradigm-prefixes.json", []byte(`["","по","наи"]`))
	writeFile(t, dir, "gramtab-opencorpora-int.json", []byte(
		`["NOUN,anim,masc,sing,nomn","NOUN,anim,masc,sing,gent","VERB,impf,trans"]`,
	))

	m := make(map[string]uint32)
	stdWords(m)
	wordsDAWG, guide := testdawg.Build(m)
	writeFile(t, dir, "words.dawg", testdawg.Marshal(wordsDAWG, guide))
	return dir
}

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gomorphy")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/amarin/gomorphy/cmd/gomorphy")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build CLI: %s", out)
	return bin
}

func TestCLIEndToEnd(t *testing.T) {
	dir := buildFixtureDir(t)
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
