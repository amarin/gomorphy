package xmlscan_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/xmlscan"
)

type recorder struct {
	events []string
}

func (r *recorder) OnGrammeme(parent, name []byte) error {
	r.events = append(r.events, fmt.Sprintf("grammeme parent=%q name=%q", string(parent), string(name)))

	return nil
}

func (r *recorder) OnGrammemeRef(v []byte) error {
	r.events = append(r.events, "gref "+string(v))

	return nil
}

func (r *recorder) OnLemma(id uint32, text []byte) error {
	r.events = append(r.events, fmt.Sprintf("lemma %d %q", id, string(text)))

	return nil
}

func (r *recorder) OnLemmaHeadEnd() error { r.events = append(r.events, "lemma-end"); return nil }

func (r *recorder) OnDictionaryRoot(version, revision []byte) error {
	r.events = append(r.events, fmt.Sprintf("root version=%q revision=%q", string(version), string(revision)))

	return nil
}

func (r *recorder) OnForm(text []byte) error {
	r.events = append(r.events, "form "+string(text))

	return nil
}

func (r *recorder) OnFormEnd() error { r.events = append(r.events, "form-end"); return nil }

func (r *recorder) OnLink(from, to, linkType []byte) error {
	r.events = append(r.events, fmt.Sprintf("link from=%q to=%q type=%q", string(from), string(to), string(linkType)))

	return nil
}

func scanAll(t *testing.T, input string, bufSize int) []string {
	t.Helper()

	rec := &recorder{}
	sc := xmlscan.NewBufferSize(strings.NewReader(input), rec, bufSize)

	if err := sc.Scan(); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	return rec.events
}

const sampleDict = `<?xml version="1.0" encoding="utf-8"?>
<dictionary version="0.92" revision="417257">
<grammemes>
<grammeme parent="">POST</grammeme>
<grammeme parent="POST"><name>NOUN</name><alias>сущ</alias><description>noun</description></grammeme>
<grammeme parent="NOUN" auto="1"/>
</grammemes>
<link_types><type id="1">ADJF-ADJS</type></link_types>
<restrictions>
<restr type="obligatory" auto="0"><left type="lemma"><g v="NOUN"/></left><right type="lemma">sing</right></restr>
</restrictions>
<lemmata>
<lemma id="5"><l t="ёжик"><g v="NOUN"/><g v="inan"/><g v="masc"/></l>
<f t="ёжик"><g v="nomn"/></f><f t="ёжика"><g v="gent"/></f></lemma>
<lemma id="6"><l t="весёлый" t2=""><g v="ADJF"/></l><f t="весёлый"/></lemma>
</lemmata>
<links>
<link id="1" from="5" to="6" type="3"/>
</links>
</dictionary>`

func wantEvents() []string {
	return []string{
		`root version="0.92" revision="417257"`,
		`grammeme parent="" name=""`,
		`grammeme parent="POST" name="NOUN"`,
		`grammeme parent="NOUN" name=""`,
		`lemma 5 "ёжик"`,
		"gref NOUN",
		"gref inan",
		"gref masc",
		"lemma-end",
		"form ёжик",
		"gref nomn",
		"form-end",
		"form ёжика",
		"gref gent",
		"form-end",
		`lemma 6 "весёлый"`,
		"gref ADJF",
		"lemma-end",
		"form весёлый",
		"form-end",
		`link from="5" to="6" type="3"`,
	}
}

