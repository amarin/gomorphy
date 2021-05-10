package opencorpora_test

import (
	"reflect"
	"testing"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestLinkType_MarshalBinary(t *testing.T) {
	type testStruct struct {
		name     string
		IdAttr   int
		wantData []byte
		wantErr  bool
	}
	tests := []testStruct{
		{"ok_0", 0, []byte{0}, false},
		{"ok_1", 1, []byte{1}, false},
		{"ok_127", 255, []byte{0xFF}, false},
		{"nok_128", 256, []byte{0}, true},
		{"nok_negative", -1, []byte{0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LinkType{IDAttr: tt.IdAttr}
			gotData, err := l.MarshalBinary()
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

func TestLinkType_UnmarshalBinary(t *testing.T) {
	type testStruct struct {
		name     string
		IdAttr   int
		wantData []byte
		wantErr  bool
	}
	tests := []testStruct{
		{"ok_0", 0, []byte{0}, false},
		{"ok_1", 1, []byte{1}, false},
		{"ok_127", 255, []byte{0xFF}, false},
		{"nok_128", -1, []byte{0x1, 0xFF}, true},
		{"nok_empty", -1, []byte{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LinkType{IDAttr: tt.IdAttr}
			if err := l.UnmarshalBinary(tt.wantData); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil && l.IDAttr != tt.IdAttr {
				t.Errorf("UnmarshalBinary() gotData = %v, want %v", l.IDAttr, tt.IdAttr)
			}
		})
	}
}
