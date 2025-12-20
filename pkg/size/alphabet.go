package size

// Alphabet задаёт максимальную ёмкость алфавита
type Alphabet interface {
	uint8 | uint16
}
