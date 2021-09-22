package grammemes_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/amarin/gomorphy/pkg/grammemes"
	"github.com/stretchr/testify/require"
)

type grammemeTestFields struct {
	ParentAttr grammemes.Name
	Name       grammemes.Name
}

var grammemeTests = []struct { // nolint:gochecknoglobals
	name   string
	fields grammemeTestFields
	want   string
}{
	{
		"with_empty_parent_and_alias_and_description",
		grammemeTestFields{"", "POST"},
		string("Grammeme{" + grammemes.EmptyParent + ",POST}"),
	},
	{
		"with_empty_alias_and_description",
		grammemeTestFields{"POST", "NOUN"},
		"Grammeme{POST,NOUN}",
	},
	{
		"with_empty_description",
		grammemeTestFields{"POST", "NOUN"},
		"Grammeme{POST,NOUN}",
	},
	{
		"with_description",
		grammemeTestFields{"POST", "NOUN"},
		"Grammeme{POST,NOUN}",
	},
}
var grammemeSerializeTest = []struct {
	name     string
	grammeme grammemes.Grammeme
	binData  string
}{
	{
		"with_no_parent",
		*grammemes.NewGrammeme("", "POST"),
		"504f53542d2d2d2d",
	},
	{
		"with_parent",
		*grammemes.NewGrammeme("NOUN", "POST"),
		"504f53544e4f554e",
	},
}

func TestGrammeme_String(t *testing.T) { //nolint:paralleltest
	for _, tt := range grammemeTests { //nolint:paralleltest
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			g := grammemes.NewGrammeme(tt.fields.ParentAttr, tt.fields.Name)
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
			g := grammemes.NewGrammeme(tt.fields.ParentAttr, tt.fields.Name)

			for i := 0; i < b.N; i++ {
				_ = g.String()
			}
		})
	}
}

func TestNewGrammeme(t *testing.T) { //nolint:paralleltest
	type args struct {
		parent grammemes.Name
		name   grammemes.Name
	}

	for _, tt := range []struct { //nolint:paralleltest
		name string
		args args
		want *grammemes.Grammeme
	}{
		{
			name: "root",
			args: args{"", "POST"},
			want: &grammemes.Grammeme{Parent: "----", Name: "POST"},
		},
		{
			name: "child",
			args: args{"POST", "NOUN"},
			want: &grammemes.Grammeme{Parent: "POST", Name: "NOUN"},
		},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			expectedGrammeme := grammemes.NewGrammeme(tt.args.parent, tt.args.name)
			if got := expectedGrammeme; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGrammeme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGrammeme_ReadFrom(t *testing.T) { //nolint:paralleltest
	for _, tt := range grammemeSerializeTest { //nolint:paralleltest
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt

			data, err := hex.DecodeString(tt.binData)
			require.NoError(t, err)
			g := grammemes.NewGrammeme("", "")

			taken, readErr := g.ReadFrom(bytes.NewBuffer(data))
			require.Equal(t, 8, int(taken))
			require.NoError(t, readErr)
			require.Equal(t, tt.grammeme.Name, g.Name)
			require.Equal(t, tt.grammeme.Parent, g.Parent)
		})
	}
}

func TestGrammeme_WriteTo(t *testing.T) { //nolint:paralleltest
	for _, tt := range grammemeSerializeTest { //nolint:paralleltest
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt

			w := new(bytes.Buffer)
			_, err := tt.grammeme.WriteTo(w)
			require.NoError(t, err)
			require.Equal(t, tt.binData, hex.EncodeToString(w.Bytes()))
		})
	}
}
