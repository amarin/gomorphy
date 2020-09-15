package grammemes_test

import (
	"encoding/hex"
	"testing"

	"gitlab.com/go-grammar-rus/grammemes"
)

func TestListList_MarshalBinaryWithIndex(t *testing.T) {
	ROOT := grammemes.NewGrammeme("", "ROOT", "", "")
	TAG1 := grammemes.NewGrammeme("ROOT", "TAG1", "", "")
	TAG2 := grammemes.NewGrammeme("ROOT", "TAG2", "", "")
	TAG3 := grammemes.NewGrammeme("ROOT", "TAG3", "", "")
	TAG4 := grammemes.NewGrammeme("", "TAG4", "", "")

	index := grammemes.NewIndex(*ROOT, *TAG1, *TAG2, *TAG3, *TAG4)

	list1 := grammemes.NewList(index, ROOT, TAG1)
	list2 := grammemes.NewList(index, ROOT, TAG2)
	list3 := grammemes.NewList(index, ROOT, TAG3)
	list11 := grammemes.NewList(index, ROOT, TAG1, TAG4)
	list12 := grammemes.NewList(index, ROOT, TAG2, TAG4)
	list13 := grammemes.NewList(index, ROOT, TAG3, TAG4)

	listIndex := grammemes.NewListIndex(index, list1, list2, list3, list11, list12, list13)

	for _, tt := range []struct {
		name       string
		listOfList *grammemes.ListList
		want       string
		wantErr    bool
	}{
		{"ok_empty",
			grammemes.NewListOfList(),
			"080800", false},
		{"ok_one",
			grammemes.NewListOfList(list1),
			"08080100", false},
		{"ok_two",
			grammemes.NewListOfList(list1, list2),
			"0808020001", false},
		{"ok_three",
			grammemes.NewListOfList(list1, list2, list3),
			"080803000102", false},
		{"ok_four",
			grammemes.NewListOfList(list1, list2, list3, list11),
			"08080400010203", false},
		{"ok_five",
			grammemes.NewListOfList(list1, list2, list3, list11, list12),
			"0808050001020304", false},
		{"ok_six",
			grammemes.NewListOfList(list1, list2, list3, list11, list12, list13),
			"080806000102030405", false},
		{"ok_with_repeat",
			grammemes.NewListOfList(list1, list2, list3, list11, list12, list13, list1, list2),
			"0808080001020304050001", false},
	} {
		tt1 := tt // pin tt
		t.Run(tt1.name, func(t *testing.T) {
			tt2 := tt1 //
			got, err := tt2.listOfList.MarshalBinaryWithIndex(listIndex)
			switch {
			case (err != nil) != tt2.wantErr:
				t.Fatalf("MarshalBinaryWithIndex() error = %v, wantErr %v", err, tt2.wantErr)
			case hex.EncodeToString(got) != tt2.want:
				t.Fatalf("MarshalBinaryWithIndex() expect:\n%v\ngot: %v", tt2.want, hex.EncodeToString(got))
			}
		})
	}
}
