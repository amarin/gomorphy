package index_test

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
)

func TestNewItems(t *testing.T) {
	items := index.NewItems()
	require.Equal(t, int(items.NextID()), 1)
}

func TestItems_NewChild(t *testing.T) {
	maxThreads := 25
	items := index.NewItems()
	letters := "abcdefjhijklmnopqrstuvwxyz"

	randomRune := func() rune {
		return rune(letters[rand.Intn(len(letters)-1)])
	}

	for i := 0; i < maxThreads; i++ {
		t.Run("iter_"+strconv.Itoa(i), func(t *testing.T) {
			require.Equal(t, i+1, int(items.NextID()))

			parentID := dag.ID(i)
			runeToUse := randomRune()
			newItem := items.NewChild(parentID, runeToUse)

			require.Equal(t, newItem.Parent, parentID)
			require.Equal(t, int(newItem.ID), i+1)
			require.Equal(t, int(newItem.Variants), 0)
			require.Equal(t, newItem.Letter, runeToUse)

			getItem := items.Get(newItem.ID)
			require.Equal(t, getItem.Parent, parentID)
			require.Equal(t, int(getItem.ID), i+1)
			require.Equal(t, int(getItem.Variants), 0)
			require.Equal(t, getItem.Letter, runeToUse)

			require.Equal(t, i+2, int(items.NextID()))
		})

	}

	require.Equal(t, maxThreads, int(items.NextID())-1)
}
