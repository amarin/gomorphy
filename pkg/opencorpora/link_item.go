package opencorpora

// Связь между леммами. Определяет возможные преобразования из, например, заданного существительного в глагол.
type Link struct {
	IdAttr   uint32 `xml:"id,attr"`
	FromAttr uint32 `xml:"from,attr"`
	ToAttr   uint32 `xml:"to,attr"`
	TypeAttr uint32 `xml:"type,attr"`
}

// func (l Link) MarshalBinary() (data []byte, err error) {
// 	buf := binutils.NewEmptyBuffer()
// 	_, err = buf.WriteUint32(l.IdAttr, err)
// 	_, err = buf.WriteUint32(l.FromAttr, err)
// 	_, err = buf.WriteUint32(l.ToAttr, err)
// 	_, err = buf.WriteUint32(l.TypeAttr, err)
// 	if err != nil {
// 		err = WrapOpenCorporaError(err, "Link")
// 	}
// 	return buf.Bytes(), err
// }
//
// func (l *Link) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
// 	err = buffer.ReadUint32(&l.IdAttr, err)
// 	err = buffer.ReadUint32(&l.FromAttr, err)
// 	err = buffer.ReadUint32(&l.ToAttr, err)
// 	err = buffer.ReadUint32(&l.TypeAttr, err)
// 	if err != nil {
// 		err = WrapOpenCorporaError(err, "Link")
// 	}
// 	return err
// }
//
// func (l *Link) UnmarshalBinary(data []byte) error {
// 	return l.UnmarshalFromBuffer(binutils.NewBuffer(data))
// }
