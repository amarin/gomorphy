package opencorpora

type LinkList []*Link

// func (l LinkList) MarshalBinary() (data []byte, err error) {
// 	buffer := binutils.NewEmptyBuffer()
// 	for _, link := range l {
// 		if _, err := buffer.WriteObject(link, err); err != nil {
// 			break
// 		}
// 	}
// 	if err != nil {
// 		err = WrapOpenCorporaError(err, "Link")
// 	}
// 	return buffer.Bytes(), err
// }
//
// func (l *LinkList) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
// 	for {
// 		link := new(Link)
// 		if err = buffer.UnmarshalObject(link, nil); err != nil {
// 			break
// 		}
// 		*l = append(*l, link)
// 		if buffer.Len() == 0 {
// 			break
// 		}
// 	}
// 	if err != nil {
// 		err = WrapOpenCorporaError(err, "LinkList")
// 	}
// 	return err
// }
//
// func (l *LinkList) UnmarshalBinary(data []byte) error {
// 	return l.UnmarshalFromBuffer(binutils.NewBuffer(data))
// }
