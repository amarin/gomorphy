package text

import (
	"golang.org/x/text/encoding/charmap"
)

// EncodeString translates string into 8-bit character set encoded bytes.
// Its a simple wrapper around charmap.Charmap encode routines.
func EncodeString(textString string, encoding *charmap.Charmap) (data []byte, err error) {
	encodedBytes, err := encoding.NewEncoder().Bytes([]byte(textString))
	if err != nil {
		return []byte{}, WrapErrorf(err, "cant encode string")
	}

	return encodedBytes, nil
}

// DecodeBytes translates 8-bit character set encoded bytes into string.
// Its a simple wrapper around charmap.Charmap Decoder.
func DecodeBytes(data []byte, encoding *charmap.Charmap) (decodedString string, err error) {
	decodedString, err = encoding.NewDecoder().String(string(data))
	if err != nil {
		return "", WrapErrorf(err, "cant decode bytes")
	}

	return decodedString, nil
}
