package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"testing"
	"unsafe"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDAWGFind(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{
		"кот":  7,
		"кота": 42,
	})
	d := NewDAWG(dict, guide)

	require.Equal(t, uint32(7), d.Find("кот"))
	require.Equal(t, uint32(42), d.Find("кота"))
	require.Equal(t, uint32(0), d.Find("котик"))
	require.Equal(t, uint32(0), d.Find(""))
}

func TestDAWGFollowByteMiss(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{"кота": 42})
	d := NewDAWG(dict, guide)

	require.NotZero(t, d.FollowByte("кота"[0], 0))

	index := d.Follow("кот", 0)
	require.NotZero(t, index)

	require.Equal(t, uint32(0), d.FollowByte(0xB5, index), "второй байт 'е' (0xB5) не путь 'кота' (0xB0)")
	require.Equal(t, uint32(0), d.FollowByte(0x01, 0), "нет перехода по несуществующему байту")
	require.Equal(t, uint32(0), d.FollowRune('ж', 0), "нет слова, начинающегося с 'ж'")
}

func TestDAWGFollowRuneMatchesFollow(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{"кота": 42})
	d := NewDAWG(dict, guide)

	index := d.FollowRune('к', 0)
	require.NotZero(t, index)
	index = d.FollowRune('о', index)
	require.NotZero(t, index)
	index = d.FollowRune('т', index)
	require.NotZero(t, index)
	index = d.FollowRune('а', index)
	require.NotZero(t, index)
	require.True(t, d.HasValue(index))
	require.Equal(t, uint32(42), d.Value(index))

	require.NotZero(t, d.Follow("кота", 0))
}

func TestDAWGValuesForIndex(t *testing.T) {
	payload := []byte{0x00, 0x2a, 0x00, 0x07}
	key := "кот" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString(payload)
	dict, guide := testdawg.Build(map[string]uint32{key: 5})
	d := NewDAWG(dict, guide)

	wordIndex := d.Follow("кот", 0)
	require.NotZero(t, wordIndex)
	sepIndex := d.FollowByte(PayloadSeparator, wordIndex)
	require.NotZero(t, sepIndex)

	values := d.ValuesForIndex(sepIndex)
	require.Len(t, values, 1)
	assert.Equal(t, payload, values[0])
}

func TestDAWGReadFromStream(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{"кота": 42})

	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint32(len(dict))))
	for _, u := range dict {
		require.NoError(t, binary.Write(&buf, binary.LittleEndian, u))
	}
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint32(len(guide)/2)))
	buf.Write(guide)

	d, err := ReadDAWG(&buf)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, uint32(42), d.Find("кота"))
}

func TestDAWGReadFromStreamMatchesMarshal(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{"кота": 42})

	d, err := ReadDAWG(bytes.NewReader(testdawg.Marshal(dict, guide)))
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, uint32(42), d.Find("кота"))
}

func collectChildLabels(d *DAWG, index uint32) []byte {
	var out []byte
	d.ForEachChild(index, func(label byte, next uint32) {
		out = append(out, label)
	})
	return out
}

