package opencorpora_test

import (
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

type testCategoryListStruct struct {
	name     string
	c        CategoryList
	wantData string
	wantErr  bool
}

var testCategoryListData = []testCategoryListStruct{
	{"empty_category_list",
		CategoryList{},
		"00",
		false},
	{"single_empty_category",
		CategoryList{&Category{""}},
		"0120202020",
		false},
	{"single_noun_category",
		CategoryList{&Category{"NOUN"}},
		"014e4f554e",
		false},
	{"plur_noun",
		CategoryList{&Category{"NOUN"}, &Category{"plur"}},
		"024e4f554e706c7572",
		false},
	{"plur_noun_gent",
		CategoryList{&Category{"NOUN"}, &Category{"plur"}, &Category{"gent"}},
		"034e4f554e706c757267656e74",
		false},
}

func TestCategoryList_MarshalBinary(t *testing.T) {
	for _, tt := range testCategoryListData {
		t.Run(tt.name, func(t *testing.T) {
			if gotData, err := tt.c.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if hex.EncodeToString(gotData) != tt.wantData {
				t.Errorf("MarshalBinary() \ngot: %v,\nwant %v", hex.EncodeToString(gotData), tt.wantData)
			}
		})
	}
}

func TestCategoryList_UnmarshalBinary(t *testing.T) {
	tests := make([]testCategoryListStruct, 0)
	for _, tt := range testCategoryListData {
		tests = append(tests, tt)
	}
	tests = append(tests, []testCategoryListStruct{
		// no data len byte should raise
		{"err_empty_data", nil, "", true},
		// len of CategoryList less than available data should raise
		{"err_extra_data", nil, "024e4f554e706c757267656e74", true},
		// len of CategoryList greater than available data should raise
		{"err_data_missed", nil, "024e4f554e", true},
	}...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryListPtr := new(CategoryList)
			if data, err := hex.DecodeString(tt.wantData); err != nil {
				t.Errorf("Enexpected data string: %v", err)
			} else if err := categoryListPtr.UnmarshalBinary(data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				return
			} else if len(*categoryListPtr) != len(tt.c) {
				t.Errorf("UnmarshalBinary() expected %d ites, got %d", len(tt.c), len(*categoryListPtr))
			}
			categoryList := *categoryListPtr
			for idx := range tt.c {
				if tt.c[idx].VAttr != categoryList[idx].VAttr {
					t.Errorf("UnmarshalBinary() item %d mismatch: received %v != %v expected", idx, categoryList[idx], tt.c[idx])
				}
			}
		})
	}
}

func TestCategoryList_UnmarshalFromBuffer(t *testing.T) {
	tests := make([]testCategoryListStruct, 0)
	for _, tt := range testCategoryListData {
		tests = append(tests, tt)
	}
	tests = append(tests, []testCategoryListStruct{
		// extra data in buffer should not touched and not a error
		{"extra_data_untouched",
			CategoryList{&Category{"NOUN"}, &Category{"plur"}},
			"024e4f554e706c757267656e74",
			false},
		// no data len byte should raise
		{"err_empty_data", nil, "", true},
		// len of CategoryList greater than available data should raise
		{"err_data_missed", nil, "024e4f554e", true},
	}...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryListPtr := new(CategoryList)
			if data, err := hex.DecodeString(tt.wantData); err != nil {
				t.Errorf("Enexpected data string: %v", err)
			} else if err := categoryListPtr.UnmarshalFromBuffer(binutils.NewBuffer(data)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				return
			} else if len(*categoryListPtr) != len(tt.c) {
				t.Errorf("UnmarshalBinary() expected %d ites, got %d", len(tt.c), len(*categoryListPtr))
			}
			categoryList := *categoryListPtr
			for idx := range tt.c {
				if tt.c[idx].VAttr != categoryList[idx].VAttr {
					t.Errorf("UnmarshalBinary() item %d mismatch: received %v != %v expected", idx, categoryList[idx], tt.c[idx])
				}
			}
		})
	}
}
