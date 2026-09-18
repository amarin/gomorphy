package internal

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDAWGContains(t *testing.T) {
	keys := []string{"кот\x01AAAA", "кота\x01AAAB", "коту\x01AAAC", "дом\x01AAAA"}

	d, err := BuildDAWG(keys)
	require.NoError(t, err)
	require.NotNil(t, d)

	for _, k := range keys {
		assert.True(t, d.Contains(k), "key %q must be found", k)
	}

	for _, k := range []string{"", "ко", "кот", "кот\x01", "кот\x01AAA", "кот\x01AAAAA", "дома", "дом\x01AAAB\x01"} {
		assert.False(t, d.Contains(k), "non-key %q must not be found", k)
	}
}

func TestBuildDAWGTerminalWithChildren(t *testing.T) {
	d, err := BuildDAWG([]string{"аб", "абв", "г"})
	require.NoError(t, err)

	assert.True(t, d.Contains("аб"))
	assert.True(t, d.Contains("абв"))
	assert.True(t, d.Contains("г"))
	assert.False(t, d.Contains("абг"))
	assert.False(t, d.Contains("а"))
}

func TestBuildDAWGRoundtrip(t *testing.T) {
	keys := []string{"кот\x01AAAA", "кота\x01AAAB", "коту\x01AAAC", "дом\x01AAAA", "дома\x01AAAB"}

	d, err := BuildDAWG(keys)
	require.NoError(t, err)

	data := d.Bytes()
	parsed, err := ParseDAWG(data)
	require.NoError(t, err)

	for _, k := range keys {
		assert.True(t, parsed.Contains(k), "after roundtrip key %q must be found", k)
	}
	for _, k := range []string{"кот", "дом\x01AAAC", "ёж"} {
		assert.False(t, parsed.Contains(k), "after roundtrip non-key %q must not be found", k)
	}
}

// TestBuildDAWGSimilarItems verifies the parsing path: a dictionary built
// by the builder is readable by the existing SimilarItems+ValuesForIndex
// mechanism (payload in base64 after the 0x01 separator).
func TestBuildDAWGSimilarItems(t *testing.T) {
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
	}
	d, err := BuildDAWG(keys)
	require.NoError(t, err)

	items := d.SimilarItems("кот", RussianCharPolicy(), nil)
	require.Len(t, items, 1)
	assert.Equal(t, "кот", items[0].Key)
	assert.Len(t, items[0].Values, 2, "ожидались два чтения омонима")
	assert.Equal(t, []byte{0, 0, 0, 0}, items[0].Values[0])

	// The е/ё substitution works for the built dictionary: "ежик" finds "ёжик".
	items = d.SimilarItems("ежик", RussianCharPolicy(), nil)
	require.Len(t, items, 1)
	assert.Equal(t, "ёжик", items[0].Key)
	// payload(2, 0) = {0, 2, 0, 0} — para=2, form=0 in big-endian.
	assert.Equal(t, []byte{0, 2, 0, 0}, items[0].Values[0])
}

// TestBuildDAWGExhaustiveAcceptance exhaustively checks the strings accepted
// by the automaton against a reference set (small alphabet → strong check).
func TestBuildDAWGExhaustiveAcceptance(t *testing.T) {
	alphabet := []byte("ab")
	all := func(maxLen int) []string {
		var out []string
		var walk func(string)
		walk = func(p string) {
			if len(p) <= maxLen {
				out = append(out, p)
			}
			if len(p) == maxLen {
				return
			}
			for _, c := range alphabet {
				walk(p + string(c))
			}
		}
		walk("")
		return out
	}

	// A random subset of strings of length 1..3.
	rnd := rand.New(rand.NewSource(7)) //nolint:gosec // deterministic test
	reference := make(map[string]bool)
	var keys []string
	for _, s := range all(3) {
		if s == "" {
			continue
		}
		if rnd.Intn(2) == 0 {
			reference[s] = true
			keys = append(keys, s)
		}
	}
	require.NotEmpty(t, keys)

	d, err := BuildDAWG(keys)
	require.NoError(t, err)

	for _, s := range all(5) {
		assert.Equal(t, reference[s], d.Contains(s), "строка %q", s)
	}
}

