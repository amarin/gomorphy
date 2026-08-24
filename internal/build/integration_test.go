//go:build integration

package build_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/amarin/gomorphy/internal/build"
	"github.com/amarin/gomorphy/internal/xmlscan"
)

func openDict(t *testing.T) *os.File {
	t.Helper()

	path := os.Getenv("GOMORPHY_DICT_XML")
	if path == "" {
		dir, _ := os.Getwd()

		for {
			candidate := filepath.Join(dir, ".data", "opencorpora", "dict.xml")
			if _, err := os.Stat(candidate); err == nil {
				path = candidate

				break
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				t.Skip("dict.xml not found")
			}

			dir = parent
		}
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open dict: %v", err)
	}

	return f
}

// xmlFeed adapts xmlscan events to Builder calls.
type xmlFeed struct {
	b        *build.Builder
	curText  string
	curLemma int
	lGrams   []string
	fGrams   []string
	haveForm bool
	err      error

	rawPairs   map[string]struct{}
	rawAncodes map[string]struct{}
}

func (f *xmlFeed) commitRaw(grams []string) {
	ak := strings.Join(grams, ",")
	f.rawAncodes[ak] = struct{}{}
	f.rawPairs[f.curText+"|"+ak] = struct{}{}
}

func (f *xmlFeed) fail(err error) {
	if f.err == nil {
		f.err = err
	}
}

func (f *xmlFeed) OnGrammeme(_, name []byte) error {
	if len(name) > 0 {
		if _, err := f.b.AddGrammeme(string(name)); err != nil {
			f.fail(err)
		}
	}

	return nil
}

func (f *xmlFeed) OnGrammemeRef(v []byte) error {
	if f.haveForm {
		f.fGrams = append(f.fGrams, string(v))
	} else {
		f.lGrams = append(f.lGrams, string(v))
	}

	return nil
}

func (f *xmlFeed) OnLemma(_ uint32, text []byte) error {
	f.curText = string(text)
	f.lGrams = f.lGrams[:0]

	return nil
}

func (f *xmlFeed) OnLemmaEnd() error {
	id, err := f.b.AddLemma(f.curText, f.lGrams...)
	if err != nil {
		f.fail(err)

		return nil
	}

	f.curLemma = id
	f.commitRaw(f.lGrams)

	return nil
}

func (f *xmlFeed) OnForm(text []byte) error {
	f.curText = string(text)
	f.fGrams = f.fGrams[:0]
	f.haveForm = true

	return nil
}

func (f *xmlFeed) OnFormEnd() error {
	if err := f.b.AddForm(f.curLemma, f.curText, f.fGrams...); err != nil {
		f.fail(err)
	}

	f.haveForm = false
	f.commitRaw(f.fGrams)

	return nil
}

func TestIntegrationFullBuild(t *testing.T) {
	f := openDict(t) //nolint:govet // closed via defer below

	defer f.Close()

	b := build.NewBuilderWithCapacity(3_100_000, 400_000, 5_500_000)
	feed := &xmlFeed{b: b, rawPairs: map[string]struct{}{}, rawAncodes: map[string]struct{}{}}

	start := time.Now()
	sc := xmlscan.New(f, feed)

	if err := sc.Scan(); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if feed.err != nil {
		t.Fatalf("builder error: %v", feed.err)
	}

	scanDur := time.Since(start)

	buildStart := time.Now()
	snap := b.Build()
	buildDur := time.Since(buildStart)

	var mem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mem)

	t.Logf(
		"scan=%s build=%s | texts=%d lemmas=%d pairs=%d states=%d grammemes=%d ancodes=%d | heap=%.1f MB",
		scanDur, buildDur,
		snap.TextCount(), snap.LemmaCount(), snap.PairCount(), snap.StateCount(),
		len(snap.GrammemeNames), len(snap.AncodeOff)-1,
		float64(mem.HeapAlloc)/(1<<20),
	)

	// Reference metrics from docs/requirements.md (dict.xml rev 417257).
	const (
		refLemmas  = 391_842
		refLines   = 5_533_109
		refPairs   = 5_393_737
		refAncodes = 876
	)

	if snap.LemmaCount() != refLemmas {
		t.Errorf("lemmas: got %d, want %d", snap.LemmaCount(), refLemmas)
	}

	if got := len(snap.AncodeOff) - 1; got != refAncodes {
		t.Errorf("ancodes: got %d, want %d", got, refAncodes)
	}

	if snap.PairCount() != refPairs {
		t.Errorf("pairs: got %d, want %d", snap.PairCount(), refPairs)
	}

	totalLines := 0
	for p := 0; p < snap.PairCount(); p++ {
		totalLines += len(snap.PairLemmasOf(uint32(p)))
	}

	if totalLines != refLines {
		t.Errorf("lines: got %d, want %d", totalLines, refLines)
	}

	// Cross-check snapshot pairs against independently collected raw keys.
	if snap.PairCount() != len(feed.rawPairs) {
		t.Errorf("pair sets diverge: snapshot=%d raw=%d", snap.PairCount(), len(feed.rawPairs))
	}

	// Spot-check a few common words resolve.
	for _, w := range []string{"кот", "дом", "быть"} {
		state, ok := snap.LookupWord([]byte(strings.ToLower(w)))
		if !ok {
			t.Errorf("word %q not found in built dictionary", w)

			continue
		}

		var first []uint32

		for _, p := range snap.PostingsOf(state) {
			first = snap.PairLemmasOf(p)

			break
		}

		if len(first) == 0 {
			t.Errorf("word %q has no lemmas", w)

			continue
		}

		t.Logf("%q -> lemma %q (%d postings)", w, snap.LemmaText(first[0]), len(snap.PostingsOf(state)))
	}
}
