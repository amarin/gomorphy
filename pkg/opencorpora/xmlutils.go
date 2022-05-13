package opencorpora

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

type Attr []xml.Attr

func (attributesList Attr) GetString(name string) (val string, err error) {
	for _, attr := range attributesList {
		if attr.Name.Local == name {
			return attr.Value, nil
		}
	}

	return "", fmt.Errorf("missed attr `%v`", name)
}

// GetInt returns 0 and error if named attribute missed or value conversion to int failed,
// otherwise int value and nil error returned.
func (attributesList Attr) GetInt(name string) (val int, err error) {
	var strValue string

	if strValue, err = attributesList.GetString(name); err != nil {
		return 0, err
	}

	if val, err = strconv.Atoi(strValue); err != nil {
		return 0, fmt.Errorf("attr `%v` value: %v", name, err)
	}

	return
}

func getAttr(name string, attributesList []xml.Attr) (val string, err error) {
	for _, attr := range attributesList {
		if attr.Name.Local == name {
			return attr.Value, nil
		}
	}

	return "", fmt.Errorf("missed attr `%v`", name)
}

func getIntAttr(name string, attributesList []xml.Attr) (val int, err error) {
	var strValue string

	if strValue, err = getAttr(name, attributesList); err != nil {
		return 0, err
	}

	if val, err = strconv.Atoi(strValue); err != nil {
		return 0, fmt.Errorf("attr `%v` value: %v", name, err)
	}
	return
}
