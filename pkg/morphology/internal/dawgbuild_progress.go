package internal

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"sort"
)

// BuildDAWGWithValuesProgress builds a DAWG with progress reporting.
// progress is called periodically with (processed, total) — total=len(keys).
func BuildDAWGWithValuesProgress(keys []string, values []uint32, progress func(processed, total int)) (*DAWG, error) {
	if len(keys) != len(values) {
		return nil, fmt.Errorf("dawg: keys and values must have same length")
	}
	total := len(keys)

	// Phase 1: build payload keys.
	payloadKeys := make([]string, total)
	for i, k := range keys {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, values[i])
		payloadKeys[i] = k + string([]byte{PayloadSeparator}) + base64.StdEncoding.EncodeToString(b)
	}

	// Phase 2: sort — once, no progress (the user sees this as "preparing").
	sort.Strings(payloadKeys)

	// Phase 3: build list-form DAWG (insertion + merging).
	// This is the slowest phase — report every 1% of keys.
	b := newDawgBuilder()

	// Progress every 1% of keys, but at least every 1000 keys.
	progressEvery := total / 100
	if progressEvery < 1000 {
		progressEvery = 1000
	}

	b.insertKeys(payloadKeys, func(i int) {
		if progress != nil && (i+1)%progressEvery == 0 {
			progress(i+1, total)
		}
	})

	// Phase 4: compile to double-array (DFS) with progress.
	// Progress callback uses same total (keys count) for consistency.
	result, err := b.compileWithTotal(total, progress)
	if err != nil {
		return nil, fmt.Errorf("dawg: compile: %w", err)
	}
	return result, nil
}
