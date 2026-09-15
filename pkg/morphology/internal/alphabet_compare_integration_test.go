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
