package opencorpora

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/internal/parse/xmlstream"
	"github.com/amarin/gomorphy/pkg/dag"
	"github.com/amarin/gomorphy/pkg/tag"
)

const defaultLogAverageEachSeconds = 10

// ErrControlledStop raised when limit to parse items set.
var ErrControlledStop = fmt.Errorf("%w: controlled stop", Error)

type Parser struct {
	xmlstream.Parser

	logging.Logger
	index      any
	dictionary *Dictionary

	currentGrammeme *tag.Tag
	currentLemma    *Lemma
	currentForm     *WordForm

	parserStarted   time.Time
	reportAfter     time.Time
	parsedLemmas    int // parsed lemma's items
	parsedForms     int // parsed lemma forms
	logAverageSpeed int // report average parse speed each logAverageSpeed seconds

	// max forms to parse.
	// If !=0 parser will be stopped with ErrControlledStop after maximum maxLemmas lemmas forms parsing.
	maxLemmas int
}

func newParser(indexInstance any) *Parser {
	switch indexInstance.(type) {
	case Index, newIndexType:
	default:
		panic("unknown index type, must be either Index or newIndexType")
	}

	dictionary := &Dictionary{
		VersionAttr:  0,
		RevisionAttr: 0,
		Grammemes:    nil,
		Restrictions: nil,
		Lemmata:      nil,
		Linktypes:    nil,
		Links:        nil,
	}

	parser := &Parser{
		Parser:          *xmlstream.New(),
		Logger:          logging.NewNamedLogger("parser").WithLevel(logging.LevelDebug),
		index:           indexInstance,
		dictionary:      dictionary,
		parserStarted:   time.Now(),
		reportAfter:     time.Now().Add(time.Second * defaultLogAverageEachSeconds),
		parsedForms:     0,
		logAverageSpeed: defaultLogAverageEachSeconds,
	}

	parser.OnElement("", parser.mute)
	parser.OnElement(".dictionary", parser.onDictionary)
	parser.OnElement(".dictionary.grammemes", parser.onGrammemes)
	parser.OnElement(".dictionary.grammemes.grammeme", parser.onGrammeme)
	parser.OnElement(".dictionary.grammemes.grammeme.name", parser.onGrammemeName)
	parser.OnElement(".dictionary.grammemes.grammeme.alias", parser.mute)
	parser.OnElement(".dictionary.grammemes.grammeme.description", parser.mute)
	parser.OnElement(".dictionary.restrictions", parser.mute)
	parser.OnElement(".dictionary.restrictions.restr", parser.mute)
	parser.OnElement(".dictionary.restrictions.restr.left", parser.mute)
	parser.OnElement(".dictionary.restrictions.restr.right", parser.mute)
	parser.OnElement(".dictionary.lemmata", parser.mute)
	parser.OnElement(".dictionary.lemmata.lemma", parser.onDictionaryLemmataLemma)
	parser.OnElement(".dictionary.lemmata.lemma.l", parser.onDictionaryLemmataLemmaL)
	parser.OnElement(".dictionary.lemmata.lemma.l.g", parser.onDictionaryLemmataLemmaLG)
	parser.OnElement(".dictionary.lemmata.lemma.f", parser.onDictionaryLemmataLemmaF)
	parser.OnElement(".dictionary.lemmata.lemma.f.g", parser.onDictionaryLemmataLemmaFG)
	parser.OnElement(".dictionary.link_types", parser.mute)
	parser.OnElement(".dictionary.link_types.type", parser.mute)
	parser.OnElement(".dictionary.links", parser.mute)
	parser.OnElement(".dictionary.links.link", parser.mute)

	return parser
}

func (parser *Parser) SetMaxLemmas(maxLemmas int) {
	parser.maxLemmas = maxLemmas
}

// mute игнорирует текущий элемент
func (parser *Parser) mute() *xmlstream.ElementProcessor {
	return &xmlstream.IgnoreElement
}

// onDictionary обрабатывает начало словаря, извлекая параметры version и revision
func (parser *Parser) onDictionary() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) error {
			return parser.dictionary.processElem(parser, element)
		},
		OnData: xmlstream.IgnoreElementData,
		OnEnd:  nil,
	}
}

func (parser *Parser) onGrammemes() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) error {
			parser.Info("parsing grammemes")
			return nil
		},
		OnData: xmlstream.IgnoreElementData,
		OnEnd: func(element xml.EndElement) error {
			switch typed := parser.index.(type) {
			case newIndexType:
				parser.Infof("parsing grammemes: total %d tags registered", typed.TagsCount())
			}
			return nil
		},
	}
}

// onGrammeme обрабатывает тег `grammeme`.
func (parser *Parser) onGrammeme() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) (err error) {
			var parentStr string

			parentStr, err = getAttr("parent", element.Attr)
			switch {
			case err != nil:
				return fmt.Errorf("%w: required parent attr", Error)
			case parentStr == "":
				return nil
			}

			parser.currentGrammeme = new(tag.Tag)
			parser.currentGrammeme.Parent = tag.Name(parentStr)

			return nil
		},
		OnData: xmlstream.IgnoreElementData,
		OnEnd: func(element xml.EndElement) error {
			//switch typed := parser.index.(type) {
			//case dag.Index:
			//	_ = typed.TagID(parser.currentGrammeme.Name, parser.currentGrammeme.Parent)
			//case newIndexType:
			//	_, err := typed.RegisterTag(*parser.currentGrammeme)
			//	if err != nil {
			//		return err
			//	}
			//}
			//
			parser.currentGrammeme = nil
			return nil
		},
	}
}

