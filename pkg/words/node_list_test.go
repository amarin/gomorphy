package words_test

import (
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/internal/grammeme"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/pkg/words"
)

// TestNodeList_RequiredBits checks NodeList.RequiredBits detects valid bytes sizing for lists of different sizes.
func TestNodeList_RequiredBits(t *testing.T) {
	for _, tt := range []struct {
		name     string
		listSize uint64
		want     binutils.BitsPerIndex
		wantErr  bool
	}{
		{"uint8_for_zero_sized_list",
			0, binutils.Use8bit, false},
		{"uint8_for_max_uint8_minus_one",
			math.MaxUint8 - 1, binutils.Use8bit, false},
		{"uint16_for_max_uint8_as_nil_parent_requires_extra_value",
			math.MaxUint8, binutils.Use16bit, false},
		{"uint16_for_max_uint16_minus_one",
			math.MaxUint16 - 1, binutils.Use16bit, false},
		{"uint32_for_max_uint16_as_nil_parent_requires_extra_value",
			math.MaxUint16, binutils.Use32bit, false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			nodeList := make(words.NodeList, tt.listSize)
			if got, err := nodeList.RequiredBits(); (err != nil) != tt.wantErr {
				t.Fatalf("RequiredBits() error = %v, wantErr %v", err, tt.wantErr)
			} else if got != tt.want {
				t.Errorf("RequiredBits() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNodeList_WriteIndexLen check node list write expected sizing bytes as index len.
func TestNodeList_WriteIndexLen(t *testing.T) {
	for _, tt := range []struct {
		name      string
		list      *words.NodeList
		usingBits binutils.BitsPerIndex
		wantBytes string
		wantErr   bool
	}{
		{"uint8",
			words.NewNodeList(), binutils.Use8bit, "00", false},
		{"uint16",
			words.NewNodeList(), binutils.Use16bit, "0000", false},
		{"uint32",
			words.NewNodeList(), binutils.Use32bit, "00000000", false},
		{"uint64",
			words.NewNodeList(), binutils.Use64bit, "0000000000000000", false},
		{"wrong_sizing",
			words.NewNodeList(), binutils.BitsPerIndex(7), "", true},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			buffer := binutils.NewEmptyBuffer()
			if _, err := tt.list.WriteIndexLen(buffer, tt.usingBits); (err != nil) != tt.wantErr {
				t.Errorf("WriteIndexLen() error = %v, wantErr %v", err, tt.wantErr)
			} else if hex.EncodeToString(buffer.Bytes()) != tt.wantBytes {
				t.Errorf(
					"WriteIndexLen() \nwritten\n\t%v, \nwant\n\t%v",
					hex.EncodeToString(buffer.Bytes()), tt.wantBytes)
			}
		})
	}
}

// TestNodeList_WriteNoParent checks NodeList uses MaxUintX for writing nil parent for different index sizing.
func TestNodeList_WriteNoParent(t *testing.T) {
	for _, tt := range []struct {
		name      string
		usingBits binutils.BitsPerIndex
		wantBytes string
		wantErr   bool
	}{
		{"unit8", binutils.Use8bit, "ff", false},
		{"unit16", binutils.Use16bit, "ffff", false},
		{"unit32", binutils.Use32bit, "ffffffff", false},
		{"unit64", binutils.Use64bit, "ffffffffffffffff", false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			list := words.NewNodeList()
			buffer := binutils.NewEmptyBuffer()
			if _, err := list.WriteNoParent(buffer, tt.usingBits); (err != nil) != tt.wantErr {
				t.Errorf("WriteNoParent() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if hex.EncodeToString(buffer.Bytes()) != tt.wantBytes {
				t.Errorf(
					"WriteIndexLen() \nwritten\n\t%v, \nwant\n\t%v",
					hex.EncodeToString(buffer.Bytes()), tt.wantBytes)
			}
		})
	}
}

// TestNodeList_MarshalParentIdx check NodeList marshalling node parent as proper value either if parent nil or
// parent is known item in NodeList.
// nolint:funlen
func TestNodeList_MarshalParentIdx(t *testing.T) {
	commonParent := words.NewMappingNode(nil, ' ')
	unknownParent := words.NewMappingNode(nil, ' ')
	mapping := *words.NewNodePointersMap()
	mapping[commonParent] = 0x10

	for _, tt := range []struct {
		name      string
		n         *words.Node
		bits      binutils.BitsPerIndex
		wantBytes string
		wantErr   bool
	}{
		{"uint8_nil_parent", words.NewMappingNode(nil, ' '),
			binutils.Use8bit, "ff", false},
		{"uint8_have_parent", words.NewMappingNode(commonParent, ' '),
			binutils.Use8bit, "10", false},
		{"uint16_nil_parent", words.NewMappingNode(nil, ' '),
			binutils.Use16bit, "ffff", false},
		{"uint16_have_parent", words.NewMappingNode(commonParent, ' '),
			binutils.Use16bit, "0010", false},
		{"uint32_nil_parent", words.NewMappingNode(nil, ' '),
			binutils.Use32bit, "ffffffff", false},
		{"uint32_have_parent", words.NewMappingNode(commonParent, ' '),
			binutils.Use32bit, "00000010", false},
		{"uint64_nil_parent", words.NewMappingNode(nil, ' '),
			binutils.Use64bit, "ffffffffffffffff", false},
		{"uint64_have_parent", words.NewMappingNode(commonParent, ' '),
			binutils.Use64bit, "0000000000000010", false},
		{"uint64_unknown_parent", words.NewMappingNode(unknownParent, ' '),
			binutils.Use64bit, "", true},
		{"uint32_unknown_parent", words.NewMappingNode(unknownParent, ' '),
			binutils.Use32bit, "", true},
		{"uint16_unknown_parent", words.NewMappingNode(unknownParent, ' '),
			binutils.Use16bit, "", true},
		{"uint8_unknown_parent", words.NewMappingNode(unknownParent, ' '),
			binutils.Use8bit, "", true},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			list := words.NewNodeList()
			buffer := binutils.NewEmptyBuffer()
			if _, err := list.MarshalParentIdx(buffer, tt.n, tt.bits, &mapping); (err != nil) != tt.wantErr {
				t.Errorf("WriteNoParent() error = %v, wantErr %v\nmap:\n%v",
					err, tt.wantErr, strings.Join(mapping.PointersStrings(), "\n"))
			} else if err == nil && hex.EncodeToString(buffer.Bytes()) != tt.wantBytes {
				t.Errorf(
					"WriteIndexLen() \nwritten\n\t%v, \nwant\n\t%v",
					hex.EncodeToString(buffer.Bytes()), tt.wantBytes)
			}
		})
	}
}

func TestNodeList_ReadIndexLen(t *testing.T) {
	for _, tt := range []struct {
		name      string
		listLen   uint64
		usingBits binutils.BitsPerIndex
		wantBytes string
		wantErr   bool
	}{
		{"uint8_0", 0, binutils.Use8bit, "00", false},
		{"uint16_0", 0, binutils.Use16bit, "0000", false},
		{"uint32_0", 0, binutils.Use32bit, "00000000", false},
		{"uint64_0", 0, binutils.Use64bit, "0000000000000000", false},
		{"uint8_0_extra_bytes", 0, binutils.Use8bit, "00ff", false},
		{"uint16_0_extra_bytes", 0, binutils.Use16bit, "0000ff", false},
		{"uint32_0_extra_bytes", 0, binutils.Use32bit, "00000000ff", false},
		{"uint64_0_extra_bytes", 0, binutils.Use64bit, "0000000000000000ff", false},
		{"uint8_1", 1, binutils.Use8bit, "01", false},
		{"uint16_1", 1, binutils.Use16bit, "0001", false},
		{"uint32_1", 1, binutils.Use32bit, "00000001", false},
		{"uint64_1", 1, binutils.Use64bit, "0000000000000001", false},
		{"uint8_1_extra_bytes", 1, binutils.Use8bit, "01ff", false},
		{"uint16_1_extra_bytes", 1, binutils.Use16bit, "0001ff", false},
		{"uint32_1_extra_bytes", 1, binutils.Use32bit, "00000001ff", false},
		{"uint64_1_extra_bytes", 1, binutils.Use64bit, "0000000000000001ff", false},
		{"uint8_max", math.MaxUint8, binutils.Use8bit, "ffff", false},
		{"uint16_max", math.MaxUint16, binutils.Use16bit, "ffffff", false},
		{"uint32_max", math.MaxUint32, binutils.Use32bit, "ffffffffff", false},
		{"uint64_max", math.MaxUint64, binutils.Use64bit, "ffffffffffffffffff", false},
		{"uint16_wrong_bytes", math.MaxUint16, binutils.Use16bit, "ff", true},
		{"uint32_wrong_bytes", math.MaxUint32, binutils.Use32bit, "ffffff", true},
		{"uint64_wrong_bytes", math.MaxUint64, binutils.Use64bit, "ffffffffffffff", true},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			buffer := binutils.NewEmptyBuffer()
			list := words.NewNodeList()
			data, err := hex.DecodeString(tt.wantBytes)
			if err != nil {
				t.Fatalf("cant prepare test data: %v", err)
			}
			if _, err := buffer.WriteBytes(data); err != nil {
				t.Fatalf("cant load test data to buffer: %v", err)
			}
			if got, err := list.ReadIndexLen(buffer, tt.usingBits); (err != nil) != tt.wantErr {
				t.Errorf("ReadIndexLen() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil && got != tt.listLen {
				t.Errorf("ReadIndexLen() got = %v, want %v", got, tt.listLen)
			}
		})
	}
}

