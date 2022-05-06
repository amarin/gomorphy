package index_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
)

func TestNewIndex(t *testing.T) {
	idx := index.New()
	require.Equal(t, idx.WordsCount(), 0)
	require.Equal(t, idx.NodesCount(), 0)
	last, err := idx.AddString("example")
	require.NoError(t, err)
	require.Equal(t, idx.NodesCount(), 7)
	require.Equal(t, "example", last.Word())
	require.Equal(t, idx.WordsCount(), 0)
	idx.TagID("TEST", "    ")
	require.NoError(t, last.AddTagSet("TEST"))
	require.Equal(t, 1, idx.WordsCount())
}

func TestIndex_AddFetch(t *testing.T) {
	tests := []struct {
		name         string
		addBefore    string
		searchString string
		wantErr      bool
	}{
		{"empty_search_returns_none", "", "", true},
		{"empty_index_returns_none", "", "example", true},
		{"filled_index_full_search", "example", "example", false},
		{"filled_index_root_search", "example", "e", false},
		{"filled_index_partial_search", "example", "exam", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got dag.Node
				err error
			)

			idxInstance := index.New()
			if len(tt.addBefore) != 0 {
				_, err = idxInstance.AddString(tt.addBefore)
				require.NoErrorf(t, err, "%v", err)
			}
			got, err = idxInstance.FetchRunes([]rune(tt.searchString))
			require.Equalf(t, tt.wantErr, err != nil, "want error %v got %v", tt.wantErr, err)
			if err != nil {
				return
			}

			require.EqualValues(t, tt.searchString, got.Word())
		})
	}
}

func TestIndex_Add(t *testing.T) {
	idx := index.New()
	_, err := idx.AddRunes([]rune{'a'})
	require.NoError(t, err)
	require.EqualValues(t, 1, idx.NodesCount())

	_, err = idx.AddRunes([]rune("ab"))
	require.NoError(t, err)
	require.EqualValues(t, 2, idx.NodesCount())

	_, err = idx.AddRunes([]rune("ac"))
	require.NoError(t, err)
	require.EqualValues(t, 3, idx.NodesCount())

	_, err = idx.AddRunes([]rune("abc"))
	require.NoError(t, err)
	require.EqualValues(t, 4, idx.NodesCount())

}

func TestIndexImpl_Add(t *testing.T) {
	for _, tt := range []struct {
		name  string
		add   []string
		count int
	}{
		{"add_nothing", nil, 0},
		{"t", []string{"t"}, 1},
		{"te", []string{"te"}, 2},
		{"tes", []string{"tes"}, 3},
		{"test", []string{"test"}, 4},
		{"text_&_test", []string{"test", "text"}, 6},
		{"test_&_check", []string{"test", "check"}, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			idx := index.New()
			require.Equalf(t, 0, idx.NodesCount(), "expected empty index")

			wordsAdded := 0
			for _, indexWord := range tt.add {
				_, err := idx.AddString(indexWord)
				require.NoError(t, err)
				wordsAdded += 1
			}
			require.Equal(t, len(tt.add), wordsAdded)
			require.Equal(t, tt.count, idx.NodesCount())
		})
	}
}

func TestIndex_NodesCount(t *testing.T) {
	for _, tt := range []struct {
		name  string
		words []string
		want  int
	}{
		{"nothing_added", []string{}, 0},
		{"cyrillic_single_word", []string{"конь"}, 4},
		{"cyrillic_separated_words", []string{"конь", "лес"}, 7},
		{"cyrillic_crossed_words", []string{"конь", "кот"}, 5},
		{"ascii_single_word", []string{"table"}, 5},
		{"ascii_separated_words", []string{"more", "less"}, 8},
		{"ascii_crossed_words", []string{"table", "tell"}, 8},
		{"chinese_characters", []string{"美丽"}, 2},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt

			idx := index.New()
			require.Equal(t, 0, idx.NodesCount())

			for _, word := range tt.words {
				_, err := idx.AddString(word)
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, idx.NodesCount())
		})
	}
}

func TestIndex_AddChild(t *testing.T) {
	idx := index.New()
	_, err := idx.AddString("abcd")
	require.NoError(t, err)

	_, err = idx.FetchRunes([]rune("abc"))

	nodeF, err := idx.AddToNode(3, []rune{'f'})
	require.NoError(t, err)
	require.NotNil(t, nodeF)
	// require.EqualValues(t, 5, nodeF.ID())
	// require.EqualValues(t, 3, nodeF.ParentID())

	nodeFAgain, err := idx.FetchRunes([]rune("abcf"))
	require.NoError(t, err)
	require.NotNil(t, nodeFAgain)
	// require.EqualValues(t, 5, nodeFAgain.ID())
	// require.EqualValues(t, 3, nodeFAgain.ParentID())
}

func TestIndex_AddStringCross(t *testing.T) {
	for _, tt := range []struct {
		name           string
		addFirst       string
		addSecond      string
		search         string
		expectNodeID   dag.ID
		expectParentID dag.ID
	}{
		{"test_x_text", "test", "text", "tex", 5, 2},
		{"write_x_wrote", "write", "wrote", "wrot", 7, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			idx := index.New()
			_, err := idx.AddString(tt.addFirst)
			require.NoError(t, err)
			_, err = idx.AddString(tt.addSecond)
			require.NoError(t, err)
			nodeX, err := idx.FetchString(tt.search)
			require.NoError(t, err)
			require.NotNil(t, nodeX)
			// require.Equal(t, []rune(tt.search[len(tt.search)-1:])[0], nodeX.Rune())
			// require.EqualValues(t, tt.expectNodeID, nodeX.ID())
			// require.EqualValues(t, tt.expectParentID, nodeX.ParentID())
		})
	}
}

func TestIndex_AddToNode(t *testing.T) {
	idx := index.New()
	l0a, err := idx.AddToNode(0, []rune("a"))
	require.NoError(t, err)
	require.EqualValues(t, "a", l0a.Word())

	l0a1b, err := idx.AddToNode(1, []rune("b"))
	require.NoError(t, err)
	require.EqualValues(t, "ab", l0a1b.Word())

	l0a1b2c, err := idx.AddToNode(2, []rune("c"))
	require.NoError(t, err)
	require.EqualValues(t, "abc", l0a1b2c.Word())

	l0a1b2z, err := idx.AddToNode(2, []rune("z"))
	require.NoError(t, err)
	require.EqualValues(t, "abz", l0a1b2z.Word())
}
