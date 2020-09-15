package opencorpora_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

var validGrammemes = []*Grammeme{
	{
		ParentAttr:  "",
		Name:        "POST",
		Alias:       "ЧР",
		Description: "часть речи",
	},
	{
		ParentAttr:  "POST",
		Name:        "NOUN",
		Alias:       "СУЩ",
		Description: "имя существительное",
	},
	{
		ParentAttr:  "POST",
		Name:        "ADJF",
		Alias:       "ПРИЛ",
		Description: "имя прилагательное (полное)",
	},
}

func TestGrammemes_MarshalUnmarshalBinary(t *testing.T) {
	tests := []struct {
		name   string
		fields []*Grammeme
	}{
		{"ok_valid_grammemes", validGrammemes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldGrammemes := Grammemes{Grammeme: tt.fields}
			newGrammemes := new(Grammemes)
			if gotData, err := oldGrammemes.MarshalBinary(); err != nil {
				t.Errorf("MarshalBinary() error = %v", err)
			} else if err := newGrammemes.UnmarshalBinary(gotData); err != nil {
				t.Errorf("UnmarshalBinary() error = %v\n\nData: %v", err, hex.EncodeToString(gotData))
			} else if len(oldGrammemes.Grammeme) != len(newGrammemes.Grammeme) {
				t.Errorf("Unmarshaled length %d != %d of original", len(newGrammemes.Grammeme), len(oldGrammemes.Grammeme))
			} else if fmt.Sprintf("%v", oldGrammemes) != fmt.Sprintf("%v", newGrammemes) {
				t.Errorf("Unmarshaled %v != %v original", newGrammemes, oldGrammemes)
			}
		})
	}
}
