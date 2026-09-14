package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagSetAdd(t *testing.T) {
	ts := NewTagSet("opencorpora")

	id1, err := ts.Add("NOUN")
	require.NoError(t, err)
	id2, err := ts.Add("NOUN")
	require.NoError(t, err)

	require.Equal(t, uint16(0), id1)
	assert.Equal(t, id1, id2, "добавление существующего тега возвращает тот же id")
	assert.Equal(t, []string{"NOUN"}, ts.Tags)
	assert.Equal(t, 1, len(ts.Index))
}

func TestTagSetLookup(t *testing.T) {
	ts := NewTagSet("x")
	_, err := ts.Add("NOUN")
	require.NoError(t, err)
	_, err = ts.Add("anim")
	require.NoError(t, err)

	id, ok := ts.ID("anim")
	assert.True(t, ok)
	assert.Equal(t, uint16(1), id)

	_, ok = ts.ID("nope")
	assert.False(t, ok)

	assert.Equal(t, "NOUN", ts.TagName(0))
	assert.Equal(t, "", ts.TagName(99))
}

func TestTagSetIDsAreMonotonic(t *testing.T) {
	ts := NewTagSet("x")
	for i := 0; i < 300; i++ {
		name := "tag_" + string(rune('a'+i%26)) + "n" + string(rune('0'+i/26))
		id, err := ts.Add(name)
		require.NoError(t, err)
		assert.Equal(t, uint16(i), id)
	}
}

func TestTagSetAddOverflow(t *testing.T) {
	ts := &TagSet{Name: "x", Index: make(map[string]uint16), Tags: make([]string, 1<<16)}

	_, err := ts.Add("one-too-many")
	require.ErrorIs(t, err, ErrTagSetFull)
}
