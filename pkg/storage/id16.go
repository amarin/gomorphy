package storage

const (
	maxUint8inUint16 = 0x00ff
	uint8bits        = 8
)

// ID16 represents uint16 index of elements.
type ID16 uint16

// Upper returns upper ID8 encapsulated into ID16 value.
func (id16value ID16) Upper() ID8 {
	return ID8(id16value >> uint8bits & maxUint8inUint16)
}

// Lower returns lower ID8 value encapsulated into ID16 value.
func (id16value ID16) Lower() ID8 {
	return ID8(maxUint8inUint16 & id16value)
}

// Combine8 makes an ID16 value putting upper ID8 onto upper bits and lower ID8 onto lower bits.
func Combine8(hi, lo ID8) ID16 {
	return ID16(hi)<<uint8bits | ID16(lo)
}