func sortedBytes(bs []byte) []byte {
	out := append([]byte(nil), bs...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func TestDAWGForEachChild(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{
		"cat" + string(PayloadSeparator) + "AAAA": 1,
		"car" + string(PayloadSeparator) + "AAAA": 2,
		"dog" + string(PayloadSeparator) + "AAAA": 3,
	})
	d := NewDAWG(dict, guide)

	assert.Equal(t, []byte{'c', 'd'}, sortedBytes(collectChildLabels(d, 0)))

	cIndex := d.Follow("c", 0)
	require.NotZero(t, cIndex)
	assert.Equal(t, []byte{'a'}, collectChildLabels(d, cIndex))

	caIndex := d.Follow("ca", 0)
	require.NotZero(t, caIndex)
	assert.Equal(t, []byte{'r', 't'}, sortedBytes(collectChildLabels(d, caIndex)))

	carIndex := d.Follow("car", 0)
	require.NotZero(t, carIndex)
	assert.Equal(t, []byte{PayloadSeparator}, collectChildLabels(d, carIndex))

	sepIndex := d.FollowByte(PayloadSeparator, carIndex)
	require.NotZero(t, sepIndex)
	require.Len(t, d.ValuesForIndex(sepIndex), 1)
}

func TestDAWGForEachChildNoGuide(t *testing.T) {
	d := NewDAWG([]uint32{5, 6, 7}, nil)
	assert.Empty(t, collectChildLabels(d, 0))
}

func TestDAWGBytesRoundtrip(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{
		"кот" + string(PayloadSeparator) + "AAAA":  5,
		"кота" + string(PayloadSeparator) + "BBBB": 6,
	})
	d := NewDAWG(dict, guide)

	rd, err := ParseDAWG(d.Bytes())
	require.NoError(t, err)
	assert.Equal(t, d.dict, rd.dict)
	assert.Equal(t, d.guide, rd.guide)
	assert.Equal(t, d.Find("кота"), rd.Find("кота"))
}

func TestParseDAWGZeroCopy(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{"кота": 42})
	raw := NewDAWG(dict, guide).Bytes()

	d, err := ParseDAWG(raw)
	require.NoError(t, err)

	require.NotEmpty(t, d.dict)
	rawData := uintptr(unsafe.Pointer(unsafe.SliceData(raw)))
	dictData := uintptr(unsafe.Pointer(unsafe.SliceData(d.dict)))
	assert.Equal(t, rawData+4, dictData, "dict должен алиасить mmap, а не копироваться")

	guideData := uintptr(unsafe.Pointer(unsafe.SliceData(d.guide)))
	require.NoError(t, err)
	guideOffset := int64(4 + len(dict)*4 + 4)
	assert.Equal(t, rawData+uintptr(guideOffset), guideData)
}

func TestParseDAWGCopiesMisaligned(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{"кота": 42})
	raw := NewDAWG(dict, guide).Bytes()
	shifted := append([]byte{0}, raw...)[1:]

	d, err := ParseDAWG(shifted)
	require.NoError(t, err)
	assert.Equal(t, dict, d.dict)
	assert.Equal(t, guide, d.guide)
}

func TestParseDAWGErrors(t *testing.T) {
	_, err := ParseDAWG(nil)
	require.Error(t, err)

	_, err = ParseDAWG([]byte{1, 2, 3})
	require.Error(t, err)

	_, err = ParseDAWG([]byte{5, 0, 0, 0, 1, 2, 3})
	require.Error(t, err)
}

func TestDAWGWalk(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{
		"кот" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 0, 0, 1}):  0,
		"кот" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 1, 0, 0}):  0,
		"кота" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 2, 0, 0}): 0,
		"мышь" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 3, 0, 0}): 0,
	})
	d := NewDAWG(dict, guide)

	found := map[string][][]byte{}
	d.Walk(func(key string, values [][]byte) {
		found[key] = append(found[key], values...)
	})

	require.Len(t, found, 3, "3 distinct words: кот, кота, мышь")
	require.Len(t, found["кот"], 2, "кот has 2 payload values (homonym)")
	assert.ElementsMatch(t, [][]byte{{0, 0, 0, 1}, {0, 1, 0, 0}}, found["кот"])
	require.Len(t, found["кота"], 1)
	assert.Equal(t, []byte{0, 2, 0, 0}, found["кота"][0])
	require.Len(t, found["мышь"], 1)
	assert.Equal(t, []byte{0, 3, 0, 0}, found["мышь"][0])
}

func TestDAWGWalkEmpty(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{})
	d := NewDAWG(dict, guide)

	calls := 0
	d.Walk(func(key string, values [][]byte) { calls++ })
	assert.Equal(t, 0, calls)
}
