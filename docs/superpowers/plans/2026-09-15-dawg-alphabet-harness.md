# DAWG Dense-Alphabet Comparison Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a pluggable `Alphabet` codec (identity / dense-1-byte / dense-2-byte) and a repeatable, committed harness that measures DAWG size and build time across all three on real OpenCorpora wordforms with realistic payload-bearing keys — no production wiring, purely to produce numbers for a later decision.

**Architecture:** `Alphabet` is a pure string↔byte-sequence codec living entirely in `pkg/morphology/internal`, decoupled from the DAWG engine (`dawg.go`/`dawgbuild.go`/`dawgbuild_freelist.go`), which already operates one raw byte at a time and needs no changes. A `//go:build integration`-tagged test builds three DAWGs (one per alphabet) from the same real wordform corpus with synthetic-but-realistic payload and reports slot count, byte size, and build time for each.

**Tech Stack:** Go (stdlib `sort`, `fmt`, `bufio`, `encoding/base64`, `encoding/binary`, `math/rand`, `time`), `github.com/stretchr/testify` for unit tests (matching `dawgbuild_test.go`'s existing convention in this package).

**Spec:** [docs/superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md](../specs/2026-09-15-dawg-alphabet-harness-design.md)

## Global Constraints

- No changes to `import.go`, `parse.go`, `open.go`, `save.go`, or the `.dat` binary format.
- No CLI flag, format version marker, or `Open()`-time alphabet detection.
- Widths 1 and 2 bytes only — no 4-byte variant.
- Reserved codes: 0 (DAWG guide-traversal sentinel) and 1 (`PayloadSeparator`) must never be assigned to a real rune by `DenseAlphabet` — enforced, not just documented.
- `DenseAlphabet` construction must error (not silently truncate/wrap) when a corpus has more distinct runes than the chosen width can address.
- The harness must use realistic payload-bearing keys (`word + PayloadSeparator + base64(uint32 value)`), with **random**, not sequential, values — this session's spike found sequential values pathological at scale.
- `go test ./... -race` must stay green after every task (the new integration-tagged harness file is excluded from this by its build tag, which is expected).

---

## Task 1: `Alphabet` interface, `IdentityAlphabet`, `DenseAlphabet`

**Files:**
- Create: `pkg/morphology/internal/alphabet.go`
- Create: `pkg/morphology/internal/alphabet_test.go`

**Interfaces:**
- Produces: `type Alphabet interface { Name() string; Encode(s string) ([]byte, error); Decode(b []byte) (string, error) }`, `type IdentityAlphabet struct{}` (implements `Alphabet`), `type DenseAlphabet struct{...}` (implements `Alphabet`), `func NewDenseAlphabet(width int, corpus []string) (*DenseAlphabet, error)`.
- Consumes: `PayloadSeparator` (existing constant, `dawg.go:23`).

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/internal/alphabet_test.go`:

```go
package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityAlphabetRoundtrip(t *testing.T) {
	a := IdentityAlphabet{}
	assert.Equal(t, "identity", a.Name())

	cases := []string{"", "abc", "кот", "поясней"}
	for _, s := range cases {
		enc, err := a.Encode(s)
		require.NoError(t, err)
		dec, err := a.Decode(enc)
		require.NoError(t, err)
		assert.Equal(t, s, dec)
	}
}

func TestDenseAlphabetRoundtrip(t *testing.T) {
	corpus := []string{"кот", "кота", "мышь", "дом"}
	for _, width := range []int{1, 2} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			a, err := NewDenseAlphabet(width, corpus)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("dense-%d", width), a.Name())

			for _, s := range corpus {
				enc, err := a.Encode(s)
				require.NoError(t, err)
				dec, err := a.Decode(enc)
				require.NoError(t, err)
				assert.Equal(t, s, dec)
			}
		})
	}
}

