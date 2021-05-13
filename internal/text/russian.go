package text

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"golang.org/x/text/encoding/charmap"
)

// ErrText indicates any errors with RussianText processing happened.
var ErrText = errors.New("text")

// RussianText stores word or phrase in russian language.
// It stores data as ordinal string in memory and uses 1-byte per character encoding to store dame data in file.
type RussianText string

// String returns string representation of russian text.
// Implements fmt.Stringer.
func (w RussianText) String() string {
	return string(w)
}

// Len returns RussianText string length in characters(runes). To detect internal length use len().
func (w RussianText) Len() int {
	return len([]rune(w))
}

// ReadFrom loads text data from specified io.Reader instance.
// Returns taken bytes count and any error if occurs. Implements io.ReaderFrom.
func (w *RussianText) ReadFrom(r io.Reader) (n int64, err error) {
	currentByte := make([]byte, 1)
	encodedBytes := make([]byte, 0)

	for {
		if _, err = r.Read(currentByte); err != nil {
			return n, fmt.Errorf("%v: read next byte: %w", ErrText, err)
		}
		n++

		if currentByte[0] == 0 {
			break
		}

		encodedBytes = append(encodedBytes, currentByte[0])
	}

	decodedString, err := DecodeBytes(encodedBytes, charmap.KOI8R)
	if err != nil {
		return n, fmt.Errorf("%v: translate: %w", ErrText, err)
	}

	*w = RussianText(decodedString)

	return n, nil
}

// EncodeTo makes byte string using specified encoding.
func (w RussianText) EncodeTo(charMap *charmap.Charmap) (data []byte, err error) {
	if data, err = EncodeString(string(w), charMap); err != nil {
		return nil, fmt.Errorf("%v: encode %v: %w", ErrText, charMap.String(), err)
	}

	return append(data, 0), nil
}

// MarshalBinary makes 1-byte encoded string using KOI8R.
// To
// Implements encoding.BinaryMarshaler.
func (w RussianText) MarshalBinary() (data []byte, err error) {
	if data, err = w.EncodeTo(charmap.KOI8R); err != nil {
		return nil, err
	}

	return append(data, 0), nil
}

// WriteTo writes text bytes into specified writer.
// Returns written bytes count and any error if occurs.
// Implements io.WriterTo.
func (w *RussianText) WriteTo(writer io.Writer) (n int64, err error) {
	var (
		textBytes    []byte
		bytesWritten int
	)

	if textBytes, err = w.MarshalBinary(); err != nil {
		return 0, err
	}

	if bytesWritten, err = writer.Write(textBytes); err != nil {
		return int64(bytesWritten), fmt.Errorf("%v: write: %w", ErrText, err)
	}

	return int64(bytesWritten), nil
}

// UnmarshalBinary reads text data from bytes string.
// As expected it will used rare it uses w.ReadFrom with intermediate bytes buffer as reader instance.
// Implements encoding.BinaryUnmarshaler.
func (w *RussianText) UnmarshalBinary(data []byte) (err error) {
	_, err = w.ReadFrom(bytes.NewReader(data))

	return err
}
