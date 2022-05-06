package index_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/index"
)

func TestNode_Add(t *testing.T) {
	idx := index.New()
	_, err := idx.AddString("test")
	require.NoError(t, err)
	// require.Equal(t, 4, int(node1.ID()))
}
