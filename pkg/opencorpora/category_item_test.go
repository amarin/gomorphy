package opencorpora_test

import (
	"reflect"
	"testing"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/internal/grammeme"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestCategory_MarshalBinary(t *testing.T) {
	for _, tt := range []struct {
		name     string
		g        grammeme.Name
		wantData []byte
		wantErr  bool
	}{
		{"ok_4_bytes", "NOUN", []byte{78, 79, 85, 78}, false},
		{"ok_empty", "", []byte{32, 32, 32, 32}, false},
		{"nok_3_bytes", "ABC", []byte{}, true},
		{"nok_5_bytes", "bytes", []byte{}, true},
		{"nok_non_ascii", "байт", []byte{}, true},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			c := Category{VAttr: tt.g}
			gotData, err := c.MarshalBinary()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("MarshalBinary() gotData = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestCategory_UnmarshalBinary(t *testing.T) {
	for _, tt := range []struct {
		name    string
		g       grammeme.Name
		args    []byte
		wantErr bool
	}{
		{"ok_4_bytes", "aaaa", []byte{97, 97, 97, 97}, false},
		{"nok_3_bytes", "aaa", []byte{97, 97, 97}, true},
		{"nok_5_bytes", "aaaaa", []byte{97, 97, 97, 98, 99}, true},
		{"ok_empty", "", []byte{32, 32, 32, 32}, false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			target := new(Category)
			if err := target.UnmarshalBinary(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil && target.VAttr != tt.g {
				t.Errorf("UnmarshalBinary(%v)= %v, expected %v", tt.args, *target, tt.g)
			}
		})
	}
}

func TestCategory_String(t *testing.T) {
	for _, tt := range []struct {
		name    string
		g       grammeme.Name
		args    []byte
		wantErr bool
	}{
		{"ok_4_bytes", "aaaa", []byte{97, 97, 97, 97}, false},
		{"nok_3_bytes", "aaa", []byte{97, 97, 97}, true},
		{"nok_5_bytes", "aaaaa", []byte{97, 97, 97, 98, 99}, true},
		{"ok_empty", "", []byte{32, 32, 32, 32}, false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			target := &Category{VAttr: tt.g}
			if target.String() != string(tt.g) {
				t.Errorf("Category(%v).String() = %v != %v", tt.g, target.String(), tt.g)
			}
		})
	}
}

func TestCategory_UnmarshalFromBuffer(t *testing.T) {
	for _, tt := range []struct {
		name    string
		g       grammeme.Name
		args    []byte
		wantErr bool
	}{
		{"ok_4_bytes", "aaaa", []byte{97, 97, 97, 97}, false},
		{"ok_extra_bytes", "aaaa", []byte{97, 97, 97, 97, 98}, false},
		{"nok_3_bytes", "", []byte{97, 97, 97}, true},
		{"ok_empty", "", []byte{32, 32, 32, 32}, false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			target := new(Category)
			buffer := binutils.NewBuffer(tt.args)
			if err := target.UnmarshalFromBuffer(buffer); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFromBuffer() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil && target.VAttr != tt.g {
				t.Errorf("UnmarshalFromBuffer(%v)= %v, expected %v", tt.args, *target, tt.g)
			}
		})
	}
}