func TestDenseAlphabetReservedCodesNeverAssigned(t *testing.T) {
	corpus := []string{"кот", "кота", "мышь", "дом", "абвгдеёжзийклмнопрстуфхцчшщъыьэюя"}
	for _, width := range []int{1, 2} {
		a, err := NewDenseAlphabet(width, corpus)
		require.NoError(t, err)
		for r, code := range a.codeOf {
			assert.NotEqual(t, uint16(0), code, "rune %q must not get reserved code 0", r)
			assert.NotEqual(t, uint16(PayloadSeparator), code, "rune %q must not get reserved code 1 (PayloadSeparator)", r)
		}
	}
}

func TestDenseAlphabetWidth1OverflowsOnTooManyRunes(t *testing.T) {
	// 255 distinct runes - one more than width 1's capacity of 254
	// (256 possible byte values minus the 2 reserved codes 0 and 1).
	var corpus []string
	for r := rune(0x400); r < 0x400+255; r++ {
		corpus = append(corpus, string(r))
	}
	_, err := NewDenseAlphabet(1, corpus)
	assert.Error(t, err)
}

func TestDenseAlphabetWidth2CapacityIsMuchLarger(t *testing.T) {
	// The same 255-rune corpus that overflows width 1 must fit
	// comfortably in width 2 (capacity 65534).
	var corpus []string
	for r := rune(0x400); r < 0x400+255; r++ {
		corpus = append(corpus, string(r))
	}
	_, err := NewDenseAlphabet(2, corpus)
	assert.NoError(t, err)
}

func TestDenseAlphabetInvalidWidth(t *testing.T) {
	_, err := NewDenseAlphabet(3, []string{"a"})
	assert.Error(t, err)
}

func TestDenseAlphabetDecodeRejectsMisalignedLength(t *testing.T) {
	a, err := NewDenseAlphabet(2, []string{"кот"})
	require.NoError(t, err)
	_, err = a.Decode([]byte{1, 2, 3}) // length 3, not a multiple of width 2
	assert.Error(t, err)
}

func TestDenseAlphabetEncodeRejectsUnknownRune(t *testing.T) {
	a, err := NewDenseAlphabet(1, []string{"кот"})
	require.NoError(t, err)
	_, err = a.Encode("мышь") // none of м/ы/ш/ь are in the "кот" corpus
	assert.Error(t, err)
}

