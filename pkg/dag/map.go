package dag

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amarin/binutils"
)

// NodeMap provides mapping of runes to Node's.
type NodeMap map[rune]Node

func (n NodeMap) BinaryWriteTo(writer *binutils.BinaryWriter) error {
	// TODO implement me
	panic("implement me")
}

// String returns NodeMap string representation.
func (n NodeMap) String() string {
	res := "NodeMap("

	if len(n) < 10 {
		nodes := make([]string, 0)
		for char, child := range n {
			nodes = append(nodes, fmt.Sprintf("`%v`:%v", string(char), child))
		}

		res += strings.Join(nodes, ",")
	} else {
		res += "len()=" + strconv.Itoa(len(n))
	}

	return res + ")"
}

// IdMap provides mapping of runes to ID's.
type IdMap map[rune]ID

// String returns NodeMap string representation.
func (idMap IdMap) String() string {
	res := "IDMap("

	if len(idMap) < 10 {
		nodes := make([]string, 0)
		for char, child := range idMap {
			nodes = append(nodes, fmt.Sprintf("`%v`:%d", string(char), child))
		}

		res += strings.Join(nodes, ",")
	} else {
		res += "len()=" + strconv.Itoa(len(idMap))
	}

	return res + ")"
}
