package words_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/pkg/words"
)

type marshalUnmarshalTestStruct struct {
	name     string
	index    *words.Index
	words    []*words.Word
	wantData string
	wantErr  bool
}

// nolint:gochecknoglobals
var (
	POST                  = &grammemes.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN                  = &grammemes.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA                = grammemes.NewIndex(*POST, *NOUN)
	word1                 = words.NewWord(indexA, "я", POST, NOUN)
	word2                 = words.NewWord(indexA, "як", POST, NOUN)
	word3                 = words.NewWord(indexA, "яма", POST, NOUN)
	prefix                = "57496478"
	marshalUnmarshalTests = []marshalUnmarshalTestStruct{
		{"empty_map",
			words.NewIndex(indexA),
			[]*words.Word{},
			"0800",
			false},
		{"map_1",
			words.NewIndex(indexA),
			[]*words.Word{word1},
			"0801ffd101020001",
			false},
		{"map_1_2",
			words.NewIndex(indexA),
			[]*words.Word{word1, word2},
			"0802ffd10102000100cb01020001",
			false},
		{"map_1_3",
			words.NewIndex(indexA),
			[]*words.Word{word1, word2, word3},
			"0804ffd10102000100cb0102000100cd0002c101020001",
			false},
		{"map_1_2_3",
			words.NewIndex(indexA),
			[]*words.Word{word1, word3},
			"0803ffd10102000100cd0001c101020001",
			false},
	}
)

func TestIndex_MarshalBinary(t *testing.T) {
	for _, tt := range marshalUnmarshalTests {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			for _, word := range tt.words {
				if err := tt.index.AddWord(word); err != nil {
					t.Fatalf("cant add word to test grammemesIndex: %v", err)
				}
			}
			if gotData, err := tt.index.MarshalBinary(); err != nil {
				t.Errorf("MarshalBinary() error = %v", err)
				return
			} else if hex.EncodeToString(gotData) != tt.wantData {
				slice := tt.index.Container().Slice()
				t.Errorf(
					"MarshalBinary() \n  got: %v \n want: %v\n\nslice:\n  %v",
					hex.EncodeToString(gotData),
					tt.wantData,
					strings.Join(slice.Strings(), "\n  "),
				)
			}
		})
	}
}

func TestIndex_UnmarshalFromBuffer(t *testing.T) {
	tests := append(marshalUnmarshalTests, []marshalUnmarshalTestStruct{
		{"wrong_prefix",
			words.NewIndex(indexA),
			[]*words.Word{},
			"ff" + prefix + "0000",
			true},
	}...)
	for _, tt := range tests {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			index := words.NewIndex(indexA)

			binaryData, err := hex.DecodeString(tt.wantData)

			if err != nil {
				t.Fatalf("cant prepare binary data to test: %v", err)
			}

			buffer := binutils.NewBuffer(binaryData)
			if err := index.UnmarshalFromBuffer(buffer); (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalFromBuffer() \nerr:\n  %v\nexpect error\n  %v\ndata: %v",
					err, tt.wantErr, tt.wantData)
			}

			for _, word := range tt.words {
				if grammemeList := index.SearchForms(word.Text()); len(grammemeList) == 0 {
					slice := index.Container().Slice()
					t.Fatalf(
						"cant find expected word `%v` in index\n\nslice:\n  %v",
						word.Text(),
						strings.Join(slice.Strings(), "\n  "))
				}
			}
		})
	}
}

func TestIndex_SearchForms(t *testing.T) { // проверка поиска через индекс
	indexedWord := words.NewWord(indexA, "йож", POST, NOUN)
	missedWord := words.NewWord(indexA, "килограмм", POST, NOUN)

	list := words.NewIndex(indexA)

	if err := list.AddWord(indexedWord); err != nil {
		t.Fatalf("cant add word to grammemesIndex: %v", err)
	}

	if found := list.SearchForms(missedWord.Text()); len(found) != 0 {
		t.Fatalf("found grammemes for unknown word %v \nin %v", missedWord.Text(), list)
	}

	if found := list.SearchForms(""); len(found) != 0 {
		t.Fatalf("found grammemes for empty word \nin %v", list)
	}

	found := list.SearchForms(indexedWord.Text())
	if len(found) != 1 {
		t.Fatalf("cant find grammemes for existed word %v \nin %v", indexedWord.Text(), list)
	}

	variant := found[0]

	if variant.Len() != 2 {
		t.Fatalf("unexpected grammemes list len %d \nin %v", variant.Len(), variant)
	} else if !variant.EqualTo(indexedWord.Grammemes()) {
		t.Fatalf("unexpected grammemes: \nwant: %v \n got: %v", variant, indexedWord.Grammemes())
	}
}
