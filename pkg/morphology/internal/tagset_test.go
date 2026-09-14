package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagSetAdd(t *testing.T) {
	ts := NewTagSet("opencorpora")

	id1 := ts.Add("NOUN")
	id2 := ts.Add("NOUN")

	require.Equal(t, uint16(0), id1)
	assert.Equal(t, id1, id2, "добавление существующего тега возвращает тот же id")
	assert.Equal(t, []string{"NOUN"}, ts.Tags)
	assert.Equal(t, 1, len(ts.Index))
}

func TestTagSetLookup(t *testing.T) {
	ts := NewTagSet("x")
	ts.Add("NOUN")
	ts.Add("anim")

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
		assert.Equal(t, uint16(i), ts.Add(name))
	}
}
