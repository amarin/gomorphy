//go:build scaling

package internal

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestDAWGBuildScales checks that build time grows sub-quadratically
// (not quadratically) with the number of keys — the very property the
// placement algorithm was replaced for. Requires the "scaling" build tag
// and is not part of `make test`/the fast CI run (like the integration
// tests — see `-tags integration` in the Makefile): run it explicitly
// (`go test -tags scaling -run TestDAWGBuildScales -v -timeout 30m ./...`),
// since at the upper bound (5M) it takes several minutes and several
// gigabytes of RSS even with non-catastrophic (non-O(n^2)) scaling.
func TestDAWGBuildScales(t *testing.T) {

	sizes := []int{100_000, 500_000, 1_000_000, 2_000_000, 5_000_000}
	var perKey []float64

	for _, n := range sizes {
		keys, values := randomWordformKeys(n, 12345)

		start := time.Now()
		d, err := BuildDAWGWithValues(keys, values)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("N=%d: BuildDAWGWithValues: %v", n, err)
		}
		if d == nil {
			t.Fatalf("N=%d: nil DAWG", n)
		}

		perN := elapsed.Seconds() / float64(n)
		perKey = append(perKey, perN)
		t.Logf("N=%9d  elapsed=%-12s  per-key=%.3fµs  dic-len=%d",
			n, elapsed, perN*1e6, len(d.dict))
	}

	// Quadratic growth (the old findBaseBitset) would give a SHARP rise in
	// per-key cost: O(n^2) from N=100K to N=5M (50x) would give >2500x,
	// orders of magnitude worse than observed. With the free list, growth
	// is sub-quadratic (~n^1.3-1.4), not perfectly flat: this key
	// generator produces almost-random suffixes + a unique payload per
	// key, i.e. almost no shared suffixes, unlike real wordforms
	// (thousands of words end in "-ами", "-ов", etc.). Because of this the
	// dic array is heavily inflated relative to N (already ~4.9M slots for
	// 100K keys) and quickly moves into the range where a dictionary
	// unit's extended-offset field (see encodable()) accepts only ~1 in
	// 256 candidates — a property of the dawgdic format itself, not an
	// allocator defect. The threshold below is deliberately lower than
	// what an actual regression to quadratic behavior would produce, but
	// higher than typical GC/allocation noise.
	const maxAcceptableGrowth = 8.0
	for i := 1; i < len(perKey); i++ {
		ratio := perKey[i] / perKey[0]
		if ratio > maxAcceptableGrowth {
			t.Errorf("per-key cost grew %.1fx from N=%d to N=%d (%.3fµs -> %.3fµs); "+
				"expected sub-quadratic scaling (<%.0fx), got growth consistent with a regression",
				ratio, sizes[0], sizes[i], perKey[0]*1e6, perKey[i]*1e6, maxAcceptableGrowth)
		}
	}
}

// randomWordformKeys generates N deterministic (seeded) keys of the form
// "word\x01payload" — with length and alphabet resembling real
// wordforms (Cyrillic, 4-12 characters) + a 4-byte payload, as in
// BuildDAWGWithValues.
func randomWordformKeys(n int, seed int64) ([]string, []uint32) {
	rnd := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic test data
	alphabet := []rune("абвгдежзийклмнопрстуфхцчшщъыьэюя")

	keys := make([]string, n)
	values := make([]uint32, n)
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		var w []rune
		length := 4 + rnd.Intn(9) // 4..12
		for j := 0; j < length; j++ {
			w = append(w, alphabet[rnd.Intn(len(alphabet))])
		}
		key := fmt.Sprintf("%s-%d", string(w), i) // suffix guarantees uniqueness
		for seen[key] {
			key = key + "x"
		}
		seen[key] = true
		keys[i] = key
		values[i] = uint32(i)
	}
	return keys, values
}
