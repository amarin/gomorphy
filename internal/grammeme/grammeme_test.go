package grammeme_test

import (
	"reflect"
	"testing"

	"github.com/amarin/gomorphy/internal/grammeme"
	"github.com/amarin/gomorphy/internal/text"
)

type grammemeTestFields struct {
	ParentAttr  grammeme.Name
	Name        grammeme.Name
	Alias       text.RussianText
	Description text.RussianText
}

var grammemeTests = []struct { // nolint:gochecknoglobals
	name   string
	fields grammemeTestFields
	want   string
}{
	{
		"with_empty_parent_and_alias_and_description",
		grammemeTestFields{"", "POST", "", ""},
		"Grammeme{,POST,,}",
	},
	{
		"with_empty_alias_and_description",
		grammemeTestFields{"POST", "NOUN", "", ""},
		"Grammeme{POST,NOUN,,}",
	},
	{
		"with_empty_description",
		grammemeTestFields{"POST", "NOUN", "СУЩ", ""},
		"Grammeme{POST,NOUN,СУЩ,}",
	},
	{
		"with_description",
		grammemeTestFields{"POST", "NOUN", "СУЩ", "Существительное"},
		"Grammeme{POST,NOUN,СУЩ,Существительное}",
	},
}

func TestGrammeme_String(t *testing.T) {
	for _, tt := range grammemeTests {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			g := grammeme.NewGrammeme(tt.fields.ParentAttr, tt.fields.Name, tt.fields.Alias, tt.fields.Description)
			if got := g.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkGrammeme_String(b *testing.B) {
	for _, tt := range grammemeTests {
		tt := tt // pin
		b.Run(tt.name, func(b *testing.B) {
			tt := tt // pin
			g := grammeme.NewGrammeme(tt.fields.ParentAttr, tt.fields.Name, tt.fields.Alias, tt.fields.Description)

			for i := 0; i < b.N; i++ {
				_ = g.String()
			}
		})
	}
}

func TestNewGrammeme(t *testing.T) {
	type args struct {
		parent      grammeme.Name
		name        grammeme.Name
		alias       text.RussianText
		description text.RussianText
	}

	for _, tt := range []struct {
		name string
		args args
		want *grammeme.Grammeme
	}{
		{
			name: "ok_base",
			args: args{"", "POST", "", ""},
			want: grammeme.NewGrammeme("", "POST", "", "")},
		{
			name: "ok_child",
			args: args{"POST", "NOUN", "", ""},
			want: grammeme.NewGrammeme("POST", "NOUN", "", "")},
		{
			name: "ok_child_with_alias",
			args: args{"POST", "NOUN", "СУЩ", ""},
			want: grammeme.NewGrammeme("POST", "NOUN", "СУЩ", "")},
		{
			name: "ok_child_with_alias_and_description",
			args: args{"POST", "NOUN", "СУЩ", "Существительное"},
			want: grammeme.NewGrammeme("POST", "NOUN", "СУЩ", "Существительное")},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			expectedGrammeme := grammeme.NewGrammeme(tt.args.parent, tt.args.name, tt.args.alias, tt.args.description)
			if got := expectedGrammeme; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGrammeme() = %v, want %v", got, tt.want)
			}
		})
	}
}
