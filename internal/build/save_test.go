package build_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/build"
)

// goldenWords is the roundtrip control set: word -> expected lemma base texts.
var goldenWords = map[string][]string{
	"кот":    {"кот"},
	"кота":   {"кот"},
	"коту":   {"кот"},
	"пила":   {"пила", "пилить"},
	"пилы":   {"пила"},
	"пилю":   {"пилить"},
	"стекло": {"стекло"},
	"стекла": {"стекло", "стекать"},
}

func snapshotLemmaTexts(t *testing.T, snap *build.Snapshot, word string) []string {
	t.Helper()

	state, ok := snap.LookupWord([]byte(word))
	if !ok {
		return nil
	}

	set := map[string]bool{}

	for _, p := range snap.PostingsOf(state) {
		for _, l := range snap.PairLemmasOf(p) {
			set[string(snap.LemmaText(l))] = true
		}
	}

	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}

	sortStrings(out)

	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func TestSaveOpenRoundtrip(t *testing.T) {
	snapA := mustBuildSmall(t)

	path := filepath.Join(t.TempDir(), "dict.gmrf")
	if err := snapA.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	snapB, region, err := build.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	defer func() { _ = region.Close() }()

	if !reflect.DeepEqual(snapA, snapB) {
		t.Fatal("roundtrip snapshots differ")
	}

	for _, word := range []string{"кот", "кота", "коту", "пила", "пилы", "пилю", "стекло", "стекла"} {
		if _, ok := snapB.LookupWord([]byte(word)); !ok {
			t.Errorf("word %q not found after roundtrip", word)
		}
	}

	if _, ok := snapB.LookupWord([]byte("несуществует")); ok {
		t.Error("absent word found after roundtrip")
	}
}

func TestGoldenWordSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "golden.gmrf")

	if err := mustBuildSmall(t).SaveTo(path); err != nil {
		t.Fatal(err)
	}

	snap, region, err := build.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = region.Close() }()

	for word, wantLemmas := range goldenWords {
		got := snapshotLemmaTexts(t, snap, word)

		if len(got) != len(wantLemmas) {
			t.Errorf("%q: got lemmas %v, want %v", word, got, wantLemmas)

			continue
		}

		for i := range wantLemmas {
			sortStrings(wantLemmas)

			if got[i] != wantLemmas[i] {
				t.Errorf("%q: got lemmas %v, want %v", word, got, wantLemmas)

				break
			}
		}
	}

	// Grammeme names survive the roundtrip.
	state, ok := snap.LookupWord([]byte("кота"))
	if !ok {
		t.Fatal("кота missing")
	}

	pair := snap.PostingsOf(state)[0]

	var names []string

	for _, g := range snap.Ancode(snap.PairAncodes[pair]) {
		names = append(names, snap.GrammemeNames[g])
	}

	if got := strings.Join(names, ","); got != "NOUN,anim,masc,gent" {
		t.Errorf("кота ancode = %q, want NOUN,anim,masc,gent", got)
	}
}

func TestOpenCorruptTruncated(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.gmrf")

	if err := mustBuildSmall(t).SaveTo(src); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}

	trunc := filepath.Join(t.TempDir(), "trunc.gmrf")
	if err := os.WriteFile(trunc, raw[:len(raw)/2], 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := build.OpenFile(trunc); err == nil {
		t.Fatal("expected error for truncated file")
	} else if !strings.Contains(err.Error(), "checksum") && !strings.Contains(err.Error(), "catalog") && !strings.Contains(err.Error(), "magic") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenCorruptChecksum(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.gmrf")

	if err := mustBuildSmall(t).SaveTo(src); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}

	raw[len(raw)/2] ^= 0xFF

	corrupt := filepath.Join(t.TempDir(), "corrupt.gmrf")
	if err := os.WriteFile(corrupt, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := build.OpenFile(corrupt); err == nil {
		t.Fatal("expected checksum error")
	} else if !strings.Contains(strings.ToLower(err.Error()), "checksum") {
		t.Errorf("expected checksum error, got: %v", err)
	}
}

func TestOpenBadMagic(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.gmrf")
	if err := os.WriteFile(bad, []byte("NOPE-not-a-dictionary-file-at-all"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := build.OpenFile(bad); err == nil || !strings.Contains(err.Error(), "magic") {
		t.Fatalf("expected magic error, got: %v", err)
	}
}