func TestDenseAlphabetDeterministic(t *testing.T) {
	corpus := []string{"кот", "мышь", "дом", "яснее"}
	a1, err := NewDenseAlphabet(1, corpus)
	require.NoError(t, err)
	a2, err := NewDenseAlphabet(1, corpus)
	require.NoError(t, err)
	assert.Equal(t, a1.codeOf, a2.codeOf)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/morphology/internal/... -run 'TestIdentityAlphabet|TestDenseAlphabet' -v`
Expected: compile error — none of `IdentityAlphabet`, `DenseAlphabet`, `NewDenseAlphabet` exist yet.

- [ ] **Step 3: Implement `alphabet.go`**

Create `pkg/morphology/internal/alphabet.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/morphology/internal/... -run 'TestIdentityAlphabet|TestDenseAlphabet' -v`
Expected: PASS, all subtests green.

- [ ] **Step 5: Run the full existing test suite to check for regressions**

Run: `go test ./pkg/morphology/... -race`
Expected: PASS, no regressions (this task only adds new files, touches nothing existing).

- [ ] **Step 6: Commit**

```bash
git add pkg/morphology/internal/alphabet.go pkg/morphology/internal/alphabet_test.go
git commit -m "internal: add pluggable Alphabet codec (identity, dense-1, dense-2)

Pure string<->byte-sequence codec, decoupled from the DAWG engine
(dawg.go/dawgbuild.go need no changes - FollowByte already walks keys
one raw byte at a time, so a dense multi-byte-per-character alphabet is
just a codec at the key-construction boundary). Reserves code 0 (guide
sentinel) and code 1 (PayloadSeparator) so dense-coded bytes can never
collide with either. No production wiring - see
docs/superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md."
```

---

## Task 2: Comparison harness

**Files:**
- Create: `pkg/morphology/internal/alphabet_compare_integration_test.go`

**Interfaces:**
- Consumes: `Alphabet`, `IdentityAlphabet`, `DenseAlphabet`, `NewDenseAlphabet` (Task 1); `BuildDAWG(keys []string) (*DAWG, error)` (existing, `dawgbuild.go:33`); `PayloadSeparator` (existing, `dawg.go:23`).

- [ ] **Step 1: Write the harness**

Create `pkg/morphology/internal/alphabet_compare_integration_test.go`:

```go
//go:build integration

// Compares DAWG size and build time across alphabets (identity, dense-1,
// dense-2) on real OpenCorpora wordforms with realistic payload-bearing
// keys (word + PayloadSeparator + base64(uint32) - the same shape
// BuildDAWGWithValues produces, see dawgbuild.go). This is a measurement
// report, not a pass/fail test: read its -v output for the numbers.
//
// Needs a real wordform list. Generate one with:
//
//	grep -o '<f t="[^"]*"' .data/opencorpora/dict.xml | sed -E 's/<f t="//;s/"$//' | sort -u > /tmp/wordforms.txt
//
// Then run:
//
//	go test -tags=integration ./pkg/morphology/internal/ -run TestAlphabetCompare -v
//
// A full-corpus run (~3M words) takes minutes per alphabet; the default
// sample (100,000 words, evenly strided) is fast enough for a routine
// check. Override with GOMORPHY_ALPHABET_BENCH_SAMPLE (0 or >= corpus size
// means "use everything").
package internal

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

const (
	wordlistEnvVar      = "GOMORPHY_ALPHABET_BENCH_WORDLIST"
	defaultWordlistPath = "/tmp/wordforms.txt"
	sampleEnvVar        = "GOMORPHY_ALPHABET_BENCH_SAMPLE"
	defaultSample       = 100000
)

func TestAlphabetCompare(t *testing.T) {
	path := os.Getenv(wordlistEnvVar)
	if path == "" {
		path = defaultWordlistPath
	}
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("wordlist not found at %s (%s to override) - see this file's doc comment for the extraction command", path, wordlistEnvVar)
	}
	defer func() { _ = f.Close() }()

	var all []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		w := sc.Text()
		if w != "" {
			all = append(all, w)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded %d words from %s", len(all), path)

	sample := defaultSample
	if v := os.Getenv(sampleEnvVar); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("%s=%q is not a valid integer: %v", sampleEnvVar, v, err)
		}
		sample = n
	}
	stride := 1
	if sample > 0 && sample < len(all) {
		stride = len(all) / sample
	}
	var words []string
	for i := 0; i < len(all); i += stride {
		words = append(words, all[i])
	}
	t.Logf("using %d words (stride %d; override sample size with %s)", len(words), stride, sampleEnvVar)

	// Random (not sequential) payload values - this session's spike found
	// sequential big-endian values pathological at scale, plausibly
	// because they give alphabetically-adjacent words (which already
	// share trie prefixes) near-identical payload suffixes too.
	rng := rand.New(rand.NewSource(1))
	values := make([]uint32, len(words))
	for i := range values {
		values[i] = rng.Uint32()
	}

	type result struct {
		name  string
		slots int
		bytes int
		build time.Duration
	}
	var results []result

	run := func(a Alphabet) {
		keys := make([]string, len(words))
		sep := string([]byte{PayloadSeparator})
		for i, w := range words {
			enc, err := a.Encode(w)
			if err != nil {
				t.Fatalf("%s: encode %q: %v", a.Name(), w, err)
			}
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, values[i])
			keys[i] = string(enc) + sep + base64.StdEncoding.EncodeToString(b)
		}

		t.Logf("building DAWG for alphabet %q (%d keys)...", a.Name(), len(keys))
		start := time.Now()
		dawg, err := BuildDAWG(keys)
		if err != nil {
			t.Fatalf("%s: build: %v", a.Name(), err)
		}
		elapsed := time.Since(start)

		results = append(results, result{
			name:  a.Name(),
			slots: len(dawg.dict),
			bytes: len(dawg.dict) * 4,
			build: elapsed,
		})
	}

	run(IdentityAlphabet{})

	dense1, err := NewDenseAlphabet(1, words)
	if err != nil {
		t.Fatalf("NewDenseAlphabet(1, ...): %v", err)
	}
	run(dense1)

	dense2, err := NewDenseAlphabet(2, words)
	if err != nil {
		t.Fatalf("NewDenseAlphabet(2, ...): %v", err)
	}
	run(dense2)

	t.Log("=== results ===")
	baseline := results[0].bytes
	for _, r := range results {
		reduction := 100 * (1 - float64(r.bytes)/float64(baseline))
		t.Logf("%-10s slots=%9d bytes=%10d (%.2f MB) build=%-12s reduction=%.1f%%",
			r.name, r.slots, r.bytes, float64(r.bytes)/1e6, r.build, reduction)
	}
}
```

- [ ] **Step 2: Verify the wordlist exists, generating it if needed**

```bash
ls -la /tmp/wordforms.txt 2>/dev/null || \
  (grep -o '<f t="[^"]*"' .data/opencorpora/dict.xml | sed -E 's/<f t="//;s/"$//' | sort -u > /tmp/wordforms.txt)
