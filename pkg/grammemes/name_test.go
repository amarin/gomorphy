package grammemes_test

import (
	"reflect"
	"testing"

	"github.com/amarin/gomorphy/pkg/grammemes"
)

func TestGrammemeName_MarshalBinary(t *testing.T) { //nolint:paralleltest
	tests := []struct {
		name     string
		g        grammemes.Name
		wantData []byte
		wantErr  bool
	}{
		{"ok_4_bytes", "NOUN", []byte{78, 79, 85, 78}, false},
		{"ok_empty", "", []byte{32, 32, 32, 32}, false},
		{"nok_3_bytes", "ABC", []byte{}, true},
		{"nok_5_bytes", "bytes", []byte{}, true},
		{"nok_non_ascii", "байт", []byte{}, true},
	}

	for _, tt := range tests { //nolint:paralleltest
		tt := tt // pin variable
		t.Run(tt.name, func(t *testing.T) {
			gotData, err := tt.g.MarshalBinary()
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

func TestGrammemeName_UnmarshalBinary(t *testing.T) { //nolint:paralleltest
	tests := []struct {
		name    string
		g       grammemes.Name
		args    []byte
		wantErr bool
	}{
		{"ok_4_bytes", "aaaa", []byte{97, 97, 97, 97}, false},
		{"nok_3_bytes", "aaa", []byte{97, 97, 97}, true},
		{"nok_5_bytes", "aaaaa", []byte{97, 97, 97, 98, 99}, true},
		{"ok_empty", "", []byte{32, 32, 32, 32}, false},
	}

	for _, tt := range tests { //nolint:paralleltest
		tt := tt // pin variable
		t.Run(tt.name, func(t *testing.T) {
			target := new(grammemes.Name)
			if err := target.UnmarshalBinary(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil && *target != tt.g {
				t.Errorf("UnmarshalBinary(%v)= `%v`, expected `%v`", tt.args, *target, tt.g)
			}
		})
	}
}
