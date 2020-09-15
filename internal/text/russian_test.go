package text_test

import (
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"

	"gitlab.com/go-grammar-rus/text"
)

func TestWord_MarshalBinary(t *testing.T) {
	for _, tt := range []struct {
		name     string
		w        text.RussianText
		wantData string
		wantErr  bool
	}{
		{"marshal_one", "один", "cfc4c9ce00", false},
		{"marshal_two", "два", "c4d7c100", false},
		{"marshal_three", "три", "d4d2c900", false},
		{"marshal_yat_error", "ѣ", "", true},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			if gotData, err := tt.w.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil && hex.EncodeToString(gotData) != tt.wantData {
				t.Errorf(
					"RussianText(`%v`).MarshalBinary()\nGot:  %s\nWant: %v",
					tt.w, hex.EncodeToString(gotData), tt.wantData)
			}
		})
	}
}

func TestWord_UnmarshalBinary(t *testing.T) {
	for _, tt := range []struct {
		name     string
		w        text.RussianText
		wantData string
		wantErr  bool
	}{
		{"unmarshal_one", "один", "cfc4c9ce00", false},
		{"unmarshal_two", "два", "c4d7c100", false},
		{"unmarshal_three", "три", "d4d2c900", false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			word := text.NewRussianText("")
			if bytesData, err := hex.DecodeString(tt.wantData); err != nil {
				t.Fatalf("Cant prepare data to test: %v", err)
			} else if err := word.UnmarshalBinary(bytesData); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil && word != tt.w {
				t.Errorf("RussianText.UnmarshalBinary(`%v`)\nGot:  %s\nWant: %v", tt.wantData, word, tt.w)
			}
		})
	}
}

func TestWord_UnmarshalFromBuffer(t *testing.T) {
	for _, tt := range []struct {
		name     string
		w        text.RussianText
		wantData string
		wantErr  bool
	}{
		{"unmarshal_one", "один", "cfc4c9ce00", false},
		{"unmarshal_two", "два", "c4d7c100", false},
		{"unmarshal_three", "три", "d4d2c900", false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			word := new(text.RussianText)
			if bytesData, err := hex.DecodeString(tt.wantData); err != nil {
				t.Fatalf("Cant prepare data to test: %v", err)
			} else if buffer := binutils.NewBuffer(bytesData); false {
			} else if err := word.UnmarshalFromBuffer(buffer); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil && *word != tt.w {
				t.Errorf("RussianText.UnmarshalBinary(`%v`)\nGot:  %s\nWant: %v", tt.wantData, *word, tt.w)
			}
		})
	}
}

func TestRussianText_Len(t *testing.T) {
	for _, tt := range []struct {
		w    text.RussianText
		want int
	}{
		{"", 0},
		{"я", 1},
		{"ты", 2},
		{"она", 3},
		{"тебе", 4},
		{"наших", 5},
		{"стакан", 6},
		{"кремень", 7},
		{"карандаш", 8},
		{"наволочка", 9},
		{"крокодилище", 11},
		{"пододеяльник", 12},
		{"человеконенавистничество", 24},
	} {
		if tt.w.Len() != tt.want {
			t.Fatalf("Len(`%v`) = %v, want %v", tt.w, tt.w.Len(), tt.want)
		}
	}
}
