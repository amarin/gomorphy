package opencorpora

import (
	"encoding/xml"
	"fmt"
	"time"

	"git.media-tel.ru/railgo/logging"

	"github.com/amarin/gomorphy/pkg/dag"
)

const defaultLogAverageEachSeconds = 10

// ErrControlledStop raised when limit to parse items set.
var ErrControlledStop = fmt.Errorf("%w: controlled stop", Error)

type processStart func(element xml.StartElement) error
type processData func(data string) error
type processEnd func(element xml.EndElement) error

type elementProcessor struct {
	processStart processStart
	processData  processData
	processEnd   processEnd
}

var ignoreElementStart = func(element xml.StartElement) error { return nil }
var ignoreElementData = func(data string) error { return nil }
var ignoreElementEnd = func(element xml.EndElement) error { return nil }
var ignoreElement = elementProcessor{processStart: ignoreElementStart, processData: ignoreElementData, processEnd: ignoreElementEnd}

type Parser struct {
	logging.Logger
	index           dag.Index
	dictionary      *Dictionary
	collectedData   string
	currentPath     string
	currentGrammeme *dag.Tag
	currentLemma    *Lemma
	currentForm     *WordForm
	parsers         map[string]elementProcessor
	parserStarted   time.Time
	reportAfter     time.Time
	parsedLemmas    int // parsed lemma's items
	parsedForms     int // parsed lemma forms
	logAverageSpeed int // report average parse speed each logAverageSpeed seconds

	// max forms to parse.
	// If !=0 parser will be stopped with ErrControlledStop after maximum maxLemmas lemmas forms parsing.
	maxLemmas int
}

func newParser(indexInstance dag.Index) *Parser {
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
		Logger:          logging.NewNamedLogger("parser").WithLevel(logging.LevelDebug),
		index:           indexInstance,
		dictionary:      dictionary,
		collectedData:   "",
		parsers:         make(map[string]elementProcessor),
		parserStarted:   time.Now(),
		reportAfter:     time.Now().Add(time.Second * defaultLogAverageEachSeconds),
		parsedForms:     0,
		logAverageSpeed: defaultLogAverageEachSeconds,
	}

	parser.on("", parser.mute)
	parser.on(".dictionary", parser.onDictionary)
	parser.on(".dictionary.grammemes", parser.mute)
	parser.on(".dictionary.grammemes.grammeme", parser.onGrammeme)
	parser.on(".dictionary.grammemes.grammeme.name", parser.onGrammemeName)
	parser.on(".dictionary.grammemes.grammeme.alias", parser.mute)
	parser.on(".dictionary.grammemes.grammeme.description", parser.mute)
	parser.on(".dictionary.restrictions", parser.mute)
	parser.on(".dictionary.restrictions.restr", parser.mute)
	parser.on(".dictionary.restrictions.restr.left", parser.mute)
	parser.on(".dictionary.restrictions.restr.right", parser.mute)
	parser.on(".dictionary.lemmata", parser.mute)
	parser.on(".dictionary.lemmata.lemma", parser.onDictionaryLemmataLemma)
	parser.on(".dictionary.lemmata.lemma.l", parser.onDictionaryLemmataLemmaL)
	parser.on(".dictionary.lemmata.lemma.l.g", parser.onDictionaryLemmataLemmaLG)
	parser.on(".dictionary.lemmata.lemma.f", parser.onDictionaryLemmataLemmaF)
	parser.on(".dictionary.lemmata.lemma.f.g", parser.onDictionaryLemmataLemmaFG)
	parser.on(".dictionary.link_types", parser.mute)
	parser.on(".dictionary.link_types.type", parser.mute)
	parser.on(".dictionary.links", parser.mute)
	parser.on(".dictionary.links.link", parser.mute)

	return parser
}

func (parser *Parser) SetMaxLemmas(maxLemmas int) {
	parser.maxLemmas = maxLemmas
}

func (parser *Parser) ProcessStartElement(element xml.StartElement) error {
	parser.collectedData = ""
	parser.currentPath += "." + element.Name.Local
	p, ok := parser.parsers[parser.currentPath]

	switch {
	case !ok:
		return fmt.Errorf("%w: unexpected start: `%v`", Error, parser.currentPath)
	case p.processStart == nil:
		parser.Info("> " + parser.currentPath)
		return nil
	case fmt.Sprintf("%p", p.processStart) == fmt.Sprintf("%p", ignoreElementStart):
		return nil
	default:
		return p.processStart(element)
	}
}

func (parser *Parser) ProcessCharData(data xml.CharData) error {
	p, ok := parser.parsers[parser.currentPath]
	if !ok {
		return fmt.Errorf("%w: unexpected char: `%v`", Error, parser.currentPath)
	}
	switch {
	case p.processData == nil:
		parser.Debug("= " + parser.currentPath)
		return nil
	case fmt.Sprintf("%p", p.processData) == fmt.Sprintf("%p", ignoreElementData):
		return nil
	default:
		return p.processData(string(data))
	}
}

