//go:build integration

// Integration tests here require the real OpenCorpora dict.xml at
// .data/opencorpora/dict.xml (see docs/todo.md / Makefile's
// test-integration target). Run explicitly:
//
//	go test -tags=integration ./pkg/morphology/importers/opencorpora/... -v
package opencorpora

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/xmlscan"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

const (
	realDictEnvVar      = "GOMORPHY_DICT_XML"
	realDictDefaultPath = ".data/opencorpora/dict.xml"
)

func openRealDict(t *testing.T) *os.File {
	t.Helper()

	path := os.Getenv(realDictEnvVar)
	if path == "" {
		path = findRealDictXML()
		if path == "" {
			t.Skipf("dict.xml not found (%s to override)", realDictEnvVar)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open dict %q: %v", path, err)
	}
	return f
}

// findRealDictXML walks up from the working dir looking for
// .data/opencorpora/dict.xml, so this test passes both from the
// package dir and from the repo root (same approach as
// internal/xmlscan/integration_test.go).
func findRealDictXML() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, realDictDefaultPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// TestStripCmp2PrefixRealDictAnomalies reports (does not fail on)
// lemmas where a Cmp2-tagged form does not literally start with "по"
// — the narrow Cmp2-only fix's fallback handles these safely by not
// merging them; this test exists to keep the count visible, not to
// gate on zero.
func TestStripCmp2PrefixRealDictAnomalies(t *testing.T) {
	f := openRealDict(t)
	defer func() { _ = f.Close() }()

	tagSet := internal.NewTagSet("opencorpora")
	var lemmas []lemmaEntry
	handler := &xmlHandler{tagSet: tagSet, lemmas: &lemmas}
	if err := xmlscan.New(f, handler).Scan(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if handler.err != nil {
		t.Fatalf("handler: %v", handler.err)
	}

	anomalies := 0
	var examples []string
	for _, lem := range lemmas {
		if _, _, ok := stripCmp2Prefix(lem.forms); !ok {
			anomalies++
			if len(examples) < 10 {
				examples = append(examples, lem.text)
			}
		}
	}
	if anomalies > 0 {
		t.Logf("real dict.xml has %d lemmas where a Cmp2 form does not literally start with \"по\" "+
			"(examples: %v) — these fall back to the pre-fix (non-merged) behavior via "+
			"stripCmp2Prefix's anomaly path, which is correct and safe, just misses the merge "+
			"benefit for these lemmas. Root cause: the base word already carries its own prefix "+
			"(недо-/около-/мульти-/etc.) and OpenCorpora infixes \"по\" after it rather than "+
			"prepending it to the whole word (e.g. \"недобитее\" -> Cmp2 form \"недопобитее\", not "+
			"\"понедобитее\") — out of scope for the narrow Cmp2-only fix by design, see "+
			"docs/superpowers/specs/2026-09-15-comparative-prefix-split-design.md.",
			anomalies, examples)
	}
}

// TestImportFromXMLRealDictComparativeWordsResolve spot-checks known
// comparative-degree words from
// docs/research/0003-comparative-paradigms-not-merging.md against the
// real compiled dictionary: each word must be found, and at least one
// of its DAWG readings must resolve to the expected normal form. Real
// dict.xml has legitimate cross-lemma homonyms (a surface string can
// be a form of more than one unrelated lemma), so — unlike the tiny
// plan fixture used elsewhere — we check "at least one reading
// matches" rather than requiring every reading to match, mirroring
// the normalFormsForWord + assert.Contains pattern already used in
// Task 2's import_test.go.
func TestImportFromXMLRealDictComparativeWordsResolve(t *testing.T) {
	f := openRealDict(t)
	defer func() { _ = f.Close() }()

	d, err := CompileFromXML(f, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	strAt := func(list []string, id uint16) string {
		if int(id) < len(list) {
			return list[id]
		}
		return ""
	}

	cases := []struct {
		word       string
		wantNormal string
	}{
		{"яснее", "яснее"},
		{"ясней", "яснее"},
		{"пояснее", "яснее"},
		{"поясней", "яснее"},
		{"абажурнее", "абажурнее"},
		{"поабажурнее", "абажурнее"}, // "по" + full base word, per docs/research/0003-...md's own example
		{"поправимее", "поправимее"},
		{"попоправимее", "поправимее"},
	}

	for _, tc := range cases {
		var norms []string
		for shard := range d.Words {
			for _, it := range d.Words[shard].SimilarItems(tc.word, d.CharPolicy) {
				if it.Key != tc.word {
					continue
				}
				for _, v := range it.Values {
					if len(v) != 4 {
						continue
					}
					paraID := binary.BigEndian.Uint16(v[:2])
					formIdx := binary.BigEndian.Uint16(v[2:4])
					if int(paraID) >= len(d.Paradigms[shard]) {
						continue
					}
					para := d.Paradigms[shard][paraID]
					if int(formIdx) >= para.Len() {
						continue
					}

					ownPrefix := strAt(d.Prefixes, para.Prefix(int(formIdx)))
					ownSuffix := strAt(d.Suffixes[shard], para.Suffix(int(formIdx)))
					stem := strings.TrimSuffix(strings.TrimPrefix(tc.word, ownPrefix), ownSuffix)

					p0 := strAt(d.Prefixes, para.Prefix(0))
					s0 := strAt(d.Suffixes[shard], para.Suffix(0))
					norms = append(norms, p0+stem+s0)
				}
			}
		}
		if len(norms) == 0 {
			t.Errorf("%q not found in any shard", tc.word)
			continue
		}
		found := false
		for _, n := range norms {
			if n == tc.wantNormal {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q resolved to normal forms %v, want at least one to be %q "+
				"(real dict.xml has legitimate cross-lemma homonyms for some surface strings — "+
				"see task-3-report.md for traced examples of exactly which lemmas collide)",
				tc.word, norms, tc.wantNormal)
		}
	}
}
