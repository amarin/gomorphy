package dag_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/amarin/binutils"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/dag"
)

type tagTestFields struct {
	ParentAttr dag.TagName
	Name       dag.TagName
}

var tagTests = []struct { // nolint:gochecknoglobals
	name   string
	fields tagTestFields
	want   string
}{
	{
		"with_empty_parent_and_alias_and_description",
		tagTestFields{"", "POST"},
		"POST",
	},
	{
		"with_empty_alias_and_description",
		tagTestFields{"POST", "NOUN"},
		"NOUN",
	},
}
var tagSerializeTest = []struct {
	name    string
	tag     dag.Tag
	binData string
}{
	{
		"with_no_parent",
		*dag.NewTag("", "POST"),
		"2d2d2d2d504f5354",
	},
	{
		"with_parent",
		*dag.NewTag("NOUN", "POST"),
		"4e4f554e504f5354",
	},
}

func TestTag_String(t *testing.T) { //nolint:paralleltest
	for _, tt := range tagTests { //nolint:paralleltest
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			g := dag.NewTag(tt.fields.ParentAttr, tt.fields.Name)
			if got := g.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkTag_String(b *testing.B) {
	for _, tt := range tagTests {
		tt := tt // pin
		b.Run(tt.name, func(b *testing.B) {
			tt := tt // pin
			g := dag.NewTag(tt.fields.ParentAttr, tt.fields.Name)

			for i := 0; i < b.N; i++ {
				_ = g.String()
			}
		})
	}
}

func TestNewTag(t *testing.T) { //nolint:paralleltest
	type args struct {
		parent dag.TagName
		name   dag.TagName
	}

	for _, tt := range []struct { //nolint:paralleltest
		name string
		args args
		want *dag.Tag
	}{
		{
			name: "root",
			args: args{"", "POST"},
			want: &dag.Tag{Parent: "----", Name: "POST"},
		},
		{
			name: "child",
			args: args{"POST", "NOUN"},
			want: &dag.Tag{Parent: "POST", Name: "NOUN"},
		},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			expectedTag := dag.NewTag(tt.args.parent, tt.args.name)
			if got := expectedTag; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTag_BinaryReadFrom(t *testing.T) {
	for _, tt := range tagSerializeTest {
		t.Run(tt.name, func(t *testing.T) {
			g := &dag.Tag{}
			data, err := hex.DecodeString(tt.binData)
			require.NoError(t, err)
			reader := binutils.NewBinaryReader(bytes.NewBuffer(data))
			_, err = g.BinaryReadFrom(reader)
			require.NoError(t, err)
			require.Equal(t, tt.tag.Name, g.Name)
			require.Equal(t, tt.tag.Parent, g.Parent)
		})
	}
}
