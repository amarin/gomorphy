package internal

import (
	"encoding/base64"
	"encoding/binary"
	"io"
	"unicode/utf8"
	"unsafe"
)

// Layout of a dawgdic dictionary unit (uint32):
//   - bits 0–7:   label (transition byte)
//   - bit  8:     has_leaf (the node has a value)
//   - bit  9:     extension (high bits of the offset field)
//   - bits 10–31: offset (or the value itself, if the unit is a value unit)
const (
	isLeafBit    = 1 << 31
	hasLeafBit   = 1 << 8
	extensionBit = 1 << 9
)

// PayloadSeparator separates the word from the payload in a words.dawg (pymorphy2) key.
const PayloadSeparator byte = 0x01

// DAWG — a read-only minimized finite-state automaton (dawgdic format):
// a dictionary array of units + a guide (2 bytes per node: child + sibling).
type DAWG struct {
	dict  []uint32
	guide []byte
}

// NewDAWG creates a DAWG from ready-made arrays. The arrays are not copied.
func NewDAWG(dict []uint32, guide []byte) *DAWG {
	return &DAWG{dict: dict, guide: guide}
}

// ReadDAWG reads a dictionary in the pymorphy2 words.dawg format:
// uint32 count + count×uint32 units + uint32 count_guide + guide (count×2 bytes).
func ReadDAWG(r io.Reader) (*DAWG, error) {
	var size uint32
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return nil, err
	}
	dict := make([]uint32, size)
	if err := binary.Read(r, binary.LittleEndian, dict); err != nil {
		return nil, err
	}
	var guideSize uint32
	if err := binary.Read(r, binary.LittleEndian, &guideSize); err != nil {
		return nil, err
	}
	guide := make([]byte, guideSize*2)
	if err := binary.Read(r, binary.LittleEndian, guide); err != nil {
		return nil, err
	}
	return &DAWG{dict: dict, guide: guide}, nil
}

// Bytes serializes the DAWG into the words.dawg stream format (the inverse of ReadDAWG):
// uint32 count + count×uint32 units + uint32 guide_size + guide.
func (d *DAWG) Bytes() []byte {
	if d == nil {
		return nil
	}
	w := 4 + len(d.dict)*4 + 4 + len(d.guide)
	buf := make([]byte, w)
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(d.dict)))
	p := 4
	for _, u := range d.dict {
		binary.LittleEndian.PutUint32(buf[p:], u)
		p += 4
	}
	binary.LittleEndian.PutUint32(buf[p:], uint32(len(d.guide)/2))
	p += 4
	copy(buf[p:], d.guide)
	return buf
}

// ParseDAWG parses a DAWG from serialized bytes (the Bytes format).
// When the unit array is 4-byte aligned, the dictionary and guide alias
// the input slice (zero-copy for mmap); otherwise, they are copied.
func ParseDAWG(data []byte) (*DAWG, error) {
	dict, guide, err := splitDAWG(data)
	if err != nil {
		return nil, err
	}
	return &DAWG{dict: dict, guide: guide}, nil
}

func splitDAWG(data []byte) ([]uint32, []byte, error) {
	if len(data) < 8 {
		return nil, nil, io.ErrUnexpectedEOF
	}
	size := int64(binary.LittleEndian.Uint32(data[:4]))
	if size < 0 || size*4+8 > int64(len(data)) {
		return nil, nil, io.ErrUnexpectedEOF
	}
	dictBytes := data[4 : 4+size*4]
	p := int64(4 + size*4)
	if p+4 > int64(len(data)) {
		return nil, nil, io.ErrUnexpectedEOF
	}
	guideSize := int64(binary.LittleEndian.Uint32(data[p : p+4]))
	p += 4
	guideLen := guideSize * 2
	if guideLen > int64(len(data))-p {
		return nil, nil, io.ErrUnexpectedEOF
	}
	guide := data[p : p+guideLen]

	var dict []uint32
	// Zero-copy path assumes the host's native byte order matches the
	// on-disk LittleEndian layout (true on amd64/arm64). On a big-endian
	// host (e.g. s390x) this aliasing would silently produce wrong values;
	// such hosts fall through to the explicit LittleEndian decode below only
	// when the alignment check fails, so this is not currently guarded.
	if size > 0 && uintptr(unsafe.Pointer(&dictBytes[0]))%4 == 0 {
		dict = unsafe.Slice((*uint32)(unsafe.Pointer(&dictBytes[0])), int(size))
	} else {
		dict = make([]uint32, int(size))
		for i := int64(0); i < size; i++ {
			dict[i] = binary.LittleEndian.Uint32(dictBytes[i*4:])
		}
	}
	return dict, guide, nil
}