func (parser *Parser) ProcessEndElement(element xml.EndElement) error {
	p, ok := parser.parsers[parser.currentPath]
	if !ok {
		return fmt.Errorf("%w: unexpected end: `%v`", Error, parser.currentPath)
	}

	defer func() {
		parser.currentPath = parser.currentPath[:len(parser.currentPath)-len(element.Name.Local)-1]
	}()

	switch {
	case p.processEnd == nil:
		parser.Debug("< " + parser.currentPath)
		return nil
	case fmt.Sprintf("%p", p.processEnd) == fmt.Sprintf("%p", ignoreElementEnd):
		return nil
	default:
		return p.processEnd(element)
	}
}

func (parser *Parser) ProcessComment(_ xml.Comment) error {
	return nil
}

func (parser *Parser) ProcessProcInst(_ xml.ProcInst) error {
	return nil
}

func (parser *Parser) ProcessDirective(_ xml.Directive) error {
	return nil
}

// on sets elementProcessor to parse specified elementPath.
func (parser *Parser) on(elementPath string, parseWith func() *elementProcessor) {
	parser.parsers[elementPath] = *parseWith()
}

func (parser *Parser) mute() *elementProcessor {
	return &ignoreElement
}

func (parser *Parser) onDictionary() *elementProcessor {
	return &elementProcessor{
		processStart: func(element xml.StartElement) error {
			return parser.dictionary.processElem(parser, element)
		},
		processData: ignoreElementData,
		processEnd:  nil,
	}
}

func (parser *Parser) onGrammeme() *elementProcessor {
	return &elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			var parentStr string

			parser.currentGrammeme = new(dag.Tag)
			if parentStr, err = getAttr("parent", element.Attr); err != nil {
				return fmt.Errorf("%w: required parent attr", Error)
			}
			parser.currentGrammeme.Parent = dag.TagName(parentStr)

			return nil
		},
		processData: ignoreElementData,
		processEnd: func(element xml.EndElement) error {
			_ = parser.index.TagID(parser.currentGrammeme.Name, parser.currentGrammeme.Parent)
			parser.currentGrammeme = nil
			return nil
		},
	}
}

func (parser *Parser) onGrammemeName() *elementProcessor {
	return &elementProcessor{
		processStart: ignoreElementStart,
		processData: func(data string) error {
			parser.currentGrammeme.Name = dag.TagName(data)
			return nil
		},
		processEnd: ignoreElementEnd,
	}
}

func (parser *Parser) onDictionaryLemmataLemmaFG() *elementProcessor {
	return &elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			var tagString string

			if tagString, err = Attr(element.Attr).GetString("v"); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			parser.currentForm.G = append(parser.currentForm.G, &Category{VAttr: dag.TagName(tagString)})

			return nil
		},
		processData: ignoreElementData,
		processEnd:  ignoreElementEnd,
	}
}

func (parser *Parser) onDictionaryLemmataLemmaF() *elementProcessor {
	return &elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			parser.currentForm = newWordForm()
			if parser.currentForm.Form, err = Attr(element.Attr).GetString("t"); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			return nil
		},
		processData: nil,
		processEnd: func(element xml.EndElement) error {
			parser.currentLemma.F = append(parser.currentLemma.F, parser.currentForm)
			parser.currentForm = nil
			parser.parsedForms++

			return nil
		},
	}
}

func (parser *Parser) onDictionaryLemmataLemma() *elementProcessor {
	return &elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			parser.currentLemma = newLemma()
			if parser.currentLemma.IdAttr, err = getIntAttr("id", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			if parser.currentLemma.RevAttr, err = getIntAttr("rev", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			return nil
		},
		processData: ignoreElementData,
		processEnd: func(element xml.EndElement) (err error) {
			var node dag.Node
			for _, variant := range parser.currentLemma.F {
				// prepend form grammemes from Lemma.L
				variant.G = append(parser.currentLemma.L.G, variant.G...)
				// parser.Debugf("+ %v [%v]", variant.Form, variant.G)

				debugWord := "сын"
				if variant.Form == debugWord {
					parser.Infof("+ %v", debugWord)
				}

				if node, err = parser.index.AddString(variant.Form); err != nil {
					return fmt.Errorf("index: %w", err)
				}

				if variant.Form == debugWord {
					parser.Infof("+ %v: node ts %v add %v", debugWord, node.TagSets(), variant.GetTagsFromSet())
				}

				if err = node.AddTagSet(variant.GetTagsFromSet()...); err != nil {
					return fmt.Errorf("add lemma variant: %w", err)
				}

				if variant.Form == debugWord {
					parser.Infof("+ %v: node ts %v", debugWord, node.TagSets())
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

func (parser *Parser) onDictionaryLemmataLemmaL() *elementProcessor {
	return &elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			if parser.currentLemma.L.Form, err = getAttr("t", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			// parser.Debugf("lemma.l: `%v`", parser.currentLemma.L.Form)

			return nil
		},
		processData: ignoreElementData,
		processEnd:  ignoreElementEnd,
	}
}

func (parser *Parser) onDictionaryLemmataLemmaLG() *elementProcessor {
	return &elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			var tagString string

			if tagString, err = getAttr("v", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}

			parser.currentLemma.L.G = append(parser.currentLemma.L.G, &Category{VAttr: dag.TagName(tagString)})
			// parser.Debugf("lemma.l.g: `%v`: %v", parser.currentLemma.L.Form, parser.currentLemma.L.G)
			return nil
		},
		processData: ignoreElementData,
		processEnd:  ignoreElementEnd,
	}
}
