package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParadigmIndexArithmetic(t *testing.T) {
	p := NewParadigm(
		[]uint16{10, 20, 30},
		[]uint16{1, 2, 3},
		[]uint16{0, 0, 0},
	)

	require.Equal(t, 3, p.Len())

	assert.Equal(t, uint16(10), p.Suffix(0))
	assert.Equal(t, uint16(20), p.Suffix(1))
	assert.Equal(t, uint16(30), p.Suffix(2))

	assert.Equal(t, uint16(1), p.Tag(0))
	assert.Equal(t, uint16(2), p.Tag(1))
	assert.Equal(t, uint16(3), p.Tag(2))

	assert.Equal(t, uint16(0), p.Prefix(0))

	assert.Equal(t, uint16(30), p.Suffix(2))
	assert.Equal(t, uint16(0), p.Prefix(2))
}

func TestParadigmEmpty(t *testing.T) {
	p := NewParadigm(nil, nil, nil)
	assert.Equal(t, 0, p.Len())
}

func TestParadigmPrefixesKeptSeparately(t *testing.T) {
	p := NewParadigm(
		[]uint16{5, 6, 7},
		[]uint16{0, 1, 0},
		[]uint16{2, 0, 2},
	)
	require.Equal(t, 3, p.Len())
	assert.Equal(t, uint16(2), p.Prefix(0))
	assert.Equal(t, uint16(0), p.Prefix(1))
	assert.Equal(t, uint16(2), p.Prefix(2))
	assert.Equal(t, uint16(5), p.Suffix(0))
}

func TestParadigmFromData(t *testing.T) {
	p, err := NewParadigmFromData([]uint16{10, 20, 1, 2, 0, 0})
	require.NoError(t, err)
	assert.Equal(t, 2, p.Len())
	assert.Equal(t, uint16(10), p.Suffix(0))
	assert.Equal(t, uint16(20), p.Suffix(1))
	assert.Equal(t, uint16(1), p.Tag(0))
	assert.Equal(t, uint16(2), p.Tag(1))
	assert.Equal(t, uint16(0), p.Prefix(1))

	_, err = NewParadigmFromData([]uint16{1, 2, 3, 4})
	require.Error(t, err, "длина обязана делиться на 3")
}

func TestParadigmData(t *testing.T) {
	data := []uint16{10, 20, 1, 2, 0, 0}
	p, err := NewParadigmFromData(data)
	require.NoError(t, err)
	assert.Equal(t, data, p.Data())
}