func TestNodeList_ReadParentIdx(t *testing.T) {
	commonParent := words.NewMappingNode(nil, ' ')
	mapping := *words.NewNodePointersMap()
	mapping[commonParent] = 0x10

	for _, tt := range []struct {
		name      string
		bits      binutils.BitsPerIndex
		wantBytes string
		parentIdx uint64
		wantErr   bool
	}{
		{"uint8_nil_parent",
			binutils.Use8bit, "ff", 255, false},
		{"uint8_have_parent",
			binutils.Use8bit, "10", 16, false},
		{"uint16_nil_parent",
			binutils.Use16bit, "ffff", math.MaxUint16, false},
		{"uint16_have_parent",
			binutils.Use16bit, "0010", 16, false},
		{"uint32_nil_parent",
			binutils.Use32bit, "ffffffff", math.MaxUint32, false},
		{"uint32_have_parent",
			binutils.Use32bit, "00000010", 16, false},
		{"uint64_nil_parent",
			binutils.Use64bit, "ffffffffffffffff", math.MaxUint64, false},
		{"uint64_have_parent",
			binutils.Use64bit, "0000000000000010", 16, false},
		{"uint64_wrong_bytes",
			binutils.Use64bit, "00000000000000", 0, true},
		{"uint32_wrong_bytes",
			binutils.Use32bit, "ffffff", 0, true},
		{"uint16_wrong_bytes",
			binutils.Use16bit, "ff", 0, true},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			list := words.NewNodeList()
			buffer := binutils.NewEmptyBuffer()
			var target uint64
			data, err := hex.DecodeString(tt.wantBytes)
			if err != nil {
				t.Fatalf("cant prepare test data: %v", err)
			}
			if _, err := buffer.WriteBytes(data); err != nil {
				t.Fatalf("cant load test data to buffer: %v", err)
			}
			if err := list.ReadParentIdx(buffer, &target, tt.bits); (err != nil) != tt.wantErr {
				t.Errorf("ReadParentIdx() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil && target != tt.parentIdx {
				t.Errorf("ReadParentIdx() got %v, expect %v", target, tt.parentIdx)
			}
		})
	}
}

