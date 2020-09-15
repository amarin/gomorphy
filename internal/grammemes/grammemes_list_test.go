package grammemes_test

import (
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"

	"gitlab.com/go-grammar-rus/grammemes"
)

func TestList_MarshalBinary(t *testing.T) {
	testIndex := grammemes.NewIndex(
		*grammemes.NewGrammeme("", "POST", "", ""),
		*grammemes.NewGrammeme("POST", "NOUN", "", ""),
		*grammemes.NewGrammeme("POST", "ADJF", "", ""),
		*grammemes.NewGrammeme("POST", "ADJS", "", ""),
	)

	for _, tt := range []struct {
		name          string
		grammemes     []grammemes.GrammemeName
		expectedBytes string
		wantErr       bool
	}{
		{"ok_empty", []grammemes.GrammemeName{}, "00", false},
		{"ok_single_0", []grammemes.GrammemeName{"POST"}, "0100", false},
		{"ok_single_1", []grammemes.GrammemeName{"NOUN"}, "0101", false},
		{"ok_single_2", []grammemes.GrammemeName{"ADJF"}, "0102", false},
		{"ok_couple", []grammemes.GrammemeName{"NOUN", "ADJF"}, "020102", false},
		{"ok_couple_2", []grammemes.GrammemeName{"NOUN", "ADJS"}, "020103", false},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			grammemeList := grammemes.NewList(testIndex)
			for _, name := range tt.grammemes {
				if g, err := testIndex.ByName(name); err != nil {
					t.Errorf("Cant take known grammeme from dict: %v", err)
				} else if err := grammemeList.Add(g); err != nil {
					t.Errorf("Cant add grammeme: %v", err)
				}
			}
			if grammemeList.Len() != len(tt.grammemes) {
				t.Errorf(
					"Unexpected grammemes count for test: want: %v got: %v",
					tt.grammemes, grammemeList.Slice(),
				)
			} else if got, err := grammemeList.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, eq %v", err, tt.wantErr)
			} else if err == nil && hex.EncodeToString(got) != tt.expectedBytes {
				t.Errorf(
					"MarshalBinary() \nwant %v,\ngot: %v\n%v",
					tt.expectedBytes, hex.EncodeToString(got), grammemeList.Slice(),
				)
			}
		})
	}
}

// nolint:funlen
func TestList_UnmarshalFromBuffer(t *testing.T) {
	testIndex := grammemes.NewIndex(
		*grammemes.NewGrammeme("", "POST", "", ""),
		*grammemes.NewGrammeme("POST", "NOUN", "", ""),
		*grammemes.NewGrammeme("POST", "ADJF", "", ""),
		*grammemes.NewGrammeme("POST", "ADJS", "", ""),
	)
	for _, tt := range []struct {
		name          string
		grammemes     []grammemes.GrammemeName
		expectedBytes string
		wantErr       bool
	}{
		{"ok_empty",
			[]grammemes.GrammemeName{}, "00", false},
		{"ok_single_0",
			[]grammemes.GrammemeName{"POST"}, "0100", false},
		{"ok_single_1",
			[]grammemes.GrammemeName{"NOUN"}, "0101", false},
		{"ok_single_extra_data",
			[]grammemes.GrammemeName{"ADJF"}, "0102FF", false},
		{"ok_couple",
			[]grammemes.GrammemeName{"NOUN", "ADJF"}, "020102", false},
		{"ok_couple_2",
			[]grammemes.GrammemeName{"NOUN", "ADJS"}, "020103", false},
		{"ok_couple_extra_data",
			[]grammemes.GrammemeName{"NOUN", "ADJS"}, "020103FF", false},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			grammemeList := grammemes.NewList(testIndex)
			if data, err := hex.DecodeString(tt.expectedBytes); err != nil {
				t.Errorf("cant create test []byte from byte string \nbytes: %v,\nerror: %v", tt.expectedBytes, err)
			} else if err := grammemeList.UnmarshalFromBuffer(binutils.NewBuffer(data)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFromBuffer() error = %v, eq %v", err, tt.wantErr)
			} else if err == nil && grammemeList.Len() != len(tt.grammemes) {
				t.Errorf("Expected %d grammemes not %d", len(tt.grammemes), grammemeList.Len())
			} else if err == nil {
				for idx, name := range tt.grammemes {
					if grammemeList.Slice()[idx].Name != name {
						t.Errorf("Expected %d grammeme %v not %v", idx, name, grammemeList.Slice()[idx].Name)
					}
				}
			}
		})
	}
}

func TestList_Add(t *testing.T) {
	testIndex := grammemes.NewIndex(
		*grammemes.NewGrammeme("", "POST", "", ""),
		*grammemes.NewGrammeme("POST", "NOUN", "", ""),
		*grammemes.NewGrammeme("POST", "ADJF", "", ""),
		*grammemes.NewGrammeme("POST", "ADJS", "", ""),
	)
	list := grammemes.NewList(testIndex)

	if err := list.Add(&testIndex.Slice()[0]); err != nil {
		t.Errorf("Unexpected error when add one grammeme: %v", err)
	} else if err := list.Add(&testIndex.Slice()[0]); err == nil {
		t.Errorf("No expected error when add duplicate grammeme: %v", err)
	}
}

func TestList_EqualTo(t *testing.T) {
	// define some grammemes
	POST := grammemes.NewGrammeme("", "POST", "", "")
	NOUN := grammemes.NewGrammeme("POST", "NOUN", "", "")
	ADJF := grammemes.NewGrammeme("POST", "ADJF", "", "")
	// common grammeme set in indexes
	common := []grammemes.Grammeme{*POST, *NOUN, *ADJF}
	// two separate indexes with same grammemes set
	indexA := grammemes.NewIndex(common...)
	indexB := grammemes.NewIndex(common...)

	tests := []struct {
		name string
		one  *grammemes.List
		two  *grammemes.List
		eq   bool
	}{
		// grammeme list different if indexes different
		{"neq_different_indexes",
			grammemes.NewList(indexA),
			grammemes.NewList(indexB),
			false},
		// grammeme list different if grammeme set length differs
		{"neq_different_len",
			grammemes.NewList(indexA, POST),
			grammemes.NewList(indexA),
			false},
		// grammeme list different if grammeme set differs
		{"neq_different_set",
			grammemes.NewList(indexA, POST, NOUN),
			grammemes.NewList(indexA, POST, ADJF),
			false},
		// grammeme list same if grammeme set equals
		{"eq_same_set",
			grammemes.NewList(indexA, POST, NOUN),
			grammemes.NewList(indexA, POST, NOUN),
			true},
		// grammeme list same if grammeme set equals even in different order
		{"eq_same_set_different_order",
			grammemes.NewList(indexA, POST, NOUN),
			grammemes.NewList(indexA, NOUN, POST),
			true},
		{"eq_same_set_different_order_in_three",
			grammemes.NewList(indexA, POST, NOUN, ADJF),
			grammemes.NewList(indexA, NOUN, ADJF, POST),
			true},
	}
	for _, tt := range tests {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			if tt.one.EqualTo(tt.two) != tt.eq {
				t.Errorf("expected equal %v", tt.eq)
			}
		})
	}
}