wc -l /tmp/wordforms.txt
```
Expected: 3,065,312 lines (matches `docs/research/0001-dawg-alphabet-density.md`'s corpus size — if it doesn't match, the real `dict.xml` may have changed since that research; note the discrepancy in your report but proceed).

- [ ] **Step 3: Run the harness at the default sample size (100,000 words) to verify it works end-to-end**

Run: `go test -tags=integration ./pkg/morphology/internal/ -run TestAlphabetCompare -v -timeout 600s`
Expected: PASS, with a results table logged for all three alphabets (`identity`, `dense-1`, `dense-2`). Sanity-check the numbers make sense: `dense-1`'s `bytes` should be noticeably smaller than `identity`'s (matching the ~36% effect from research 0001 and this session's spike); `dense-2` should land somewhere between `dense-1` and `identity` (2 bytes/character is worse than 1, but the dense code space is still more uniformly used than UTF-8's structured/sparse byte patterns) — if `dense-2` comes out *larger* than `identity` or *smaller* than `dense-1`, treat that as a possible implementation bug, not just an interesting result, and investigate before reporting it as a finding.

- [ ] **Step 4: Run `go build`/`go vet` with the integration tag to confirm the new file is well-formed**

```bash
go build -tags=integration ./...
go vet -tags=integration ./pkg/morphology/internal/...
```
Expected: both clean.

- [ ] **Step 5: Run the full existing test suite (without the integration tag) to confirm no regressions**

Run: `go test ./pkg/morphology/... -race`
Expected: PASS — the new file is excluded by its build tag, so this run doesn't even compile it; this just confirms Task 1's additions still behave.

- [ ] **Step 6: Record the sample-run numbers in the task report**

No code change — just capture the actual `t.Logf` results table from Step 3 in your report file (see below), so the controller has concrete numbers without re-running the harness.

- [ ] **Step 7: Commit**

```bash
git add pkg/morphology/internal/alphabet_compare_integration_test.go
git commit -m "internal: add DAWG dense-alphabet comparison harness

Measures DAWG slot count, byte size, and build time for identity,
dense-1, and dense-2 alphabets on real OpenCorpora wordforms with
realistic payload-bearing keys (random values, not sequential - see
docs/superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md for
why). No production wiring; this is a repeatable measurement tool for
a later decision, not a shipped feature."
```