func TestNodeList_MakeReverseIndex(t *testing.T) {
	for _, tt := range []struct {
		name        string
		usingBits   binutils.BitsPerIndex
		expectedKey uint64
		wantErr     bool
	}{
		{"8bit", binutils.Use8bit, math.MaxUint8, false},
		{"16bit", binutils.Use16bit, math.MaxUint16, false},
		{"32bit", binutils.Use32bit, math.MaxUint32, false},
		{"64bit", binutils.Use64bit, math.MaxUint64, false},
		{"7bit_not_supported", binutils.BitsPerIndex(7), 0, true},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			list := words.NewNodeList()
			got, err := list.MakeReverseIndex(tt.usingBits)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MakeReverseIndex() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				return
			}

			if _, ok := got[tt.expectedKey]; !ok {
				t.Errorf("MakeReverseIndex() no expected key %v", tt.expectedKey)
			}
		})
	}
}

// TestNodeList_MarshalBinary checks NodeList binary marshalling works as expected in different situaltions.
// nolint:funlen
func TestNodeList_MarshalBinary(t *testing.T) {
	POST := &grammeme.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammeme.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammeme.NewIndex(*POST, *NOUN)

	cyrillicA := words.NewMappingNode(nil, 'а')
	latinF := words.NewMappingNode(nil, 'f')
	cyrillicB := words.NewMappingNode(cyrillicA, 'б')

	cyrillicV := words.NewMappingNode(cyrillicA, 'в')
	if err := cyrillicV.AddGrammemes(grammemes.NewList(indexA, POST)); err != nil {
		t.Fatalf("cant prepare test node: %v", err)
	}

	cyrillicG := words.NewMappingNode(nil, 'г')
	if err := cyrillicG.AddGrammemes(grammemes.NewList(indexA, POST, NOUN)); err != nil {
		t.Fatalf("cant prepare test node: %v", err)
	}

	for _, tt := range []struct {
		name     string
		list     *words.NodeList
		wantData string
		wantErr  bool
	}{
		{"empty",
			&words.NodeList{},
			"0800", false},
		{"single_cyrillic_a",
			&words.NodeList{cyrillicA},
			"0801ffc100", false},
		{"single_latin_f",
			&words.NodeList{latinF},
			"0801ff6600", false},
		{"cyrillic_ab",
			&words.NodeList{cyrillicA, cyrillicB},
			"0802ffc10000c200", false},
		{"cyrillic_ab_and_latin_f",
			&words.NodeList{cyrillicA, cyrillicB, latinF},
			"0803ffc10000c200ff6600", false},
		{"miss_parent_error",
			&words.NodeList{cyrillicV},
			"", true},
		{"no_parent_but_grammemes",
			&words.NodeList{cyrillicG},
			"0801ffc701020001", false},
		{"with_grammemes",
			&words.NodeList{cyrillicA, cyrillicG, cyrillicV},
			"0803ffc100ffc70102000100d7010100", false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			if gotData, err := tt.list.MarshalBinary(); (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil && hex.EncodeToString(gotData) != tt.wantData {
				t.Errorf("MarshalBinary() gotData = %v, want %v", hex.EncodeToString(gotData), tt.wantData)
			}
		})
	}
}

