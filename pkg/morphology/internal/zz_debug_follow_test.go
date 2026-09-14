package internal

import (
	"bytes"
	"sort"
	"testing"
)

func TestDebugFollowKo(t *testing.T) {
	var keys []string
	for _, w := range []string{"кот", "кода", "коды", "коду"} {
		for i := 0; i < 3; i++ {
			keys = append(keys, w+string(PayloadSeparator)+string(bytes.Repeat([]byte{'A' + byte(i)}, 8)))
		}
	}
	for _, w := range []string{"дом", "дома", "дому", "ёж", "ежа", "овёс", "овса"} {
		for i := 0; i < 3; i++ {
			keys = append(keys, w+string(PayloadSeparator)+string(bytes.Repeat([]byte{'A' + byte(i)}, 8)))
		}
	}
	sort.Strings(keys)
	d, err := BuildDAWG(keys)
	if err != nil {
		t.Fatal(err)
	}

	walk := func(key string) {
		idx := uint32(0)
		for i := 0; i < len(key); i++ {
			next := d.FollowByte(key[i], idx)
			t.Logf("  %q: byte %q idx=%d unit=0x%08x off=%d -> next=%d", key, key[i], idx, d.dict[idx], offset(d.dict[idx]), next)
			if next == 0 {
				t.Logf("  X BREAK at %q", key[i])
				return
			}
			idx = next
		}
		t.Logf("  %q OK (leaf value=%v)", key, d.HasValue(idx))
	}
	walk("кода\x01AAAAAAAA")
	walk("кот\x01AAAAAAAA")
	walk("дом\x01AAAAAAAA")

	for _, k := range []string{"кода\x01AAAAAAAA", "кот\x01AAAAAAAA", "коду\x01BBBBBBBB", "дом\x01AAAAAAAA", "овса\x01CCCCCCCC"} {
		t.Logf("contains %s: %v", k, d.Contains(k))
	}
}
func itos(v int32) string {
	var b [20]byte
	i := len(b)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		b[i] = "0123456789abcdef"[v&15]
		v >>= 4
	}
	for i > 0 {
		i--
		b[i] = '0'
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
