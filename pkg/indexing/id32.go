package indexing

const (
	maxUint16inUint32 = 0x0000ffff
	uint16bits        = 16
)

// ID32 represents uint32 index of elements.
type ID32 uint32

// Upper16 returns upper ID16 encapsulated into Id32 value.
func (id32value ID32) Upper16() ID16 {
	return ID16(id32value >> uint16bits & maxUint16inUint32)
}

// Lower16 returns Lower16 value encapsulated into Id32 value.
func (id32value ID32) Lower16() ID16 {
	return ID16(maxUint16inUint32 & id32value)
}

// Combine16 makes an ID32 value putting upper ID16 onto upper bits and lower ID16 onto lower bits.
func Combine16(upper ID16, lower ID16) ID32 {
	return ID32(upper)<<uint16bits | ID32(lower)
}
