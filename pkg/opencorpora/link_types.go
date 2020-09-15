package opencorpora

// Список известных связей между леммами.
type LinkTypes struct {
	Type []*LinkType `xml:"type"`
}

func (l LinkTypes) MarshalBinary() (data []byte, err error) {
	var res []byte
	var tData []byte
	for _, t := range l.Type {
		if tData, err = t.MarshalBinary(); err != nil {
			break
		} else {
			res = append(res, tData...)
		}
	}
	if err != nil {
		err = WrapOpenCorporaError(err, "LinkTypes")
	}
	return res, nil
}

func (l *LinkTypes) UnmarshalBinary(data []byte) (err error) {
	for idx := range data {
		newType := new(LinkType)
		if err = newType.UnmarshalBinary(data[idx : idx+1]); err != nil {
			break
		}
		l.Type = append(l.Type, newType)
	}
	if err != nil {
		err = WrapOpenCorporaError(err, "LinkTypes")
	}
	return err
}