// TestNodeList_UnmarshalFromBufferWithIndex checks NodeList.UnmarshalFromBufferWithIndex works as expected.
// nolint:funlen
func TestNodeList_UnmarshalFromBufferWithIndex(t *testing.T) {
	POST := &grammeme.Grammeme{ParentAttr: "", Name: "POST", Alias: "", Description: ""}
	NOUN := &grammeme.Grammeme{ParentAttr: "POST", Name: "NOUN", Alias: "", Description: ""}
	indexA := grammeme.NewIndex(*POST, *NOUN)

	cyrillicA := words.NewMappingNode(nil, 'а')
	latinF := words.NewMappingNode(nil, 'f')
	cyrillicB := words.NewMappingNode(cyrillicA, 'б')

	cyrillicV := words.NewMappingNode(cyrillicA, 'в')
	if err := cyrillicV.AddGrammemes(grammemes.NewList(indexA, POST)); err != nil {
		t.Fatalf("cant prepare test node: %v", err)
	}

	cyrillicG := words.NewMappingNode(nil, 'г')
	if err := cyrillicG.AddGrammemes(grammemes.NewList(indexA, POST, NOUN)); err != nil {
		t.Fatalf("cant prepare test node: %v", err)
	}

	for _, tt := range []struct {
		name     string
		list     *words.NodeList
		wantData string
		wantErr  bool
	}{
		{"empty",
			&words.NodeList{},
			"0800", false},
		{"single_cyrillic_a",
			&words.NodeList{cyrillicA},
			"0801ffc100", false},
		{"single_latin_f",
			&words.NodeList{latinF},
			"0801ff6600", false},
		{"cyrillic_ab",
			&words.NodeList{cyrillicA, cyrillicB},
			"0802ffc10000c200", false},
		{"cyrillic_ab_and_latin_f",
			&words.NodeList{cyrillicA, cyrillicB, latinF},
			"0803ffc10000c200ff6600", false},
		{"no_parent_but_grammemes",
			&words.NodeList{cyrillicG},
			"0801ffc701020001", false},
		{"with_grammemes",
			&words.NodeList{cyrillicA, cyrillicG, cyrillicV},
			"0803ffc100ffc70102000100d7010100", false},
	} {
		tt := tt // pin tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt // pin tt
			list := words.NewNodeList()
			data, err := hex.DecodeString(tt.wantData)
			buffer := binutils.NewBuffer(data)
			if err != nil {
				t.Fatalf("cant prepare binary data: %v", err)
			} else if err := list.UnmarshalFromBufferWithIndex(buffer, indexA); (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			} else if list.Len() != tt.list.Len() {
				t.Errorf("list len mismatch, got %d expected %d", list.Len(), tt.list.Len())
			}
		})
	}
}
