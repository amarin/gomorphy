package index

import (
	"errors"
	"strconv"
	"strings"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/dag"
)

var Error = errors.New("index")

// Item represents DAG index item.
type Item struct {
	Parent   dag.ID       // item Parent ID
	ID       dag.ID       // item own ID
	Letter   rune         // item rune
	Variants CollectionID // TagSet CollectionID
}

// BinaryReadFrom reads item data using supplied binutils.BinaryReader.
func (i *Item) BinaryReadFrom(reader *binutils.BinaryReader) (n int64, err error) {
	var readUint32 uint32
	n = 0

	if readUint32, err = reader.ReadUint32(); err != nil {
		return n, err
	}
	i.Parent = dag.ID(readUint32)
	n += binutils.Uint32size

	if readUint32, err = reader.ReadUint32(); err != nil {
		return n, err
	}
	i.ID = dag.ID(readUint32)
	n += binutils.Uint32size

	if readUint32, err = reader.ReadUint32(); err != nil {
		return n, err
	}
	i.Letter = rune(readUint32)
	n += binutils.RuneSize

	if readUint32, err = reader.ReadUint32(); err != nil {
		return n, err
	}
	i.Variants = CollectionID(readUint32)
	n += binutils.Uint32size

	return n, nil
}

// BinaryWriteTo writes item data using binutils.BinaryWriter.
func (i *Item) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint32(uint32(i.Parent)); err != nil {
		return err
	}

	if err = writer.WriteUint32(uint32(i.ID)); err != nil {
		return err
	}

	if err = writer.WriteRune(i.Letter); err != nil {
		return err
	}

	if err = writer.WriteUint32(uint32(i.Variants)); err != nil {
		return err
	}

	return nil
}

// String returns string representation of DAG=index item. Implements fmt.Stringer.
func (i Item) String() string {
	return string(i.Letter) + strings.Join([]string{
		strconv.Itoa(int(i.ID)),
		strconv.Itoa(int(i.Parent)),
		strconv.Itoa(int(i.Variants)),
	}, "_")
}
