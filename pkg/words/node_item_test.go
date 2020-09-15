package words_test

import (
	"testing"

	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/words"
)

func TestNode_Parent(t *testing.T) {
	container := words.NewNodeContainer(nil)
	parentOfParent := container.Child('a')
	parent := parentOfParent.Child('б')

	for _, tt := range []struct {
		name string
		node *words.Node
		want *words.Node
	}{
		{"ok_no_parent", parentOfParent, nil},
		{"map_parent", words.NewMappingNode(parent, 'в'), parent},
		{"map_parent_of_parent", words.NewMappingNode(parentOfParent, 'в'), parentOfParent},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin
			if got := tt.node.Parent(); got != tt.want {
				t.Errorf("Parent() = %v, want %v", got, tt.want) //
			}
		})
	}
}

func TestNode_Rune(t *testing.T) {
	nodeA := words.NewMappingNode(nil, 'а')
	nodeB := words.NewMappingNode(nodeA, 'б')

	for _, tt := range []struct {
		name string
		node *words.Node
		want rune
	}{
		{"ok_no_parent", nodeA, 'а'},
		{"ok_has_parent", nodeB, 'б'},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			if tt.node.Rune() != tt.want {
				t.Errorf("%v Rune() = %v, want %v", tt.node, tt.node.Rune(), tt.want)
			}
		})
	}
}

func TestNode_Root(t *testing.T) {
	word := text.RussianText("ёжик")
	runes := []rune(word)
	nodeYo := words.NewMappingNode(nil, runes[0])
	nodeZh := nodeYo.Child(runes[1])
	nodeIi := nodeZh.Child(runes[2])
	nodeKk := nodeIi.Child(runes[3])

	for _, tt := range []struct {
		name string
		node *words.Node
		want *words.Node
	}{
		{"root_of_root", nodeYo, nodeYo},
		{"from_2nd_level", nodeZh, nodeYo},
		{"from_3rd_level", nodeIi, nodeYo},
		{"from_4th_level", nodeKk, nodeYo},
	} {
		tt := tt // pin
		t.Run(tt.name, func(t *testing.T) {
			if tt.node.Root() != nodeYo {
				t.Errorf("Root() = %v, want %v", tt.node.Root(), nodeYo)
			}
		})
	}
}

func TestNode_Slice(t *testing.T) {
	firstA := words.NewMappingNode(nil, 'а')

	if len(firstA.Slice()) != 1 {
		t.Fatalf("%v.Slice() expected len =1, not %d", firstA, len(firstA.Slice()))
	} else if firstA.Parent() != nil {
		t.Fatalf("%v.Parent() expected nil, not %p", firstA, firstA.Parent())
	}

	firstB := firstA.Child('б')

	if len(firstA.Slice()) != 2 {
		t.Fatalf("%v.Slice() expected len =2, not %d", firstB, len(firstA.Slice()))
	} else if firstB.Parent() != firstA {
		t.Fatalf("%v.Parent() expected %p, not %p", firstB, firstA, firstB.Parent())
	}

	secondB := firstB.Child('б')

	if len(firstA.Slice()) != 3 {
		t.Fatalf("%v.Slice() expected len =3, not %d", secondB, len(firstA.Slice()))
	} else if secondB.Parent() != firstB {
		t.Fatalf("%v.Parent() expected %p, not %p", secondB, firstB, secondB.Parent())
	}

	secondA := secondB.Child('а')

	if len(firstA.Slice()) != 4 {
		t.Fatalf("%v.Slice() expected len =3, not %d", secondA, len(firstA.Slice()))
	} else if secondA.Parent() != secondB {
		t.Fatalf("%v.Parent() expected %p, not %p", secondA, secondB, secondA.Parent())
	}
}

func TestNode_Child(t *testing.T) {
	node := words.NewMappingNode(nil, 'а')
	if child := node.Child('б'); child == nil {
		t.Errorf("%v child nil", node)
	} else if child.Parent() != node {
		t.Errorf("%v child parent %p != %p parent", node, child.Parent(), node)
	}
}
