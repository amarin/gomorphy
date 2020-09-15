package opencorpora

// Список известных связей между леммами
type Links struct {
	Items LinkList `xml:"link"`
}

// func (l Links) MarshalBinary() (data []byte, err error) {
// 	return l.Items.MarshalBinary()
// }
//
// func (l *Links) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
// 	return buffer.UnmarshalObject(&l.Items, nil)
// }
//
// func (l *Links) UnmarshalBinary(data []byte) error {
// 	return l.UnmarshalFromBuffer(binutils.NewBuffer(data))
// }
