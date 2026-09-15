//go:build integration

package xmlscan_test

import (
	"compress/bzip2"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/xmlscan"
)

const (
	dictEnvVar     = "GOMORPHY_DICT_XML"
	defaultDictXML = ".data/opencorpora/dict.xml"

	wantLemmas = 391842
	wantLines  = 5533109 // lemma words (<l>) + form words (<f>)
)

func openDict(t *testing.T) *os.File {
	t.Helper()

	path := os.Getenv(dictEnvVar)
	if path == "" {
		path = findDictXML()
		if path == "" {
			t.Skipf("dict.xml not found (%s set to override)", dictEnvVar)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open dict %q: %v", path, err)
	}

	return f
}

// findDictXML walks up from the working dir looking for .data/opencorpora/dict.xml
// so tests pass both from package dir and repo root.
func findDictXML() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		candidate := filepath.Join(dir, defaultDictXML)
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

type counter struct {
	lemmas      int
	forms       int
	grefs       int
	grammemes   int
	lastLemmaID uint32
}

func (c *counter) OnGrammeme(_, _ []byte) error { c.grammemes++; return nil }

func (c *counter) OnGrammemeRef([]byte) error { c.grefs++; return nil }

func (c *counter) OnLemma(id uint32, _ []byte) error {
	c.lemmas++
	c.lastLemmaID = id

	return nil
}

func (c *counter) OnLemmaHeadEnd() error { return nil }

func (c *counter) OnForm([]byte) error { c.forms++; return nil }

func (c *counter) OnFormEnd() error { return nil }

func TestIntegrationFirst10kLemmas(t *testing.T) {
	f := openDict(t)

	defer f.Close()

	c := &counter{}
	sc := xmlscan.New(f, c)

	if err := sc.Scan(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if c.lemmas < 10_000 {
		t.Errorf("expected at least 10000 lemmas, got %d", c.lemmas)
	}
}

func TestIntegrationFullCounts(t *testing.T) {
	f := openDict(t)

	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(f.Name(), ".bz2") {
		r = bzip2.NewReader(f)
	}

	c := &counter{}
	sc := xmlscan.New(r, c)

	if err := sc.Scan(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	t.Logf(
		"lemmas=%d, forms=%d, lines=%d (want %d), grammeme refs=%d, grammemes=%d",
		c.lemmas, c.forms, c.lemmas+c.forms, wantLines, c.grefs, c.grammemes,
	)

	if c.lemmas != wantLemmas {
		t.Errorf("lemmas: got %d, want %d", c.lemmas, wantLemmas)
	}

	if got := c.lemmas + c.forms; got != wantLines {
		t.Errorf("lines (lemma words + form words): got %d, want %d", got, wantLines)
	}
}
