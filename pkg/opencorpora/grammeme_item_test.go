package opencorpora_test

import (
	"encoding/hex"
	"testing"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/grammemes"
	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

type testGrammemeStruct struct {
	name        string
	ParentAttr  grammemes.GrammemeName
	Name        grammemes.GrammemeName
	Alias       string
	Description string
	want        string // hex.EncodeToString() result. Take []byte with hex.DecodeFromString()
	wantErr     bool
}

var tests = []testGrammemeStruct{
	{
		"ok_simple_grammeme",
		"POST", "NOUN", "СУЩ", "имя существительное",
		"4e4f554e504f5354d0a1d0a3d0a900d0b8d0bcd18f20d181d183d189d0b5d181d182d0b2d0b8d182d0b5d0bbd18cd0bdd0bed0b500",
		false,
	},
	{
		"ok_empty_parent",
		"", "POST", "ЧР", "часть речи",
		"504f535420202020d0a7d0a000d187d0b0d181d182d18c20d180d0b5d187d0b800",
		false,
	},
	{
		"ok_empty_alias",
		"fake", "lost", "", "что-угодно",
		"6c6f737466616b6500d187d182d0be2dd183d0b3d0bed0b4d0bdd0be00",
		false,
	},
	{
		"ok_empty_description",
		"fake", "lost", "ничего", "",
		"6c6f737466616b65d0bdd0b8d187d0b5d0b3d0be0000",
		false,
	},
}

func TestGrammeme_MarshalBinary(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Grammeme{
				ParentAttr:  tt.ParentAttr,
				Name:        tt.Name,
				Alias:       tt.Alias,
				Description: tt.Description,
			}
			if got, err := g.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if tt.want != hex.EncodeToString(got) {
				t.Errorf("%v.MarshalBinary() \nExp: %v\nGot: %v", g, tt.want, hex.EncodeToString(got))
			}
		})
	}
}

func TestGrammeme_UnmarshalBinary(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := new(Grammeme)
			if data, err := hex.DecodeString(tt.want); err != nil {
				t.Fatalf("Wrong hex string in test: %#v", err)
			} else if err := g.UnmarshalBinary(data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				return
			} else if g.ParentAttr != tt.ParentAttr {
				t.Errorf("expect ParentAttr=%#v, got %#v", g.ParentAttr, tt.ParentAttr)
			} else if g.Name != tt.Name {
				t.Errorf("expect Name=%#v, got %#v", g.Name, tt.Name)
			} else if g.Alias != tt.Alias {
				t.Errorf("expect Alias=%#v, got %#v", g.Alias, tt.Alias)
			} else if g.Description != tt.Description {
				t.Errorf("expect Description=%#v, got %#v", g.Description, tt.Description)
			}
		})
	}
}

func TestGrammeme_UnmarshalFromBuffer(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := new(Grammeme)
			if data, err := hex.DecodeString(tt.want); err != nil {
				t.Fatalf("Wrong hex string in test: %#v", err)
			} else if err := g.UnmarshalFromBuffer(binutils.NewBuffer(data)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFromBuffer() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				return
			} else if g.ParentAttr != tt.ParentAttr {
				t.Errorf("expect ParentAttr=%#v, got %#v", g.ParentAttr, tt.ParentAttr)
			} else if g.Name != tt.Name {
				t.Errorf("expect Name=%#v, got %#v", g.Name, tt.Name)
			} else if g.Alias != tt.Alias {
				t.Errorf("expect Alias=%#v, got %#v", g.Alias, tt.Alias)
			} else if g.Description != tt.Description {
				t.Errorf("expect Description=%#v, got %#v", g.Description, tt.Description)
			}
		})
	}
}
