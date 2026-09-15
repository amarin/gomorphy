package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityAlphabetRoundtrip(t *testing.T) {
	a := IdentityAlphabet{}
	assert.Equal(t, "identity", a.Name())

	cases := []string{"", "abc", "кот", "поясней"}
	for _, s := range cases {
		enc, err := a.Encode(s)
		require.NoError(t, err)
		dec, err := a.Decode(enc)
		require.NoError(t, err)
		assert.Equal(t, s, dec)
	}
}

func TestDenseAlphabetRoundtrip(t *testing.T) {
	corpus := []string{"кот", "кота", "мышь", "дом"}
	for _, width := range []int{1, 2} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			a, err := NewDenseAlphabet(width, corpus)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("dense-%d", width), a.Name())

			for _, s := range corpus {
				enc, err := a.Encode(s)
				require.NoError(t, err)
				dec, err := a.Decode(enc)
				require.NoError(t, err)
				assert.Equal(t, s, dec)
			}
		})
	}
}

func TestDenseAlphabetReservedCodesNeverAssigned(t *testing.T) {
	corpus := []string{"кот", "кота", "мышь", "дом", "абвгдеёжзийклмнопрстуфхцчшщъыьэюя"}
	for _, width := range []int{1, 2} {
		a, err := NewDenseAlphabet(width, corpus)
		require.NoError(t, err)
		for r, code := range a.codeOf {
			assert.NotEqual(t, uint16(0), code, "rune %q must not get reserved code 0", r)
			assert.NotEqual(t, uint16(PayloadSeparator), code, "rune %q must not get reserved code 1 (PayloadSeparator)", r)
		}
	}
}

func TestDenseAlphabetWidth1OverflowsOnTooManyRunes(t *testing.T) {
	// 255 distinct runes - one more than width 1's capacity of 254
	// (256 possible byte values minus the 2 reserved codes 0 and 1).
	var corpus []string
	for r := rune(0x400); r < 0x400+255; r++ {
		corpus = append(corpus, string(r))
	}
	_, err := NewDenseAlphabet(1, corpus)
	assert.Error(t, err)
}

func TestDenseAlphabetWidth2CapacityIsMuchLarger(t *testing.T) {
	// The same 255-rune corpus that overflows width 1 must fit
	// comfortably in width 2 (capacity 64516).
	var corpus []string
	for r := rune(0x400); r < 0x400+255; r++ {
		corpus = append(corpus, string(r))
	}
	_, err := NewDenseAlphabet(2, corpus)
	assert.NoError(t, err)
}

func TestDenseAlphabetInvalidWidth(t *testing.T) {
	_, err := NewDenseAlphabet(3, []string{"a"})
	assert.Error(t, err)
}

func TestDenseAlphabetDecodeRejectsMisalignedLength(t *testing.T) {
	a, err := NewDenseAlphabet(2, []string{"кот"})
	require.NoError(t, err)
	_, err = a.Decode([]byte{1, 2, 3}) // length 3, not a multiple of width 2
	assert.Error(t, err)
}

func TestDenseAlphabetEncodeRejectsUnknownRune(t *testing.T) {
	a, err := NewDenseAlphabet(1, []string{"кот"})
	require.NoError(t, err)
	_, err = a.Encode("мышь") // none of м/ы/ш/ь are in the "кот" corpus
	assert.Error(t, err)
}

func TestDenseAlphabetDeterministic(t *testing.T) {
	corpus := []string{"кот", "мышь", "дом", "яснее"}
	for _, width := range []int{1, 2} {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			a1, err := NewDenseAlphabet(width, corpus)
			require.NoError(t, err)
			a2, err := NewDenseAlphabet(width, corpus)
			require.NoError(t, err)
			assert.Equal(t, a1.codeOf, a2.codeOf)
		})
	}
}

func TestDenseAlphabetEncodedBytesNeverReserved(t *testing.T) {
	corpus := []string{"кот", "кота", "мышь", "дом", "яснее", "абвгдеёжзийклмнопрстуфхцчшщъыьэюя", "ABCxyz123"}
	for _, width := range []int{1, 2} {
		a, err := NewDenseAlphabet(width, corpus)
		require.NoError(t, err)
		for _, s := range corpus {
			enc, err := a.Encode(s)
			require.NoError(t, err)
			for _, b := range enc {
				assert.NotEqual(t, byte(0), b, "encoded byte must never be 0x00 (guide sentinel), width=%d, s=%q", width, s)
				assert.NotEqual(t, byte(PayloadSeparator), b, "encoded byte must never be PayloadSeparator, width=%d, s=%q", width, s)
			}
		}
	}
}

func TestDenseAlphabetDecodeRejectsReservedOrOutOfRangeCode(t *testing.T) {
	a1, err := NewDenseAlphabet(1, []string{"кот"})
	require.NoError(t, err)
	_, err = a1.Decode([]byte{0})
	assert.Error(t, err)
	_, err = a1.Decode([]byte{1})
	assert.Error(t, err)
	_, err = a1.Decode([]byte{255}) // in-range byte, but no rune assigned to it in this small corpus
	assert.Error(t, err)

	a2, err := NewDenseAlphabet(2, []string{"кот"})
	require.NoError(t, err)
	_, err = a2.Decode([]byte{0, 0}) // reserved byte in hi position
	assert.Error(t, err)
	_, err = a2.Decode([]byte{2, 0}) // reserved byte in lo position
	assert.Error(t, err)
}
