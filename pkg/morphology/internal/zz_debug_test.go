package internal

import (
	"encoding/base64"
	"testing"
)

func TestDebugBuildDAWG(t *testing.T) {
	payload := func(para, form uint16) string {
		return base64.StdEncoding.EncodeToString([]byte{
			byte(para >> 8), byte(para), byte(form >> 8), byte(form),
		})
	}
	keys := []string{
		"кот" + string(PayloadSeparator) + payload(0, 0),
		"кот" + string(PayloadSeparator) + payload(1, 0),
		"кота" + string(PayloadSeparator) + payload(0, 1),
		"ёжик" + string(PayloadSeparator) + payload(2, 0),
		"ежик" + string(PayloadSeparator) + payload(2, 0),
	}

	d, err := BuildDAWG(keys)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("dict=%d units, guide=%d bytes", len(d.dict), len(d.guide))

	idx := d.Follow("кот", 0)
	t.Logf("after 'кот': idx=%d hasLeaf=%v off=%d", idx, d.HasValue(idx), offset(d.dict[idx]))
	if idx != 0 {
		d.ForEachChild(idx, func(label byte, next uint32) {
			t.Logf("  child %q next=%d hasLeaf=%v", label, next, d.HasValue(next))
		})
	}

	items := d.SimilarItems("кот", RussianCharPolicy(), nil)
	for _, it := range items {
		t.Logf("item %q values=%v", it.Key, it.Values)
	}

	for _, k := range keys {
		t.Logf("contains %q: %v", k, d.Contains(k))
	}

	c := &completer{dawg: d}
	c.start(0, "")
	for c.next() {
		t.Logf("completer: %q", string(c.key))
	}

	// Dump guide contents for the walk from root.
	root := d.Follow("", 0)
	t.Logf("root idx=%d unit=0x%08x off=%d", root, d.dict[root], offset(d.dict[root]))
	d.ForEachChild(root, func(label byte, next uint32) {
		sib := guideSibling(d.guide, next)
		t.Logf("  child %q next=%d guide[%d*2]=%q guide[%d*2+1]=%q", label, next, next, d.guide[next*2], next, sib)
	})
}
