package grammemes_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/amarin/binutils"

	"gitlab.com/go-grammar-rus/grammemes"
)

var (
	indexBinaryPrefix = []byte("GIdx")                        // nolint:gochecknoglobals
	indexHexPrefix    = hex.EncodeToString(indexBinaryPrefix) // nolint:gochecknoglobals
)

type testIndexStruct struct {
	name     string
	known    []grammemes.Grammeme
	wantData string
	wantErr  bool
}

var testCategoryListData = []testIndexStruct{ // nolint:gochecknoglobals
	{"empty_grammemes_list",
		[]grammemes.Grammeme{},
		indexHexPrefix + "00",
		false},
	{"single_empty_grammeme",
		[]grammemes.Grammeme{{"", "", "", ""}},
		indexHexPrefix + "0120202020202020200000",
		false},
	{"single_filled_grammeme",
		[]grammemes.Grammeme{{"", "POST", "ЧР", "часть речи"}},
		indexHexPrefix + "01504f535420202020fef200dec1d3d4d820d2c5dec900",
		false},
	{"couple_of_filled_grammemes",
		[]grammemes.Grammeme{
			{"", "POST", "ЧР", "часть речи"},
			{"POST", "NOUN", "Сущ", "Существительное"},
		},
		indexHexPrefix +
			"02504f535420202020fef200dec1d3d4d820d2c5dec900" +
			"4e4f554e504f5354f3d5dd00f3d5ddc5d3d4d7c9d4c5ccd8cecfc500",
		false},
}

func TestIndex_Idx(t *testing.T) {
	type testStruct struct {
		name     string
		grammeme grammemes.GrammemeName
		want     uint8
		wantErr  bool
	}

	knownGrammemes := []grammemes.Grammeme{
		{"", "first", "", ""},
		{"", "second", "", ""},
		{"", "third", "", ""},
		{"", "fourth", "", ""},
		{"", "fifth", "", ""},
	}
	tests := []testStruct{
		{"err_not_found", "unknown", 0, true},
		{"err_not_found_again", "sixth", 0, true},
	}

	for idx, grammeme := range knownGrammemes {
		tests = append(
			tests,
			testStruct{
				fmt.Sprintf("ok_found_%v_as_%v", grammeme.Name, idx),
				grammeme.Name, uint8(idx), false,
			},
		)
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			x := grammemes.NewIndex(knownGrammemes...)
			if got, err := x.Idx(tt.grammeme); (err != nil) != tt.wantErr {
				t.Errorf("Idx() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil && got != tt.want {
				t.Errorf("Idx() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndex_MarshalBinary(t *testing.T) {
	for _, tt := range testCategoryListData {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			index := grammemes.NewIndex(tt.known...)
			if gotData, err := index.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if hex.EncodeToString(gotData) != tt.wantData {
				t.Errorf("MarshalBinary() \ngot: %v,\nwant %v", hex.EncodeToString(gotData), tt.wantData)
			}
		})
	}
}

// nolint:funlen
func TestIndex_UnmarshalFromBuffer(t *testing.T) {
	tests := make([]testIndexStruct, 0)
	tests = append(tests, testCategoryListData...)
	tests = append(tests, []testIndexStruct{
		// extra data in buffer should not touched and not a error
		{"extra_data_after_single_filled",
			[]grammemes.Grammeme{{"", "POST", "ЧР", "часть речи"}},
			indexHexPrefix + "01504f535420202020fef200dec1d3d4d820d2c5dec900FFFF", false},
		// no data len byte should raise
		{"err_empty_data",
			nil,
			indexHexPrefix + "", true},
		{"err_wrong_prefix",
			nil,
			"ff" + indexHexPrefix + "00", true},
		// len of Grammemes list greater than available data should raise
		{"err_data_missed",
			[]grammemes.Grammeme{{"", "POST", "ЧР", "часть речи"}},
			indexHexPrefix + "02504f535420202020fef200dec1d3d4d820d2c5dec900", true},
	}...)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			index := grammemes.NewIndex()
			data, err := hex.DecodeString(tt.wantData)
			if err != nil { //
				t.Fatalf("Enexpected data string: %v", err)
			}
			err = index.UnmarshalFromBuffer(binutils.NewBuffer(data))
			switch {
			case (err != nil) != tt.wantErr:
				t.Fatalf("UnmarshalFromBuffer() error = %v, wantErr %v", err, tt.wantErr)
			case err != nil:
				return
			case index.Len() != len(tt.known):
				t.Fatalf(
					"UnmarshalFromBuffer() expected %d items, not %d\nData: %v\nExpect: %v\nGot: %v",
					len(tt.known), index.Len(), tt.wantData, tt.known, index.Slice(),
				)
			}

			indexValue := *index
			for idx := range tt.known {
				testItem := tt.known[idx]
				indexItem := indexValue.Slice()[idx]

				switch {
				case testItem.Name != indexItem.Name:
					t.Errorf("UnmarshalFromBuffer() item %d name mismatch: received %v != %v expected",
						idx, indexItem.Name, testItem.Name)
				case testItem.ParentAttr != indexItem.ParentAttr:
					t.Errorf("UnmarshalFromBuffer() item %d parent mismatch: received %v != %v expected",
						idx, indexItem.ParentAttr, testItem.ParentAttr)
				case testItem.Alias != indexItem.Alias:
					t.Errorf("UnmarshalFromBuffer() item %d alias mismatch: received %v != %v expected",
						idx, indexItem.Alias, testItem.Alias)
				case testItem.Description != indexItem.Description:
					t.Errorf("UnmarshalFromBuffer() item %d description mismatch: received %v != %v expected",
						idx, indexItem.Description, testItem.Description)
				}
			}
		})
	}
}

