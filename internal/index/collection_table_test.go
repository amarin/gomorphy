package index

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectionTable_Get_0(t *testing.T) {
	emptyCollectionTable := make(CollectionTable, 0)
	res := emptyCollectionTable.Get(0)
	t.Run("collection_table_0_returns_non_nil_collection_table", func(t *testing.T) {
		require.NotNil(t, res)
	})
	t.Run("collection_table_0_returns_empty_collection_table", func(t *testing.T) {
		require.Len(t, res, 0)
	})
}
