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
	Parent   dag.ID    // item Parent ID
	ID       dag.ID    // item own ID
	Letter   rune      // item rune
	Variants VariantID // TagSet VariantID
}

// BinaryReadFrom reads item data using supplied binutils.BinaryReader.
// Implements binutils.BinaryReaderFrom.
func (i *Item) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var readUint32 uint32

	if readUint32, err = reader.ReadUint32(); err != nil {
		return err
	}
	i.Parent = dag.ID(readUint32)

	if readUint32, err = reader.ReadUint32(); err != nil {
		return err
	}
	i.ID = dag.ID(readUint32)

	if readUint32, err = reader.ReadUint32(); err != nil {
		return err
	}
	i.Letter = rune(readUint32)

	if readUint32, err = reader.ReadUint32(); err != nil {
		return err
	}
	i.Variants = VariantID(readUint32)

	return nil
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
