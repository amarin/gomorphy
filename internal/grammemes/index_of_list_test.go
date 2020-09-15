package grammemes_test

import (
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"

	"gitlab.com/go-grammar-rus/grammemes"
)

func TestNewListIndex(t *testing.T) {
	got := grammemes.NewListIndex(grammemes.NewIndex())

	if got.Len() != 0 {
		t.Errorf("NewListIndex() = %v %T, len %d", got, got, got.Len())
	}
}

func TestListIndex_Len(t *testing.T) {
	NOUN := &grammemes.Grammeme{ParentAttr: "", Name: "NOUN", Alias: "", Description: ""}
	ADJF := &grammemes.Grammeme{ParentAttr: "", Name: "ADJF", Alias: "", Description: ""}

	index := grammemes.NewIndex(*NOUN, *ADJF)

	list1 := grammemes.NewList(index, NOUN)
	list2 := grammemes.NewList(index, NOUN, ADJF)
	list2Duplicate := grammemes.NewList(index, ADJF, NOUN)
	list4 := grammemes.NewList(index, ADJF)

	tests := []struct {
		name  string
		index *grammemes.Index
		items []*grammemes.List
		want  int
	}{
		{"ok_empty", index, []*grammemes.List{}, 0},
		{"ok_single", index, []*grammemes.List{list1}, 1},
		{"ok_couple", index, []*grammemes.List{list1, list2}, 2},
		{"ok_triple", index, []*grammemes.List{list1, list2, list4}, 3},
		{"ok_four", index, []*grammemes.List{list1, list2, list4, list2Duplicate}, 3},
	}
	for _, tt := range tests {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			listIndex := grammemes.NewListIndex(tt.index)
			for _, list := range tt.items {
				listIndex.Add(list)
			}
			if got := listIndex.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListIndex_MarshalBinary(t *testing.T) {
	NOUN := &grammemes.Grammeme{ParentAttr: "", Name: "NOUN", Alias: "", Description: ""}
	ADJF := &grammemes.Grammeme{ParentAttr: "", Name: "ADJF", Alias: "", Description: ""}

	index := grammemes.NewIndex(*NOUN, *ADJF)

	list1 := grammemes.NewList(index, NOUN)
	list2 := grammemes.NewList(index, NOUN, ADJF)
	list3 := grammemes.NewList(index, ADJF)

	for _, tt := range []struct {
		name     string
		items    []*grammemes.List
		wantData string
	}{
		{
			"ok_empty",
			[]*grammemes.List{},
			"0800",
		},
		{
			"ok_single",
			[]*grammemes.List{list1},
			"08010100",
		},
		{
			"ok_couple",
			[]*grammemes.List{list1, list2},
			"08020100020001",
		},
		{
			"ok_triple",
			[]*grammemes.List{list1, list2, list3},
			"080301000200010101",
		},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			var listIndex = grammemes.NewListIndex(index, tt.items...)
			if gotData, err := listIndex.MarshalBinary(); err != nil {
				t.Errorf("MarshalBinary() error = %v", err)
				return
			} else if hex.EncodeToString(gotData) != tt.wantData {
				t.Errorf("MarshalBinary() \ngot: %v, \nwant %v", hex.EncodeToString(gotData), tt.wantData)
			}
		})
	}
}

func TestListIndex_UnmarshalFromBuffer(t *testing.T) {
	NOUN := &grammemes.Grammeme{ParentAttr: "", Name: "NOUN", Alias: "", Description: ""}
	ADJF := &grammemes.Grammeme{ParentAttr: "", Name: "ADJF", Alias: "", Description: ""}

	index := grammemes.NewIndex(*NOUN, *ADJF)

	list1 := grammemes.NewList(index, NOUN)
	list2 := grammemes.NewList(index, NOUN, ADJF)
	list3 := grammemes.NewList(index, ADJF)

	for _, tt := range []struct {
		name     string
		items    grammemes.ListList
		wantData string
	}{
		{
			"ok_empty",
			grammemes.ListList{},
			"0800",
		},
		{
			"ok_single",
			grammemes.ListList{list1},
			"08010100",
		},
		{
			"ok_couple",
			grammemes.ListList{list1, list2},
			"08020100020001",
		},
		{
			"ok_triple",
			grammemes.ListList{list1, list2, list3},
			"080301000200010101",
		},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			var listIndex = grammemes.NewListIndex(index)
			if binaryData, err := hex.DecodeString(tt.wantData); err != nil {
				t.Fatalf("cant prepare binary data to test: %v", err)
			} else if err := listIndex.UnmarshalFromBuffer(binutils.NewBuffer(binaryData)); err != nil {
				t.Fatalf("UnmarshalFromBuffer() error = %v, \ndata: %v", err, tt.wantData)
			} else if listIndex.Len() != len(tt.items) {
				t.Errorf(
					"UnmarshalFromBuffer() len mismatch \ngot: %v, \nwant %v, \ngot: %v, \nwant %v,\ndata: %v",
					listIndex.Len(), len(tt.items),
					listIndex.Slice(), tt.items,
					tt.wantData,
				)
			}
		})
	}
}

func TestListIndex_GetOrCreateIdx(t *testing.T) {
	NOUN := &grammemes.Grammeme{ParentAttr: "", Name: "NOUN", Alias: "", Description: ""}
	ADJF := &grammemes.Grammeme{ParentAttr: "", Name: "ADJF", Alias: "", Description: ""}
	index := grammemes.NewIndex(*NOUN, *ADJF)

	list1 := grammemes.NewList(index, NOUN)
	list2 := grammemes.NewList(index, ADJF)
	list3 := grammemes.NewList(index, ADJF, NOUN)

	listIndex := grammemes.NewListIndex(index)

	if got := listIndex.GetOrCreateIdx(list1); got != 0 {
		t.Errorf("GetOrCreateIdx first expected 0, got %v", got)
	}

	if got := listIndex.GetOrCreateIdx(list2); got != 1 {
		t.Errorf("GetOrCreateIdx second expected 1, got %v", got)
	}

	if got := listIndex.GetOrCreateIdx(list3); got != 2 {
		t.Errorf("GetOrCreateIdx third expected 2, got %v", got)
	}

	if got := listIndex.GetOrCreateIdx(list1); got != 0 {
		t.Errorf("GetOrCreateIdx first expected 0, got %v", got)
	}

	if got := listIndex.GetOrCreateIdx(list2); got != 1 {
		t.Errorf("GetOrCreateIdx second expected 1, got %v", got)
	}

	if got := listIndex.GetOrCreateIdx(list3); got != 2 {
		t.Errorf("GetOrCreateIdx third expected 2, got %v", got)
	}
}

func TestListIndex_Idx(t *testing.T) {
	NOUN := &grammemes.Grammeme{ParentAttr: "", Name: "NOUN", Alias: "", Description: ""}
	ADJF := &grammemes.Grammeme{ParentAttr: "", Name: "ADJF", Alias: "", Description: ""}
	index := grammemes.NewIndex(*NOUN, *ADJF)

	list1 := grammemes.NewList(index, NOUN)
	list2 := grammemes.NewList(index, ADJF)

	listIndex := grammemes.NewListIndex(index)

	listIndex.Add(list1)

	if _, err := listIndex.Idx(list1); err != nil {
		t.Errorf("Idx error for existed list: %v", err)
	}

	if _, err := listIndex.Idx(list2); err == nil {
		t.Errorf("Idx no error for not existed list")
	}
}