func TestBuildDAWGDuplicates(t *testing.T) {
	keys := []string{"кот\x01AAAA", "кот\x01AAAA", "кот\x01AAAA", "до\x01AAAB", "до\x01AAAB"}

	d, err := BuildDAWG(keys)
	require.NoError(t, err)

	for _, k := range keys {
		assert.True(t, d.Contains(k), "key %q must be found", k)
	}
	assert.False(t, d.Contains("кот"))
}

// TestBuildDAWGMinimizesSharedSuffixes is a regression test for a bug in
// chainSig: the sibling-chain signature erroneously encoded the id of the
// node itself instead of the id of its child, so register never found
// matches and suffixes were never merged (the DAWG degenerated into an
// unminimized trie — on the real OpenCorpora dictionary this bloated the
// double-array from ~2M to ~71M slots). Many keys with different prefixes
// and one shared long suffix: without suffix merging, the double-array
// would need a separate copy of the suffix chain for every key.
func TestBuildDAWGMinimizesSharedSuffixes(t *testing.T) {
	const n = 200
	const suffix = "-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = string([]byte{byte('a' + i%26), byte('0' + i%10), byte('A' + i%26)}) + suffix
	}

	d, err := BuildDAWG(keys)
	require.NoError(t, err)
	for _, k := range keys {
		require.True(t, d.Contains(k), "key %q must be found", k)
	}

	// Without suffix merging this would need ~n*len(suffix) slots (the
	// shared tail would be copied for each key). With correct minimization
	// the tail is one shared chain, and the array size should stay an
	// order of magnitude below this bound.
	unminimizedFloor := n * len(suffix)
	assert.Less(t, len(d.dict), unminimizedFloor/4,
		"dictionary array has %d slots for %d keys sharing a %d-byte suffix — suffix minimization appears broken",
		len(d.dict), n, len(suffix))
}

func TestBuildDAWGEmpty(t *testing.T) {
	d, err := BuildDAWG(nil)
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.False(t, d.Contains(""))
	assert.False(t, d.Contains("кот"))
}

// TestBuildDAWGLargeScalePlacement stresses the placer (compileImpl split
// out during the pre-1.0 review, docs/en/code-review-pre-1.0.md) with far more
// nodes and shared-suffix merges than the small fixtures elsewhere in this
// file exercise, since node placement / base reuse is exactly the logic
// that refactor touched.
func TestBuildDAWGLargeScalePlacement(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const n = 20000

	seen := make(map[string]bool, n)
	keys := make([]string, 0, n)
	alphabet := "абвгдежзийклмнопрстуфхцчшщъыьэюя"
	runes := []rune(alphabet)

	for len(keys) < n {
		length := 3 + rng.Intn(6)
		b := make([]rune, length)
		for i := range b {
			b[i] = runes[rng.Intn(len(runes))]
		}
		k := string(b)
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}

	d, err := BuildDAWG(append([]string{}, keys...))
	require.NoError(t, err)

	for _, k := range keys {
		assert.True(t, d.Contains(k), "missing key %q", k)
	}

	missing := 0
	for i := 0; i < 2000; i++ {
		k := keys[rng.Intn(len(keys))] + "\x00notakey"
		if !seen[k] && d.Contains(k) {
			missing++
		}
	}
	assert.Zero(t, missing, "DAWG accepted keys that were never inserted")
}

// TestBuildDAWGManyKeysAndSortingStress runs a larger set of keys
// (including ones with shared prefixes and suffixes) and checks acceptance.
func TestBuildDAWGManyKeysAndSortingStress(t *testing.T) {
	var keys []string
	for _, w := range []string{"кот", "кода", "коды", "коду", "дом", "дома", "дому", "ёж", "ежа", "овёс", "овса"} {
		for i := 0; i < 3; i++ {
			keys = append(keys, w+string(PayloadSeparator)+string(bytes.Repeat([]byte{'A' + byte(i)}, 8)))
		}
	}
	sort.Strings(keys)

	d, err := BuildDAWG(keys)
	require.NoError(t, err)
	for _, k := range keys {
		assert.True(t, d.Contains(k))
	}
	for _, k := range []string{"кот", "домашний\x01AAAA", "ёж\x01AAAA"} {
		assert.False(t, d.Contains(k))
	}
}