func unitLabel(n uint32) uint32 {
	return n & (isLeafBit | 0xff)
}

// offset extracts the offset field written by encodable/dawgbuild.go: a
// plain 22-bit value, or (when extensionBit is set) a value pre-divided by
// 256 that must be shifted back up — see the encodable() comment for the
// on-disk unit layout this decodes.
func offset(n uint32) uint32 {
	return (n >> 10) << ((n & extensionBit) >> 6)
}

func hasLeafFlag(n uint32) bool {
	return n&hasLeafBit != 0
}

func valueOf(base uint32) uint32 {
	return base &^ isLeafBit
}

// FollowByte performs a transition on a single byte from node index.
// Returns 0 if there is no such transition.
func (d *DAWG) FollowByte(lbl byte, index uint32) uint32 {
	if index >= uint32(len(d.dict)) {
		return 0
	}
	off := offset(d.dict[index])
	next := index ^ off ^ uint32(lbl)
	if next >= uint32(len(d.dict)) {
		return 0
	}
	if unitLabel(d.dict[next]) != uint32(lbl) {
		return 0
	}
	return next
}

// FollowRune performs a transition on a rune (1-4 UTF-8 bytes) from node index.
func (d *DAWG) FollowRune(r rune, index uint32) uint32 {
	var buf [4]byte
	n := utf8.EncodeRune(buf[:], r)
	return d.followBytes(buf[:n], index)
}

// Follow performs a transition on a string from node index.
func (d *DAWG) Follow(s string, index uint32) uint32 {
	for i := 0; i < len(s); i++ {
		index = d.FollowByte(s[i], index)
		if index == 0 {
			return 0
		}
	}
	return index
}

func (d *DAWG) followBytes(bs []byte, index uint32) uint32 {
	for _, b := range bs {
		index = d.FollowByte(b, index)
		if index == 0 {
			return 0
		}
	}
	return index
}

// HasValue reports whether node index has a value.
func (d *DAWG) HasValue(index uint32) bool {
	return hasLeafFlag(d.dict[index])
}

// Value returns the integer value of node index (0 if it has none).
func (d *DAWG) Value(index uint32) uint32 {
	if index >= uint32(len(d.dict)) {
		return 0
	}
	off := offset(d.dict[index])
	valueIndex := index ^ off
	if valueIndex >= uint32(len(d.dict)) {
		return 0
	}
	return valueOf(d.dict[valueIndex])
}

// Find looks up a key exactly and returns its value (0 — key not found).
func (d *DAWG) Find(key string) uint32 {
	index := d.Follow(key, 0)
	if index == 0 {
		return 0
	}
	return d.Value(index)
}

// Contains reports whether the key is present in the dictionary (exact match).
func (d *DAWG) Contains(key string) bool {
	if len(d.dict) == 0 {
		return false
	}
	index := d.Follow(key, 0)
	return index != 0 && d.HasValue(index)
}

// ValuesForIndex returns all payload values of the sub-automaton rooted at
// node index (in words.dawg — base64-encoded record bytes after PayloadSeparator).
func (d *DAWG) ValuesForIndex(index uint32) [][]byte {
	var values [][]byte
	c := &completer{dawg: d}
	c.start(index, "")
	for c.next() {
		values = append(values, b64d(c.key))
	}
	return values
}

