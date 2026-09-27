package internal

import (
	"encoding/base64"
	"sync"
	"unicode/utf8"
)

// LookupEach is SimilarItems without the intermediate []Item: for every
// stored key matching key under pol (CharPolicy substitutions), in exactly
// SimilarItems' order, fn is called once per payload value, in
// ValuesForIndex order. found is the stored key's text — key itself (no
// allocation) when no substitution was taken. value is decoded into a
// scratch buffer and is valid only during the call; undecodable values are
// skipped.
func (d *DAWG) LookupEach(key string, pol *CharPolicy, alphabet Alphabet, fn func(found string, value []byte)) {
	d.lookupFrom(key, 0, 0, "", false, pol, alphabet, fn)
}

// lookupFrom matches key[pos:] from DAWG node index. head is the matched
// text for key[:pos] when substituted is true (a substitution was taken
// earlier on this path); otherwise the matched text is key[:pos] itself.
// It mirrors similarItemsRecursive: the straight path (no further
// substitutions) is reported first, then every substitution branch in
// position order.
func (d *DAWG) lookupFrom(key string, pos int, index uint32, head string, substituted bool,
	pol *CharPolicy, alphabet Alphabet, fn func(string, []byte)) {
	// Pass 1: the straight path. Like similarItemsRecursive, an exhausted
	// key looks for the payload edge right at index (the root for "").
	end, ok := index, true
	if pos < len(key) {
		end = d.followStringVia(alphabet, key[pos:], index)
		ok = end != 0
	}
	if ok {
		if sep := d.FollowByte(PayloadSeparator, end); sep != 0 {
			found := key
			if substituted {
				found = head + key[pos:]
			}
			d.forEachValue(sep, func(v []byte) { fn(found, v) })
		}
	}
	if pol == nil || len(pol.Substitutions) == 0 {
		return
	}
	// Pass 2: branch at every substitutable rune along the straight path.
	for i := pos; i < len(key); {
		r, size := utf8.DecodeRuneInString(key[i:])
		for _, s := range pol.Substitutions {
			if s.From != r {
				continue
			}
			if next := d.followRuneVia(alphabet, s.To, index); next != 0 {
				prefix := key[:i]
				if substituted {
					prefix = head + key[pos:i]
				}
				d.lookupFrom(key, i+size, next, prefix+string(s.To), true, pol, alphabet, fn)
			}
		}
		if index = d.followRuneVia(alphabet, r, index); index == 0 {
			return
		}
		i += size
	}
}

// followStringVia follows every rune of s from index (see followRuneVia);
// 0 when the path breaks.
func (d *DAWG) followStringVia(alphabet Alphabet, s string, index uint32) uint32 {
	for _, r := range s {
		if index = d.followRuneVia(alphabet, r, index); index == 0 {
			return 0
		}
	}
	return index
}

// decodeBufPool holds the 18-byte scratch buffers forEachValue base64-decodes
// values into. keyBuf/stackBuf (below) never escape their stack frame — only
// the decode buffer does, because it is handed to the caller-supplied fn,
// whose identity forEachValue cannot see (fn is an opaque parameter of a
// function too large for the compiler to inline at LookupEach's call sites:
// escape analysis must conservatively assume any callee might retain a
// pointer argument, so declaring the buffer with `var out [18]byte` forces
// a real heap allocation on every single call, independent of what the
// concrete fn actually does). Pooling sidesteps that: the buffer is
// allocated once (by Pool.New, on first use) and reused afterwards, so
// steady-state calls make no new allocations — see
// TestLookupEachFoundIsKeyWithoutSubstitution's AllocsPerRun assertion.
var decodeBufPool = sync.Pool{New: func() any { return new([18]byte) }}

// forEachValue calls fn with every payload value under the
// PayloadSeparator node sep, in ValuesForIndex order, base64-decoded into
// a pooled scratch buffer (see decodeBufPool). It walks the guide with
// nextTerminal (dawg.go) — the same step completer.next uses for
// ValuesForIndex/Walk — driven from local, stack-array-backed key/stack
// slices instead of completer's struct fields: nextTerminal only returns
// its updated slices (never stores them through a pointer receiver), so
// they stay off the heap here. They spill to the heap only for unusually
// long payloads (the make([]byte, n) fallback below).
func (d *DAWG) forEachValue(sep uint32, fn func(value []byte)) {
	if len(d.guide) == 0 {
		return
	}
	var keyBuf [24]byte
	var stackBuf [24]uint32
	key := keyBuf[:0]
	stack := append(stackBuf[:0], sep)
	var lastIndex uint32

	out := decodeBufPool.Get().(*[18]byte) //nolint:forcetypeassert // always what New returns
	defer decodeBufPool.Put(out)

	for {
		var ok bool
		key, stack, lastIndex, ok = d.nextTerminal(key, stack, lastIndex)
		if !ok {
			return
		}

		dst := out[:]
		if n := base64.StdEncoding.DecodedLen(len(key)); n > len(dst) {
			dst = make([]byte, n)
		}
		n, err := base64.StdEncoding.Decode(dst, key)
		if err != nil {
			continue
		}
		fn(dst[:n])
	}
}

// FindJoined looks up the key a + string(sep) + b exactly, like Find, but
// without building the concatenated string.
func (d *DAWG) FindJoined(a string, sep byte, b string) uint32 {
	// Node 0 is both the root and FollowByte's "no edge" result (see
	// lookupFrom), so an empty a must not run through Follow: it cannot
	// tell "stayed at the root" from "failed". Only check for failure when
	// a is non-empty, matching Find("" + string(sep) + b) exactly.
	index := uint32(0)
	if a != "" {
		index = d.Follow(a, 0)
		if index == 0 {
			return 0
		}
	}
	if index = d.FollowByte(sep, index); index == 0 {
		return 0
	}
	if index = d.Follow(b, index); index == 0 {
		return 0
	}
	return d.Value(index)
}