func TestIndex_UnmarshalBinary(t *testing.T) {
	tests := make([]testIndexStruct, 0)
	tests = append(tests, testCategoryListData...)
	tests = append(tests, []testIndexStruct{
		// extra data in buffer should not touched and not a error
		{"extra_data_after_error",
			[]grammemes.Grammeme{{"", "POST", "ЧР", "часть речи"}},
			indexHexPrefix + "01504f535420202020fef200dec1d3d4d820d2c5dec900FFFF", true},
		// no data len byte should raise
		{"err_empty_data", nil, indexHexPrefix + "", true},
		{"err_wrong_prefix", nil, "ff" + indexHexPrefix + "", true},
		// len of Grammemes list greater than available data should raise
		{"err_data_missed",
			[]grammemes.Grammeme{{"", "POST", "ЧР", "часть речи"}},
			indexHexPrefix + "02504f535420202020fef200dec1d3d4d820d2c5dec900", true},
	}...)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			index := grammemes.NewIndex()
			data, err := hex.DecodeString(tt.wantData)
			if err != nil { //
				t.Fatalf("Enexpected data string: %v", err)
			}
			err = index.UnmarshalBinary(data)
			switch {
			case (err != nil) != tt.wantErr:
				t.Fatalf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			case err != nil:
				return
			case index.Len() != len(tt.known):
				t.Fatalf(
					"UnmarshalBinary() expected %d items, not %d\nData: %v\nExpect: %v\nGot: %v",
					len(tt.known), index.Len(), tt.wantData, tt.known, index.Slice(),
				)
			}
			indexValue := *index
			for idx := range tt.known {
				testItem := tt.known[idx]
				indexItem := indexValue.Slice()[idx]
				switch {
				case testItem.Name != indexItem.Name:
					t.Errorf("UnmarshalBinary() item %d name mismatch: received %v != %v expected",
						idx, indexItem.Name, testItem.Name)
				case testItem.ParentAttr != indexItem.ParentAttr:
					t.Errorf("UnmarshalBinary() item %d parent mismatch: received %v != %v expected",
						idx, indexItem.ParentAttr, testItem.ParentAttr)
				case testItem.Alias != indexItem.Alias:
					t.Errorf("UnmarshalBinary() item %d alias mismatch: received %v != %v expected",
						idx, indexItem.Alias, testItem.Alias)
				case testItem.Description != indexItem.Description:
					t.Errorf("UnmarshalBinary() item %d description mismatch: received %v != %v expected",
						idx, indexItem.Description, testItem.Description)
				}
			}
		})
	}
}

func TestIndex_Add(t *testing.T) {
	index := grammemes.NewIndex()
	post := grammemes.NewGrammeme("", "POST", "ЧР", "Часть речи")
	pos1 := grammemes.NewGrammeme("", "POS1", "ЧР", "Часть речи")

	if err := index.Add(*post); err != nil {
		t.Errorf("Unexpected add error %v", err)
	} else if err := index.Add(*post); err == nil {
		t.Errorf("Expected error in add duplicate %v", post)
	} else if err := index.Add(*pos1); err == nil {
		t.Errorf("Expected error in add duplicate alias %v", post)
	}
}

func TestIndex_ByIdx(t *testing.T) {
	testIndex := grammemes.NewIndex(
		*grammemes.NewGrammeme("", "POST", "ЧР", "Часть речи"),
		*grammemes.NewGrammeme("POST", "NOUN", "", ""),
		*grammemes.NewGrammeme("POST", "ASJF", "", ""))

	for _, tt := range []struct {
		name    string
		idx     uint8
		wantErr bool
	}{
		{"ok_existed_1", 0, false},
		{"ok_existed_2", 1, false},
		{"ok_existed_2", 2, false},
		{"nok_missed", 3, true},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			if got, err := testIndex.ByIdx(tt.idx); (err != nil) != tt.wantErr {
				t.Errorf("ByIdx(%v) error = %v, wantErr %v", tt.idx, err, tt.wantErr)
			} else if err == nil && got.Name != testIndex.Slice()[tt.idx].Name {
				t.Errorf("ByIdx(%v) got = %v, want %v", tt.idx, got, testIndex.Slice()[tt.idx])
			}
		})
	}
}

func TestNewIndex(t *testing.T) {
	POST := grammemes.NewGrammeme("", "POST", "ЧР", "Часть речи")
	NOUN := grammemes.NewGrammeme("POST", "NOUN", "", "")
	ADJF := grammemes.NewGrammeme("POST", "ASJF", "", "")

	for _, tt := range []struct {
		name      string
		grammemes []grammemes.Grammeme
	}{
		{"ok_empty", []grammemes.Grammeme{}},
		{"ok_single", []grammemes.Grammeme{*POST}},
		{"ok_pair", []grammemes.Grammeme{*NOUN, *ADJF}},
		{"ok_triplet", []grammemes.Grammeme{*NOUN, *ADJF, *POST}},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			if got := grammemes.NewIndex(tt.grammemes...); got.Len() != len(tt.grammemes) {
				t.Errorf(
					"NewIndex() some grammemes missed:\nexpected: %v\ngot: %v",
					tt.grammemes,
					got.Slice(),
				)
			} else {
				for _, g := range tt.grammemes {
					if _, err := got.ByName(g.Name); err != nil {
						t.Errorf("NewIndex() miss %v:\nexpected: %v\ngot: %v",
							g, tt.grammemes, got.Slice())
					}
				}
			}
		})
	}
}