func equalEvents(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func dumpEvents(label string, ev []string) string {
	var sb strings.Builder

	sb.WriteString(label + ":\n")

	for i, e := range ev {
		fmt.Fprintf(&sb, "%3d %s\n", i, e)
	}

	return sb.String()
}

func diffEvents(a, b []string) string {
	var sb strings.Builder

	for i := 0; i < len(a) || i < len(b); i++ {
		switch {
		case i >= len(a):
			fmt.Fprintf(&sb, "- missing: %s\n", b[i])
		case i >= len(b):
			fmt.Fprintf(&sb, "- extra: %s\n", a[i])
		case a[i] != b[i]:
			fmt.Fprintf(&sb, "- got: %s\n- want: %s\n", a[i], b[i])
		}
	}

	return sb.String()
}

func TestScanEntitiesAndSelfClosing(t *testing.T) {
	xml := `<dictionary version="0.92"><lemmata>` +
		`<lemma id="1"><l t="a &amp; b &lt;c&gt;"><g v="X"/></l><f t="q&quot;z"/></lemma>` +
		`<lemma id="2"><l t="it&apos;s"/></lemma>` +
		`</lemmata></dictionary>`

	want := []string{
		`root version="0.92" revision=""`,
		`lemma 1 "a & b <c>"`,
		"gref X",
		"lemma-end",
		"form q\"z",
		"form-end",
		`lemma 2 "it's"`,
		"lemma-end",
	}

	for _, size := range []int{16, 32, 512} {
		got := scanAll(t, xml, size)
		if !equalEvents(got, want) {
			t.Errorf("buffer=%d mismatch:\n%s", size, diffEvents(got, want))
		}
	}
}

func TestScanErrUnexpectedEOFOpenTag(t *testing.T) {
	sc := xmlscan.NewBufferSize(strings.NewReader("<lemmata><lemma id=1"), &recorder{}, 64)
	if err := sc.Scan(); err == nil {
		t.Fatal("expected error for truncated input")
	}
}

func TestScanEmpty(t *testing.T) {
	if err := xmlscan.NewBufferSize(strings.NewReader(""), &recorder{}, 64).Scan(); err != nil {
		t.Fatalf("empty input: %v", err)
	}

	if ev := scanAll(t, "", 16); len(ev) != 0 {
		t.Errorf("expected no events, got %v", ev)
	}
}

// TestScanCommentAcrossBufferBoundary guards against a bug where a read
// buffer boundary landing right after "<!" (before "--") made the scanner
// misclassify a comment as a bare "<!...>" declaration and truncate its skip
// at the first '>' inside the comment text instead of its real "-->". A
// stray "<g .../>" planted after that first '>' (but still inside the real
// comment) makes the difference observable: it only fires a gref event if
// the comment's own delimiter search never actually reached "-->". Buffer
// refills happen at fixed-size chunks from the start of input, so a fixed
// buffer size with a varying-length filler prefix sweeps every possible
// alignment of "<!" relative to a fill boundary.
func TestScanCommentAcrossBufferBoundary(t *testing.T) {
	const size = 16

	suffix := `<dictionary><lemmata><lemma id="1"><l t="ёж">` +
		`<!-- a > <g v="BOGUS"/> --><g v="NOUN"/></l><f t="ёж"/></lemma>` +
		`</lemmata></dictionary>`

	want := []string{
		`root version="" revision=""`,
		`lemma 1 "ёж"`,
		"gref NOUN",
		"lemma-end",
		"form ёж",
		"form-end",
	}

	for prefixLen := range size {
		xml := strings.Repeat("x", prefixLen) + suffix

		got := scanAll(t, xml, size)
		if !equalEvents(got, want) {
			t.Fatalf("prefixLen=%d mismatch:\n%s", prefixLen, diffEvents(got, want))
		}
	}
}

func TestScanNumericEntities(t *testing.T) {
	xml := `<dictionary><lemmata>` +
		`<lemma id="1"><l t="&#39;a&#x27;&#1046;"/></lemma>` +
		`</lemmata></dictionary>`

	want := []string{
		`root version="" revision=""`,
		"lemma 1 \"'a'Ж\"",
		"lemma-end",
	}

	got := scanAll(t, xml, 64)
	if !equalEvents(got, want) {
		t.Fatalf("mismatch:\n%s", diffEvents(got, want))
	}
}

func TestScanSample(t *testing.T) {
	for _, size := range []int{16, 64, 256, 4096} {
		got := scanAll(t, sampleDict, size)

		want := wantEvents()
		if !equalEvents(got, want) {
			t.Errorf("buffer=%d mismatch:\n%s", size,
				dumpEvents("got vs want\n"+diffEvents(got, want), got))
		}
	}
}
