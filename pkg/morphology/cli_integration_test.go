package morphology_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
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
	require.NoError(t, err, ".dat file was created")

	// gomorphy build pymorphy defaults to a dense 1-byte alphabet, no
	// opt-out flag (see docs/en/implementation/pymorphy2-dense-alphabet.md,
	// "Agreed design decisions", item 2) — confirm the CLI-built file
	// actually carries the "alphabet" section, not just that it opens.
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	cont, err := internal.OpenContainer(raw)
	require.NoError(t, err)
	_, _, err = cont.Section("alphabet")
	require.NoError(t, err, "gomorphy build pymorphy must produce a dense (alphabet-section) .dat by default")

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

// TestCLIEndToEnd_OpenCorpora mirrors TestCLIEndToEnd for "build
// opencorpora": confirms it defaults to a dense alphabet too (same policy
// as "pymorphy", see internal.RecompileDense), not just that lookup
// still works.
func TestCLIEndToEnd_OpenCorpora(t *testing.T) {
	xmlPath := filepath.Join(t.TempDir(), "dict.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(exampleDictXML), 0o644))
	bin := buildCLI(t)
	out := filepath.Join(t.TempDir(), "opencorpora.dat")

	build, err := exec.Command(bin, "build", "opencorpora", "-i", xmlPath, "-o", out).CombinedOutput()
	require.NoError(t, err, "build: %s", build)
	require.Contains(t, string(build), "saved")

	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	cont, err := internal.OpenContainer(raw)
	require.NoError(t, err)
	_, _, err = cont.Section("alphabet")
	require.NoError(t, err, "gomorphy build opencorpora must produce a dense (alphabet-section) .dat by default")

	lookup, err := exec.Command(bin, "lookup", "-d", out, "кота").CombinedOutput()
	require.NoError(t, err, "lookup: %s", lookup)
	require.Contains(t, string(lookup), "кот")
	require.Contains(t, string(lookup), "NOUN,anim,masc,sing,gent")
}

// TestCLIEndToEnd_UniMorph mirrors TestCLIEndToEnd/TestCLIEndToEnd_OpenCorpora
// for "build unimorph": dense by default, plus the --lang flag and the
// "unsupported language" error path.
func TestCLIEndToEnd_UniMorph(t *testing.T) {
	tsvPath := filepath.Join(t.TempDir(), "rus.tsv")
	require.NoError(t, os.WriteFile(tsvPath, []byte("кот\tкот\tN;NOM;SG\nкот\tкота\tN;ACC;SG\n"), 0o644))
	bin := buildCLI(t)
	out := filepath.Join(t.TempDir(), "unimorph.dat")

	build, err := exec.Command(bin, "build", "unimorph", "-i", tsvPath, "-o", out, "--lang", "ru").CombinedOutput()
	require.NoError(t, err, "build: %s", build)
	require.Contains(t, string(build), "saved")

	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	cont, err := internal.OpenContainer(raw)
	require.NoError(t, err)
	_, _, err = cont.Section("alphabet")
	require.NoError(t, err, "gomorphy build unimorph must produce a dense (alphabet-section) .dat by default")

	lookup, err := exec.Command(bin, "lookup", "-d", out, "кота").CombinedOutput()
	require.NoError(t, err, "lookup: %s", lookup)
	require.Contains(t, string(lookup), "кот")
	require.Contains(t, string(lookup), "N;ACC;SG")

	badLang, err := exec.Command(bin, "build", "unimorph", "-i", tsvPath, "-o", out, "--lang", "en").CombinedOutput()
	require.Error(t, err, "build unimorph --lang en must fail: %s", badLang)
}
