package opencorpora_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/dag"
	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestCategory_String(t *testing.T) {
	for _, tt := range []struct {
		name    string
		g       dag.TagName
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
