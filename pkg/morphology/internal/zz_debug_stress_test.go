package internal

import (
	"bytes"
	"sort"
	"testing"
)

func TestDebugStress(t *testing.T) {
	var keys []string
	for _, w := range []string{"кот", "кода", "коды", "коду", "дом", "дома", "дому", "ёж", "ежа", "овёс", "овса"} {
		for i := 0; i < 3; i++ {
			keys = append(keys, w+string(PayloadSeparator)+string(bytes.Repeat([]byte{'A' + byte(i)}, 8)))
		}
	}
	sort.Strings(keys)

	d, err := BuildDAWG(keys)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("dict=%d units, guide=%d bytes", len(d.dict), len(d.guide))
	bad := 0
	for _, k := range keys {
		if !d.Contains(k) {
			t.Logf("MISSING: %q", k)
			bad++
		}
	}
	t.Logf("missing=%d/%d", bad, len(keys))
	for _, k := range []string{"кот", "домашний\x01AAAA", "ёж\x01AAAA"} {
		if d.Contains(k) {
			t.Logf("FALSE POSITIVE: %q", k)
		}
	}
	c := &completer{dawg: d}
	c.start(0, "")
	seen := 0
	for c.next() {
		seen++
	}
	t.Logf("completer enumerated=%d keys", seen)
}
