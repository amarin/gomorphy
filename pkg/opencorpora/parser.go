package opencorpora

import (
	"encoding/xml"
	"fmt"
	"time"

	"git.media-tel.ru/railgo/logging"

	"github.com/amarin/gomorphy/pkg/dag"
)

const defaultLogAverageEachSeconds = 10

type processStart func(element xml.StartElement) error
type processData func(data string) error
type processEnd func(element xml.EndElement) error

type elementProcessor struct {
	processStart processStart
	processData  processData
	processEnd   processEnd
}

var MuteStart = func(element xml.StartElement) error { return nil }
var MuteData = func(data string) error { return nil }
var MuteEnd = func(element xml.EndElement) error { return nil }
var MuteALL = elementProcessor{processStart: MuteStart, processData: MuteData, processEnd: MuteEnd}

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
	parsedWords     int
	logAverageSpeed int // report average parse speed each logAverageSpeed seconds
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
	case fmt.Sprintf("%p", p.processStart) == fmt.Sprintf("%p", MuteStart):
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
	case fmt.Sprintf("%p", p.processData) == fmt.Sprintf("%p", MuteData):
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
	case fmt.Sprintf("%p", p.processEnd) == fmt.Sprintf("%p", MuteEnd):
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
		parsedWords:     0,
		logAverageSpeed: defaultLogAverageEachSeconds,
	}

	parser.parsers[""] = MuteALL
	parser.parsers[".dictionary"] = elementProcessor{
		processStart: func(element xml.StartElement) error {
			return parser.dictionary.processElem(parser, element)
		},
		processData: MuteData,
		processEnd:  nil,
	}
	parser.parsers[".dictionary.grammemes"] = MuteALL
	parser.parsers[".dictionary.grammemes.grammeme"] = elementProcessor{
		processStart: func(element xml.StartElement) error {
			parser.currentGrammeme = new(dag.Tag)
			return nil
		},
		processData: MuteData,
		processEnd: func(element xml.EndElement) error {
			_ = parser.index.TagID(parser.currentGrammeme.Name, parser.currentGrammeme.Parent)
			parser.currentGrammeme = nil
			return nil
		},
	}
	parser.parsers[".dictionary.grammemes.grammeme.name"] = elementProcessor{
		processStart: MuteStart,
		processData: func(data string) error {
			parser.currentGrammeme.Name = dag.TagName(data)
			return nil
		},
		processEnd: MuteEnd,
	}
	parser.parsers[".dictionary.grammemes.grammeme.alias"] = MuteALL
	parser.parsers[".dictionary.grammemes.grammeme.description"] = MuteALL
	parser.parsers[".dictionary.restrictions"] = MuteALL
	parser.parsers[".dictionary.restrictions.restr"] = MuteALL
	parser.parsers[".dictionary.restrictions.restr.left"] = MuteALL
	parser.parsers[".dictionary.restrictions.restr.right"] = MuteALL
	parser.parsers[".dictionary.lemmata"] = MuteALL
	parser.parsers[".dictionary.lemmata.lemma"] = parser.parseLemma()
	parser.parsers[".dictionary.lemmata.lemma.l"] = elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			if parser.currentLemma.L.Form, err = getAttr("t", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}

			return nil
		},
		processData: MuteData,
		processEnd:  MuteEnd,
	}
	parser.parsers[".dictionary.lemmata.lemma.l.g"] = elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			var tagString string

			if tagString, err = getAttr("v", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			parser.currentLemma.L.G = append(parser.currentLemma.L.G, &Category{VAttr: dag.TagName(tagString)})
			return nil
		},
		processData: MuteData,
		processEnd:  MuteEnd,
	}
	parser.parsers[".dictionary.lemmata.lemma.f"] = elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			parser.currentForm = newWordForm()
			if parser.currentForm.Form, err = getAttr("t", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			// parser.Debugf("+ %v", parser.currentForm.Form)
			return nil
		},
		processData: nil,
		processEnd: func(element xml.EndElement) error {
			parser.currentLemma.F = append(parser.currentLemma.F, parser.currentForm)
			parser.currentForm = nil
			return nil
		},
	}
	parser.parsers[".dictionary.lemmata.lemma.f.g"] = elementProcessor{
		processStart: func(element xml.StartElement) (err error) {
			var tagString string

			if tagString, err = getAttr("v", element.Attr); err != nil {
				return fmt.Errorf("%w: %v: %v", Error, element.Attr, err)
			}
			parser.currentForm.G = append(parser.currentForm.G, &Category{VAttr: dag.TagName(tagString)})
			return nil
		},
		processData: MuteData,
		processEnd:  MuteEnd,
	}
	parser.parsers[".dictionary.link_types"] = MuteALL
	parser.parsers[".dictionary.link_types.type"] = MuteALL
	parser.parsers[".dictionary.links"] = MuteALL
	parser.parsers[".dictionary.links.link"] = MuteALL

	return parser
}

func (parser *Parser) parseLemma() elementProcessor {
	return elementProcessor{
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
		processData: MuteData,
		processEnd: func(element xml.EndElement) (err error) {
			var node dag.Node
			// parser.Debugf("+ %v", parser.currentLemma.L.Form)
			// parser.Debugf("+ %v", parser.currentLemma.String())
			for _, variant := range parser.currentLemma.F {
				// parser.Infof("adding %v", variant.Form)
				if node, err = parser.index.AddString(variant.Form); err != nil {
					return fmt.Errorf("index: %w", err)
				}

				if err = node.AddTagSet(variant.GetTagsFromSet()...); err != nil {
					return fmt.Errorf("add lemma variant: %w", err)
				}
			}

			parser.currentLemma = nil

			parser.parsedWords += 1
			if time.Now().After(parser.reportAfter) {
				parser.Infof("avg %d/sec", parser.parsedWords/int(time.Since(parser.parserStarted).Seconds()))
				parser.reportAfter = time.Now().Add(time.Second * time.Duration(parser.logAverageSpeed))
			}

			return nil
		},
	}
}
