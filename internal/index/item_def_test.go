package index_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/dag"
)

func TestItem_String(t *testing.T) {
	for _, tt := range []struct {
		ID       dag.ID
		Parent   dag.ID
		Letter   rune
		Variants index.CollectionID
		want     string
	}{
		{1, 1, 'z', 0, "z1_1_0"},
		{2, 1, 'б', 1, "б2_1_1"},
		{3, 2, 'µ', 99, "µ3_2_99"},
	} {
		expectedName := string(tt.Letter) + strings.Join([]string{
			strconv.Itoa(int(tt.ID)),
			strconv.Itoa(int(tt.Parent)),
			strconv.Itoa(int(tt.Variants)),
		}, "_")
		t.Run(expectedName, func(t *testing.T) {
			i := index.Item{
				Parent:   tt.Parent,
				ID:       tt.ID,
				Letter:   tt.Letter,
				Variants: tt.Variants,
			}
			if got := i.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func Test_itemList_Get(t *testing.T) {
// 	testItem0 := index.Item{Parent: 0, ID: 0, Variants: 0}
// 	testItem1 := index.Item{Parent: 0, ID: 1, Letter: 'a', Variants: 0}
// 	testItem2 := index.Item{Parent: 0, ID: 2, Letter: 'b', Variants: 0}
// 	for _, tt := range []struct {
// 		name     string
// 		itemList []index.Item
// 		id       dag.ID
// 		want     *index.Item
// 	}{
// 		{"nothing_for_0", []index.Item{testItem0, testItem1}, 0, nil},
// 		{"nothing_from_empty", make([]index.Item, 0), 1, nil},
// 		{"nothing_for_not_existed", []index.Item{testItem0, testItem1}, 2, nil},
// 		{"expected_1", []index.Item{testItem0, testItem1}, 1, &testItem1},
// 		{"expected_2", []index.Item{testItem0, testItem1, testItem2}, 2, &testItem2},
// 	} {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := tt.itemList.Get(tt.id)
//
// 			if tt.want == nil {
// 				require.Nil(t, got)
// 				return
// 			}
//
// 			require.NotNilf(t, got, "[%d] from %s", tt.id, tt.itemList)
// 			require.Equalf(t, tt.want.ID, got.ID, "[%d] from %s", tt.id, tt.itemList)
// 			require.Equalf(t, tt.want.Parent, got.Parent, "[%d] from %s", tt.id, tt.itemList)
// 			require.Equalf(t, tt.want.Variants, got.Variants, "[%d] from %s", tt.id, tt.itemList)
// 			require.Equalf(t, tt.want.Letter, got.Letter, "[%d] from %s", tt.id, tt.itemList)
// 		})
// 	}
// }
