//go:build integration

package build_test

import (
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/xmlscan"
)

// refCounter independently recounts unique (text, ordered-ancode) pairs and
// ancodes straight from XML attributes, bypassing the Builder entirely.
type refCounter struct {
	text    string
	grams   []string
	pairs   map[string]struct{}
	ancodes map[string]struct{}
	lines   int
	lemmas  int
}

func (c *refCounter) OnGrammeme(_, _ []byte) error { return nil }

func (c *refCounter) OnGrammemeRef(v []byte) error {
	c.grams = append(c.grams, string(v))

	return nil
}

func (c *refCounter) OnLemma(_ uint32, text []byte) error {
	c.text = string(text)
	c.grams = c.grams[:0]

	return nil
}

func (c *refCounter) OnLemmaEnd() error {
	c.lemmas++
	c.commit()

	return nil
}

func (c *refCounter) OnForm(text []byte) error {
	c.text = string(text)
	c.grams = c.grams[:0]

	return nil
}

func (c *refCounter) OnFormEnd() error {
	c.commit()

	return nil
}

func (c *refCounter) commit() {
	c.lines++
	ak := strings.Join(c.grams, ",")
	c.ancodes[ak] = struct{}{}
	c.pairs[c.text+"|"+ak] = struct{}{}
}

func TestIntegrationReferenceCounts(t *testing.T) {
	f := openDict(t)

	defer f.Close()

	c := &refCounter{
		pairs:   map[string]struct{}{},
		ancodes: map[string]struct{}{},
	}

	if err := xmlscan.New(f, c).Scan(); err != nil {
		t.Fatal(err)
	}

	t.Logf("lemmas=%d lines=%d uniquePairs=%d uniqueAncodes=%d",
		c.lemmas, c.lines, len(c.pairs), len(c.ancodes))

	if c.lemmas != 391_842 {
		t.Errorf("lemmas: got %d, want 391842", c.lemmas)
	}

	if c.lines != 5_533_109 {
		t.Errorf("lines: got %d, want 5533109", c.lines)
	}

	if len(c.pairs) != 5_393_737 {
		t.Errorf("pairs: got %d, want 5393737", len(c.pairs))
	}

	if len(c.ancodes) != 876 {
		t.Errorf("ancodes: got %d, want 876", len(c.ancodes))
	}
}
