package dag_test

import (
	"fmt"
	"testing"

	"github.com/amarin/gomorphy/pkg/dag"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest
func TestIndexImpl_SetNodeConstructor(t *testing.T) {
	for _, tt := range []struct { //nolint:paralleltest
		name        string
		constructor dag.NodeConstructor
	}{
		{"set_nil_constructor", nil},
		{"set_default_node_constructor", dag.DefaultNodeConstructor},
		{"set_specific_node_constructor", func(p dag.Node, r rune, d interface{}) dag.Node {
			node := new(dag.NodeImpl)
			node.SetParent(p)
			node.SetRune(r)
			node.SetData(d)

			return node
		}},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			idx := dag.NewIndex()
			idx.SetNodeConstructor(tt.constructor)
			wantPrt := fmt.Sprintf("%p", tt.constructor)
			gotPtr := fmt.Sprintf("%p", idx.NodeConstructor())
			require.Equalf(
				t, wantPrt, gotPtr,
				"constructor not set, expected % got %v", wantPrt, gotPtr)
		})
	}
}
