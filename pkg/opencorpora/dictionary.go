package opencorpora

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

// Dictionary represents OpenCorpora dictionary utf-8 XML root.
// Source file available by link http://opencorpora.org/?page=downloads,
// file format described by http://opencorpora.org/?page=export.
// Latest packed dictionary available in BZip2 at http://opencorpora.org/files/export/dict/dict.opcorpora.xml.bz2
// or as Zip at http://opencorpora.org/files/export/dict/dict.opcorpora.xml.zip.
type Dictionary struct {
	// VersionAttr contains dictionary version.
	VersionAttr float64 `xml:"version,attr"`
	// RevisionAttr provides revision number.
	RevisionAttr int `xml:"revision,attr"`
	// Grammemes provides grammar categories definitions.
	Grammemes *Grammemes `xml:"grammemes"`
	// Restrictions provides categories application restrictions.
	Restrictions *Restrictions `xml:"restrictions"`
	// Lemmata provides main dictionary lemma set.
	Lemmata *Lemmata `xml:"lemmata"`
	// Linktypes provides lemma link types.
	Linktypes *LinkTypes `xml:"link_types"`
	// Links provides lemma links.
	Links *Links `xml:"links"`
}

func (dictionary *Dictionary) processElem(parser *Parser, element xml.StartElement) (err error) {
	for _, attr := range element.Attr {
		switch attr.Name.Local {
		case "version":
			if dictionary.VersionAttr, err = strconv.ParseFloat(attr.Value, 32); err != nil {
				return fmt.Errorf("%w: parse: dictionary.version: `%v`: %v", Error, attr.Value, err)
			}
		case "revision":
			if parser.dictionary.RevisionAttr, err = strconv.Atoi(attr.Value); err != nil {
				return fmt.Errorf("%w: parse: dictionary.revision: `%v`: %v", Error, attr.Value, err)
			}
		}
	}

	return nil
}
