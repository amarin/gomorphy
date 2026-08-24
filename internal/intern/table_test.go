package intern_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/intern"
	"github.com/amarin/gomorphy/internal/stringsx"
)

func newTable(expected int) *intern.Table {
	return intern.New(stringsx.New(1<<20, expected), expected)
}

func TestIntern_DuplicatesAndUniques(t *testing.T) {
	tb := newTable(16)

	first, existed := tb.Intern([]byte("ёжик"))
	assert.False(t, existed)

	second, existed := tb.Intern([]byte("ёжик"))
	assert.True(t, existed)
	assert.Equal(t, first, second)

	third, existed := tb.Intern([]byte("ёжик "))
	assert.False(t, existed)
	assert.NotEqual(t, first, third)

	fourth, existed := tb.Intern([]byte(""))
	assert.False(t, existed)

	again, existed := tb.Intern(nil)
	assert.True(t, existed)
	assert.Equal(t, fourth, again)

	assert.Equal(t, 3, tb.Len())
}

func TestIntern_GetRoundtrip(t *testing.T) {
	tb := newTable(1024)

	words := []string{"кот", "код", "крот", "ёж", "ёжик", "", "длинное-предлинное-слово"}
	ids := make([]uint32, len(words))

	for i, w := range words {
		id, existed := tb.Intern([]byte(w))
		assert.False(t, existed, "word %q", w)
		ids[i] = id
	}

	for i, w := range words {
		assert.Equal(t, []byte(w), tb.Get(ids[i]), "get %d %q", i, w)
	}
}

func TestIntern_Rehash(t *testing.T) {
	tb := newTable(4)

	const total = 50000
	ids := make([]uint32, total)

	for i := range total {
		w := fmt.Sprintf("слово-%d", i)

		id, existed := tb.Intern([]byte(w))
		require.False(t, existed, "i=%d", i)
		ids[i] = id
	}

	assert.Equal(t, total, tb.Len())

	for i := range total {
		w := fmt.Sprintf("слово-%d", i)

		id, existed := tb.Intern([]byte(w))
		assert.True(t, existed, "i=%d", i)
		assert.Equal(t, ids[i], id, "stable id i=%d", i)
		assert.Equal(t, []byte(w), tb.Get(ids[i]))
	}
}

func TestIntern_SimilarWordsDistinct(t *testing.T) {
	tb := newTable(128)

	pairs := [][2]string{
		{"стекла", "стекала"},
		{"лук", "люк"},
		{"a", "ab"},
		{"ab", "ba"},
	}

	for _, p := range pairs {
		id1, _ := tb.Intern([]byte(p[0]))
		id2, _ := tb.Intern([]byte(p[1]))
		assert.NotEqual(t, id1, id2, "%v", p)
	}
}

func benchmarkStrings(n int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		out[i] = []byte(fmt.Sprintf("интернирование-строки-%d", i))
	}

	return out
}

func BenchmarkIntern_Hits_1M(b *testing.B) {
	const total = 1_000_000

	words := benchmarkStrings(total)
	tb := newTable(total)

	for _, w := range words {
		tb.Intern(w) // nolint:errcheck
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, ok := tb.Intern(words[i%total]); !ok {
			b.Fatal("expected existing")
		}
	}
}

func BenchmarkIntern_Unique_1M(b *testing.B) {
	words := benchmarkStrings(1_000_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb := newTable(1_000_000)

		for _, w := range words {
			if _, existed := tb.Intern(w); existed {
				b.Fatal("unexpected duplicate")
			}
		}

		sink = tb
	}
}

var sink *intern.Table
