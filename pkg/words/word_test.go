package words_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/words"
)

func TestEnding_EqualsTo(t *testing.T) {
	// define some grammemes
	POST := &grammemes.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammemes.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	// common grammeme set in indexes
	common := []grammemes.Grammeme{*POST, *NOUN}
	// two separate indexes with same grammemes set
	indexA := grammemes.NewIndex(common...)
	indexB := grammemes.NewIndex(common...)
	textA := text.RussianText("a")
	textB := text.RussianText("б")

	for _, tt := range []struct {
		name   string
		one    *words.Word
		two    *words.Word
		equals bool
	}{
		{"differs_as_text_differs",
			words.NewWord(indexA, textA, POST),
			words.NewWord(indexA, textB, POST),
			false},
		{"differs_as_different_list_index",
			words.NewWord(indexA, textA, POST),
			words.NewWord(indexB, textA, POST),
			false},
		{"differs_as_different_list_set",
			words.NewWord(indexA, textA, POST),
			words.NewWord(indexA, textA, NOUN),
			false},
		{"differs_both_text_and_set_differs",
			words.NewWord(indexA, textA, POST),
			words.NewWord(indexA, textB, NOUN),
			false},
		{"equals_text_and_set",
			words.NewWord(indexA, textA, POST),
			words.NewWord(indexA, textA, POST),
			true},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			if tt.one.EqualsTo(tt.two) != tt.equals {
				t.Errorf("EqualsTo() = %v, want %v", tt.one.EqualsTo(tt.two), tt.equals)
			}
		})
	}
}

func TestEnding_MarshalBinary(t *testing.T) {
	POST := &grammemes.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammemes.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammemes.NewIndex(*POST, *NOUN)

	for _, tt := range []struct {
		name       string
		endingText text.RussianText
		grammemes  grammemes.List
		wantData   string
		wantErr    bool
	}{
		{"ok_a", "а", *indexA.NewList(POST),
			"c1000100", false},
		{"ok_ya", "я", *indexA.NewList(NOUN),
			"d1000101", false},
		{"ok_on", "он", *indexA.NewList(NOUN, POST),
			"CfCe00020100", false},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			e := words.NewWord(tt.grammemes.GrammemeIndex(), tt.endingText, tt.grammemes.Slice()...)
			if gotData, err := e.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if hex.EncodeToString(gotData) != strings.ToLower(tt.wantData) {
				t.Errorf("MarshalBinary() \ngot: %v,\nwant %v", hex.EncodeToString(gotData), tt.wantData)
			}
		})
	}
}

func TestEnding_UnmarshalFromBuffer(t *testing.T) {
	POST := &grammemes.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammemes.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammemes.NewIndex(*POST, *NOUN)

	for _, tt := range []struct {
		name       string
		endingText text.RussianText
		grammemes  grammemes.List
		wantData   string
		wantErr    bool
	}{
		{"ok_a", "а", *indexA.NewList(POST),
			"c1000100", false},
		{"ok_ya", "я", *indexA.NewList(NOUN),
			"d1000101", false},
		{"ok_on", "он", *indexA.NewList(POST, NOUN),
			"CfCe00020100", false},
		{"ok_on_extra_data", "он", *indexA.NewList(POST, NOUN),
			"CfCe00020100Ff", false},
		{"nok_unknown_grammeme_idx", "он", *indexA.NewList(POST, NOUN),
			"CfCe00020300", true},
		{"nok_data_missed", "он", *indexA.NewList(POST, NOUN),
			"CfCe0001", true},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			e := words.NewWord(indexA, "")
			if data, err := hex.DecodeString(tt.wantData); err != nil {
				t.Fatalf("cant prepare test data: %v", err)
			} else if err := e.UnmarshalFromBuffer(binutils.NewBuffer(data)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFromBuffer() error = %v, wantErr %v", err, tt.wantErr)
			} else if e.Text() != tt.endingText {
				t.Errorf("UnmarshalFromBuffer() name mismatch got `%v` != `%v` expected\ndata: %v",
					e.Text(), tt.endingText, tt.wantData)
			}
		})
	}
}
