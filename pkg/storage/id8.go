package storage

// ID8 represents uint8 ID of elements.
type ID8 uint8

// Uint8 returns uint8 value of ID8. Implements binutils.BinaryUint8.
func (id8 ID8) Uint8() uint8 {
	return uint8(id8)
}