// ForEachChild calls fn for every outgoing edge of node index.
// Inside guide: the entry for index stores the label of the first child,
// and the child's slot stores the label of the next sibling; transitions
// are computed as a double-array hash. An edge to PayloadSeparator is
// included in the enumeration like any other.
func (d *DAWG) ForEachChild(index uint32, fn func(label byte, next uint32)) {
	if len(d.guide) == 0 {
		return
	}
	label := guideChild(d.guide, index)
	for label != 0 {
		next := d.FollowByte(label, index)
		if next == 0 {
			return
		}
		fn(label, next)
		label = guideSibling(d.guide, next)
	}
}

// HasPayloadChild reports whether the node has an outgoing PayloadSeparator edge.
// Unlike a FollowByte probe, the check goes through guide (the node's real
// edges), so it is not subject to double-array layout collisions.
func (d *DAWG) HasPayloadChild(index uint32) bool {
	found := false
	d.ForEachChild(index, func(label byte, _ uint32) {
		if label == PayloadSeparator {
			found = true
		}
	})
	return found
}

// Walk visits every (key, values) pair stored in the DAWG, in trie order.
// It reuses ForEachChild (edge traversal) and ValuesForIndex (payload
// enumeration under a PayloadSeparator edge) — the same primitives
// SimilarItems and ValuesForIndex already use for single-key lookups, just
// exhaustively instead of following a caller-given key. See
// docs/research/0005-pymorphy2-full-dawg-walk-cost.md for the validated
// approach and real-corpus timing (3,064,708 keys, 570ms).
func (d *DAWG) Walk(fn func(key string, values [][]byte)) {
	var walk func(index uint32, prefix []byte)
	walk = func(index uint32, prefix []byte) {
		d.ForEachChild(index, func(label byte, next uint32) {
			if label == PayloadSeparator {
				fn(string(prefix), d.ValuesForIndex(next))
				return
			}
			walk(next, append(prefix, label))
		})
	}
	walk(0, nil)
}

func b64d(p []byte) []byte {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(p)))
	n, err := base64.StdEncoding.Decode(dst, p)
	if err != nil {
		return nil
	}
	return dst[:n]
}

func guideChild(g []byte, n uint32) byte {
	return g[n*2]
}

func guideSibling(g []byte, n uint32) byte {
	return g[n*2+1]
}

// completer — traversal of a DAWG sub-automaton's values using guide.
type completer struct {
	dawg       *DAWG
	lastIndex  uint32
	indexStack []uint32
	key        []byte
}

func (c *completer) start(index uint32, prefix string) {
	c.key = []byte(prefix)
	if len(c.dawg.guide) > 0 {
		c.indexStack = []uint32{index}
	}
}

func (c *completer) next() bool {
	if len(c.indexStack) == 0 {
		return false
	}

	index := c.indexStack[len(c.indexStack)-1]

	if c.lastIndex != 0 {
		for {
			siblingLabel := guideSibling(c.dawg.guide, index)
			if len(c.key) > 0 {
				c.key = c.key[:len(c.key)-1]
			}

			c.indexStack = c.indexStack[:len(c.indexStack)-1]
			if len(c.indexStack) == 0 {
				return false
			}

			index = c.indexStack[len(c.indexStack)-1]
			if siblingLabel != 0 {
				if index = c.follow(siblingLabel, index); index == 0 {
					return false
				}
				break
			}
		}
	}

	return c.findTerminal(index)
}

func (c *completer) follow(label byte, index uint32) uint32 {
	index = c.dawg.FollowByte(label, index)
	if index == 0 {
		return 0
	}
	c.key = append(c.key, label)
	c.indexStack = append(c.indexStack, index)
	return index
}

func (c *completer) findTerminal(index uint32) bool {
	for !c.dawg.HasValue(index) {
		label := guideChild(c.dawg.guide, index)
		if index = c.dawg.FollowByte(label, index); index == 0 {
			return false
		}
		c.key = append(c.key, label)
		c.indexStack = append(c.indexStack, index)
	}
	c.lastIndex = index
	return true
}
