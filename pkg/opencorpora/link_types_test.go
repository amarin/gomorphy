package opencorpora_test

import (
	"reflect"
	"testing"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestLinkTypes_MarshalBinary(t *testing.T) {

	tests := []struct {
		name     string
		Type     []*LinkType
		wantData []byte
		wantErr  bool
	}{
		{"ok_couple_of_types", []*LinkType{{0}, {1}}, []byte{0x0, 0x1}, false},
		{"ok_empty", []*LinkType{}, []byte{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LinkTypes{Type: tt.Type}
			if gotData, err := l.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if len(gotData) != len(tt.wantData) {
				t.Errorf("MarshalBinary() gotData = %v len %d, want %d", gotData, len(gotData), len(tt.wantData))
			} else if len(gotData) > 0 && !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("MarshalBinary() gotData = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestLinkTypes_UnmarshalBinary(t *testing.T) {
	tests := []struct {
		name     string
		Type     []*LinkType
		wantData []byte
		wantErr  bool
	}{
		{"ok_couple_of_types", []*LinkType{{0}, {1}}, []byte{0x0, 0x1}, false},
		{"ok_empty", []*LinkType{}, []byte{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := new(LinkTypes)
			if err := l.UnmarshalBinary(tt.wantData); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil && len(l.Type) != len(tt.Type) {
				t.Errorf("UnmarshalBinary(%v) error = nil, got %d elements, expected %d", tt.wantData, len(l.Type), len(tt.Type))
			} else {
				for idx, linkType := range l.Type {
					if tt.Type[idx].IdAttr != linkType.IdAttr {
						t.Errorf("UnmarshalBinary(%v) error = nil, %d elem id=%v, expected %v", tt.wantData, idx, linkType.IdAttr, tt.Type[idx].IdAttr)
					}
				}
			}
		})
	}
}