func (parser *Parser) onGrammemeName() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: xmlstream.IgnoreElementStart,
		OnData: func(data string) (err error) {
			switch typed := parser.index.(type) {
			case newIndexType:
				parser.Infof("register tag %s", data)
				if _, err = typed.RegisterTag(tag.Name(data)); err != nil {
					return err
				}
			}
			return nil
		},
		OnEnd: xmlstream.IgnoreElementEnd,
	}
}

func (parser *Parser) onDictionaryLemmataLemmaFG() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) (err error) {
			var tagString string

			if tagString, err = Attr(element.Attr).GetString("v"); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			parser.currentForm.G = append(parser.currentForm.G, &Category{VAttr: tag.Name(tagString)})

			return nil
		},
		OnData: xmlstream.IgnoreElementData,
		OnEnd:  xmlstream.IgnoreElementEnd,
	}
}

func (parser *Parser) onDictionaryLemmataLemmaF() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) (err error) {
			parser.currentForm = newWordForm()
			if parser.currentForm.Form, err = Attr(element.Attr).GetString("t"); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			return nil
		},
		OnData: nil,
		OnEnd: func(element xml.EndElement) error {
			parser.currentLemma.F = append(parser.currentLemma.F, parser.currentForm)
			parser.currentForm = nil
			parser.parsedForms++

			return nil
		},
	}
}

func (parser *Parser) onDictionaryLemmataLemma() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) (err error) {
			parser.currentLemma = newLemma()
			if parser.currentLemma.IdAttr, err = getIntAttr("id", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			if parser.currentLemma.RevAttr, err = getIntAttr("rev", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			return nil
		},
		OnData: xmlstream.IgnoreElementData,
		OnEnd: func(element xml.EndElement) (err error) {
			var (
				node dag.Node
			)
			for _, variant := range parser.currentLemma.F {
				// prepend form categories with Lemma.L categories list
				variant.G = append(parser.currentLemma.L.G, variant.G...)

				switch typedIndex := parser.index.(type) {
				case Index:
					if node, err = typedIndex.AddString(variant.Form); err != nil {
						return fmt.Errorf("index: %w", err)
					}

					if err = node.AddTagSet(variant.GetTagsFromSet()...); err != nil {
						return fmt.Errorf("add lemma variant: %w", err)
					}

					switch indexedNode := node.(type) {
					case *index.Node:
						item := indexedNode.Item()
						parser.Debugf(
							"IDX+ I%07d P%07d %#08x %v [%v] ",
							item.ID, item.Parent, item.Variants, variant.Form, variant.G)
					default:
						parser.Debugf("+ %v [%v]", variant.Form, variant.G)
					}
				case newIndexType:
					nodeIdx, err := typedIndex.Add(variant.Form, variant.GetTagsFromSet()...)
					if err != nil {
						return fmt.Errorf("index: %w", err)
					}
					parser.Debugf(
						"IDX+ I%07d P%07d %#08x %v [%v] ",
						nodeIdx, 0, 0, variant.Form, variant.G)
				}
			}

			parser.currentLemma = nil
			parser.parsedLemmas++

			if parser.maxLemmas > 0 && parser.parsedLemmas >= parser.maxLemmas {
				return ErrControlledStop
			}

			if time.Now().After(parser.reportAfter) {
				parser.Infof("avg %d lemma/sec", parser.parsedLemmas/int(time.Since(parser.parserStarted).Seconds()))
				parser.reportAfter = time.Now().Add(time.Second * time.Duration(parser.logAverageSpeed))
			}

			return nil
		},
	}
}

func (parser *Parser) onDictionaryLemmataLemmaL() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) (err error) {
			if parser.currentLemma.L.Form, err = getAttr("t", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			// parser.Debugf("lemma.l: `%v`", parser.currentLemma.L.Form)

			return nil
		},
		OnData: xmlstream.IgnoreElementData,
		OnEnd:  xmlstream.IgnoreElementEnd,
	}
}

func (parser *Parser) onDictionaryLemmataLemmaLG() *xmlstream.ElementProcessor {
	return &xmlstream.ElementProcessor{
		OnStart: func(element xml.StartElement) (err error) {
			var tagString string

			if tagString, err = getAttr("v", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}

			parser.currentLemma.L.G = append(parser.currentLemma.L.G, &Category{VAttr: tag.Name(tagString)})
			// parser.Debugf("lemma.l.g: `%v`: %v", parser.currentLemma.L.Form, parser.currentLemma.L.G)
			return nil
		},
		OnData: xmlstream.IgnoreElementData,
		OnEnd:  xmlstream.IgnoreElementEnd,
	}
}
