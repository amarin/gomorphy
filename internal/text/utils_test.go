package text_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"gitlab.com/go-grammar-rus/text"
)

func TestEncodeString(t *testing.T) {
	for _, tt := range []struct {
		name     string
		w        string
		coding   *charmap.Charmap
		wantData string
		wantErr  bool
	}{
		{"one", "один", charmap.KOI8R, "cfc4c9ce", false},
		{"two", "два", charmap.KOI8R, "c4d7c1", false},
		{"three", "три", charmap.KOI8R, "d4d2c9", false},
		{"yat_error", "ѣ", charmap.KOI8R, "", true},
		{"one", "один", charmap.Windows1251, "eee4e8ed", false},
		{"two", "два", charmap.Windows1251, "e4e2e0", false},
		{"three", "три", charmap.Windows1251, "f2f0e8", false},
		{"yat_error", "ѣ", charmap.Windows1251, "", true},
		{"one", "один", charmap.CodePage866, "aea4a8ad", false},
		{"two", "два", charmap.CodePage866, "a4a2a0", false},
		{"three", "три", charmap.CodePage866, "e2e0a8", false},
		{"yat_error", "ѣ", charmap.CodePage866, "", true},
		{"one", "один", charmap.MacintoshCyrillic, "eee4e8ed", false},
		{"two", "два", charmap.MacintoshCyrillic, "e4e2e0", false},
		{"three", "три", charmap.MacintoshCyrillic, "f2f0e8", false},
		{"yat_error", "ѣ", charmap.MacintoshCyrillic, "", true},
	} {
		tt := tt // pin
		t.Run(fmt.Sprintf("%v_%v", tt.name, strings.ToLower(tt.coding.String())), func(t *testing.T) {
			tt := tt // pin
			if gotData, err := text.EncodeString(tt.w, tt.coding); (err != nil) != tt.wantErr {
				t.Errorf("EncodeString(`%v`,%v) error = %v, wantErr %v", tt.w, tt.coding.String(), err, tt.wantErr)
				return
			} else if err == nil && hex.EncodeToString(gotData) != tt.wantData {
				t.Errorf(
					"EncodeString(`%v`,%v)\nGot:  %s\nWant: %v",
					tt.w, tt.coding.String(), hex.EncodeToString(gotData), tt.wantData)
			}
		})
	}
}

func TestDecodeBytes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		w        string
		coding   *charmap.Charmap
		wantData string
		wantErr  bool
	}{
		{"one", "один", charmap.KOI8R, "cfc4c9ce", false},
		{"two", "два", charmap.KOI8R, "c4d7c1", false},
		{"three", "три", charmap.KOI8R, "d4d2c9", false},
		{"one", "один", charmap.Windows1251, "eee4e8ed", false},
		{"two", "два", charmap.Windows1251, "e4e2e0", false},
		{"three", "три", charmap.Windows1251, "f2f0e8", false},
		{"one", "один", charmap.CodePage866, "aea4a8ad", false},
		{"two", "два", charmap.CodePage866, "a4a2a0", false},
		{"three", "три", charmap.CodePage866, "e2e0a8", false},
		{"one", "один", charmap.MacintoshCyrillic, "eee4e8ed", false},
		{"two", "два", charmap.MacintoshCyrillic, "e4e2e0", false},
		{"three", "три", charmap.MacintoshCyrillic, "f2f0e8", false},
	} {
		tt := tt // pin tt
		t.Run(fmt.Sprintf("%v_%v", tt.name, strings.ToLower(tt.coding.String())), func(t *testing.T) {
			if bytesData, err := hex.DecodeString(tt.wantData); err != nil {
				t.Fatalf("Cant prepare data to test: %v", err)
			} else if word, err := text.DecodeBytes(bytesData, tt.coding); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil && word != tt.w {
				t.Errorf("RussianText.UnmarshalBinary(`%v`)\nGot:  %s\nWant: %v", tt.wantData, word, tt.w)
			}
		})
	}
}
