package internal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
)

// payloadDAWGGolden is the SHA-256 of BuildDAWGWithValues over
// goldenPayloadKeys, captured before leaf values were added to the builder:
// words/prediction DAWG bytes must not change.
const payloadDAWGGolden = "f78753b9f92665c6cd58b1d2a6885afea7672dc90c075b41ca5c9499abfbecbf"

func goldenPayloadKeys() ([]string, []uint32) {
	r := rand.New(rand.NewPCG(7, 11))
	letters := []rune("абвгдеёжзик")
	seen := map[string]bool{}
	var keys []string
	var vals []uint32
	for len(keys) < 5000 {
		n := 1 + r.IntN(7)
		rs := make([]rune, n)
		for i := range rs {
			rs[i] = letters[r.IntN(len(letters))]
		}
		k := string(rs)
		if seen[k] {
			continue
		}
		seen[k] = true
		keys = append(keys, k)
		vals = append(vals, r.Uint32())
	}
	return keys, vals
}

func TestPayloadDAWGBytesStable(t *testing.T) {
	keys, vals := goldenPayloadKeys()
	d, err := BuildDAWGWithValues(keys, vals)
	require.NoError(t, err)
	sum := sha256.Sum256(d.Bytes())
	got := hex.EncodeToString(sum[:])
	t.Logf("payload DAWG sha256 = %s", got)
	assert.Equal(t, payloadDAWGGolden, got)
}

func sortedKV(m map[string]uint32) ([]string, []uint32) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]uint32, len(keys))
	for i, k := range keys {
		vals[i] = m[k]
	}
	return keys, vals
}

func TestBuildIntDAWGFindAndWalk(t *testing.T) {
	want := map[string]uint32{
		"кот:NOUN": 500, "кот:VERB": 100, "кота:NOUN": 500,
		"ab": 1, "cb": 2, "abc": 3, "b": 0, // shared suffixes with different values
	}
	keys, vals := sortedKV(want)
	d, err := BuildIntDAWG(keys, vals)
	require.NoError(t, err)

	for k, v := range want {
		assert.True(t, d.Contains(k), k)
		assert.Equal(t, v, d.Find(k), k)
	}
	assert.False(t, d.Contains("ко"))

	got := map[string]uint32{}
	d.WalkValues(func(k string, v uint32) { got[k] = v })
	assert.Equal(t, want, got)

	d2, err := ParseDAWG(d.Bytes())
	require.NoError(t, err)
	assert.Equal(t, uint32(500), d2.Find("кот:NOUN"))
}

func TestBuildIntDAWGRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	letters := []rune("абвгд:")
	want := map[string]uint32{}
	for len(want) < 3000 {
		n := 1 + r.IntN(6)
		rs := make([]rune, n)
		for i := range rs {
			rs[i] = letters[r.IntN(len(letters))]
		}
		want[string(rs)] = r.Uint32N(1_000_000)
	}
	keys, vals := sortedKV(want)
	// Unsorted input must be accepted too.
	r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i]; vals[i], vals[j] = vals[j], vals[i] })

	d, err := BuildIntDAWG(keys, vals)
	require.NoError(t, err)
	for k, v := range want {
		require.Equal(t, v, d.Find(k), k)
	}
	n := 0
	d.WalkValues(func(string, uint32) { n++ })
	assert.Equal(t, len(want), n)
}

func TestBuildIntDAWGErrors(t *testing.T) {
	_, err := BuildIntDAWG([]string{"a"}, nil)
	assert.Error(t, err, "length mismatch")
	_, err = BuildIntDAWG([]string{"a", "a"}, []uint32{1, 2})
	assert.Error(t, err, "duplicate key")
	_, err = BuildIntDAWG([]string{"a"}, []uint32{1 << 31})
	assert.Error(t, err, "value out of range")
	_, err = BuildIntDAWG([]string{""}, []uint32{1})
	assert.Error(t, err, "empty key")
}

func TestWalkValuesPymorphyIntDAWG(t *testing.T) {
	want := map[string]uint32{"кот:NOUN": 500, "кота:NOUN": 7}
	dict, guide := testdawg.Build(want)
	d, err := ReadDAWG(bytes.NewReader(testdawg.Marshal(dict, guide)))
	require.NoError(t, err)
	got := map[string]uint32{}
	d.WalkValues(func(k string, v uint32) { got[k] = v })
	assert.Equal(t, want, got)
}

func TestDAWGCloneAndEmpty(t *testing.T) {
	d, err := BuildDAWGWithValues([]string{"кот"}, []uint32{7})
	require.NoError(t, err)
	c := d.Clone()
	assert.Equal(t, d.Bytes(), c.Bytes())
	c.dict[0] ^= 1
	assert.NotEqual(t, d.Bytes(), c.Bytes(), "clone must not alias the original")
	assert.False(t, d.Empty())

	e, err := BuildDAWG(nil)
	require.NoError(t, err)
	assert.True(t, e.Empty())

	var nilDAWG *DAWG
	assert.Nil(t, nilDAWG.Clone())
	assert.True(t, nilDAWG.Empty())
}
