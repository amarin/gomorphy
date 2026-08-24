package stringsx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/stringsx"
)

func TestArena_AppendGet(t *testing.T) {
	a := stringsx.New(64, 4)

	id1 := a.Append([]byte("ёжик"))
	id2 := a.Append([]byte("в тумане"))
	id3 := a.Append([]byte(""))
	id4 := a.Append([]byte("multi байт слово"))

	assert.Equal(t, uint32(0), id1)
	assert.Equal(t, uint32(1), id2)
	assert.Equal(t, uint32(2), id3)
	assert.Equal(t, uint32(3), id4)

	assert.Equal(t, []byte("ёжик"), a.Get(id1))
	assert.Equal(t, []byte("в тумане"), a.Get(id2))
	assert.Empty(t, a.Get(id3))
	assert.Equal(t, []byte("multi байт слово"), a.Get(id4))

	assert.Equal(t, 4, a.Len())
}

func TestArena_LargeAppend(t *testing.T) {
	a := stringsx.New(8, 2)

	big := make([]byte, 1<<20)
	for i := range big {
		big[i] = byte(i)
	}

	id := a.Append(big)
	require.Equal(t, big, a.Get(id))
}

func TestArena_ManyStrings(t *testing.T) {
	a := stringsx.New(1024, 1024)

	const total = 10000
	want := make([]string, total)

	for i := range total {
		want[i] = "s" + string(rune('a'+i%26)) + string(rune('0'+i%10))
		a.Append([]byte(want[i]))
	}

	assert.Equal(t, total, a.Len())

	for i := range total {
		assert.Equal(t, want[i], string(a.Get(uint32(i))), "i=%d", i)
	}
}

func BenchmarkArena_Append(b *testing.B) {
	a := stringsx.New(1<<26, 1<<21)
	word := []byte("словоформочка")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Append(word) // nolint:errcheck
	}
}
