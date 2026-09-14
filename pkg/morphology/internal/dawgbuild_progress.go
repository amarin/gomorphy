package internal

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"sort"
)

// BuildDAWGWithValuesProgress строит DAWG с прогресс-отчётами.
// progress вызывается periodically с (processed, total) — total=len(keys).
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

	// Phase 2: sort — один раз, без progress (пользователь видит это как "подготовка").
	sort.Strings(payloadKeys)

	// Phase 3: build list-form DAWG (insertion + merging).
	// Это самая медленная фаза — отчёт каждые 1% ключей.
	b := &dawgBuilder{
		register:  make(map[string]int32, 1<<20),
		merged:    make([]bool, 1),
		path:      make([]int32, 0, 32),
		sigBuf:    make([]byte, 0, 64),
		labelsBuf: make([]byte, 0, 16),
	}
	b.root = b.newNode(0)
	b.path = append(b.path, b.root)

	// Progress every 1% of keys, but at least every 1000 keys.
	progressEvery := total / 100
	if progressEvery < 1000 {
		progressEvery = 1000
	}

	for i, k := range payloadKeys {
		common := 0
		for common < len(b.lastKey) && common < len(k) && b.lastKey[common] == k[common] {
			common++
		}
		b.closeSuffix(common)
		for j := common; j < len(k); j++ {
			b.appendByte(k[j])
		}
		b.nodes[b.path[len(b.path)-1]].leaf = true
		b.lastKey = k

		if progress != nil && (i+1)%progressEvery == 0 {
			progress(i+1, total)
		}
	}
	b.closeSuffix(0)

	// Phase 4: compile to double-array (DFS) with progress.
	// Progress callback uses same total (keys count) for consistency.
	result, err := b.compileWithTotal(total, progress)
	if err != nil {
		return nil, fmt.Errorf("dawg: compile: %w", err)
	}
	return result, nil
}
