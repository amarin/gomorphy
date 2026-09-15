package internal

import (
	"fmt"
	"sort"
)

// Alphabet encodes a string (a DAWG key's word portion) into a byte
// sequence suitable for the DAWG engine, and decodes it back. The DAWG
// engine (dict []uint32 + guide []byte, dawg.go/dawgbuild.go) is
// alphabet-agnostic: FollowByte/Follow already walk a key one raw byte at
// a time, so a dense multi-byte-per-character alphabet needs no engine
// changes - only a codec at the key-construction boundary. See
// docs/superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md.
type Alphabet interface {
	// Name identifies the alphabet for logging/comparison output.
	Name() string

	// Encode converts s into the byte sequence to use as DAWG key
	// material. Returns an error if s contains a rune the alphabet
	// cannot represent.
	Encode(s string) ([]byte, error)

	// Decode is Encode's inverse: reconstructs the original string from
	// a byte sequence previously produced by Encode. Returns an error on
	// malformed input.
	Decode(b []byte) (string, error)
}

// IdentityAlphabet is a raw UTF-8 passthrough - today's actual DAWG key
// encoding, and the only correct choice for reading original pymorphy2
// words.dawg files (unchanged by this plan).
type IdentityAlphabet struct{}

func (IdentityAlphabet) Name() string { return "identity" }

func (IdentityAlphabet) Encode(s string) ([]byte, error) { return []byte(s), nil }

func (IdentityAlphabet) Decode(b []byte) (string, error) { return string(b), nil }

// DenseAlphabet maps each rune in a fixed corpus to a code of exactly
// width bytes (1 or 2), reserving code 0 (the DAWG engine's
// guide-traversal "no child/sibling" sentinel, see ForEachChild in
// dawg.go) and code 1 (PayloadSeparator, dawg.go) so a dense-coded
// character byte sequence can never be confused with either. Codes are
// assigned in ascending rune order for determinism: the same corpus
// always produces the same codec, byte for byte, run to run.
type DenseAlphabet struct {
	width  int
	codeOf map[rune]uint16
	runeOf []rune // runeOf[code-2] == r for codeOf[r] == code (codes start at 2)
}

// maxCodesForWidth returns how many non-reserved codes a dense alphabet of
// the given width can address: 256^width total codes, minus the 2
// reserved (0 and 1).
func maxCodesForWidth(width int) int {
	total := 1
	for i := 0; i < width; i++ {
		total *= 256
	}
	return total - 2
}

// NewDenseAlphabet builds a DenseAlphabet of the given width (1 or 2) from
// every distinct rune found across corpus. Returns an error if width is
// not 1 or 2, or if corpus contains more distinct runes than width can
// address (254 for width 1, 65534 for width 2) - this is a hard failure,
// not silent truncation or wraparound.
func NewDenseAlphabet(width int, corpus []string) (*DenseAlphabet, error) {
	if width != 1 && width != 2 {
		return nil, fmt.Errorf("internal: DenseAlphabet width must be 1 or 2, got %d", width)
	}

	seen := map[rune]bool{}
	for _, s := range corpus {
		for _, r := range s {
			seen[r] = true
		}
	}

	max := maxCodesForWidth(width)
	if len(seen) > max {
		return nil, fmt.Errorf("internal: corpus has %d distinct runes, exceeds %d-byte alphabet's capacity of %d", len(seen), width, max)
	}

	runes := make([]rune, 0, len(seen))
	for r := range seen {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })

	codeOf := make(map[rune]uint16, len(runes))
	runeOf := make([]rune, len(runes))
	for i, r := range runes {
		code := uint16(2 + i)
		codeOf[r] = code
		runeOf[i] = r
	}

	return &DenseAlphabet{width: width, codeOf: codeOf, runeOf: runeOf}, nil
}

func (a *DenseAlphabet) Name() string {
	return fmt.Sprintf("dense-%d", a.width)
}

func (a *DenseAlphabet) Encode(s string) ([]byte, error) {
	buf := make([]byte, 0, len(s)*a.width)
	for _, r := range s {
		code, ok := a.codeOf[r]
		if !ok {
			return nil, fmt.Errorf("internal: rune %q not in DenseAlphabet (width %d)", r, a.width)
		}
		if a.width == 1 {
			buf = append(buf, byte(code))
		} else {
			buf = append(buf, byte(code>>8), byte(code))
		}
	}
	return buf, nil
}

func (a *DenseAlphabet) Decode(b []byte) (string, error) {
	if len(b)%a.width != 0 {
		return "", fmt.Errorf("internal: byte sequence length %d is not a multiple of width %d", len(b), a.width)
	}
	out := make([]rune, 0, len(b)/a.width)
	for i := 0; i < len(b); i += a.width {
		var code uint16
		if a.width == 1 {
			code = uint16(b[i])
		} else {
			code = uint16(b[i])<<8 | uint16(b[i+1])
		}
		if code < 2 || int(code)-2 >= len(a.runeOf) {
			return "", fmt.Errorf("internal: code %d has no known rune", code)
		}
		out = append(out, a.runeOf[code-2])
	}
	return string(out), nil
}
