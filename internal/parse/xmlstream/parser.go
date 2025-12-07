package xmlstream

import (
	"encoding/xml"
	"errors"
	"fmt"
)

var (
	Error = errors.New("xml parse error")
)

// Parser provides some syntax-sugar to define XmlStreamParser of encoding/xml.
// Implements XmlStreamParser itself.
type Parser struct {
	collectedData string
	currentPath   string
	parsers       map[string]ElementProcessor
}

// New creates new instance.
func New() *Parser {
	return &Parser{
		currentPath:   "",
		collectedData: "",
		parsers:       make(map[string]ElementProcessor),
	}
}

func (parser *Parser) ProcessStartElement(element xml.StartElement) error {
	parser.collectedData = ""
	parser.currentPath += "." + element.Name.Local
	p, ok := parser.parsers[parser.currentPath]

	switch {
	case !ok:
		return fmt.Errorf("%w: unexpected start: `%v`", Error, parser.currentPath)
	case p.OnStart == nil:
		return nil
	case fmt.Sprintf("%p", p.OnStart) == fmt.Sprintf("%p", IgnoreElementStart):
		return nil
	default:
		return p.OnStart(element)
	}
}

func (parser *Parser) ProcessCharData(data xml.CharData) error {
	p, ok := parser.parsers[parser.currentPath]
	if !ok {
		return fmt.Errorf("%w: unexpected char: `%v`", Error, parser.currentPath)
	}
	switch {
	case p.OnData == nil:
		return nil
	case fmt.Sprintf("%p", p.OnData) == fmt.Sprintf("%p", IgnoreElementData):
		return nil
	default:
		return p.OnData(string(data))
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
	case p.OnEnd == nil:
		return nil
	case fmt.Sprintf("%p", p.OnEnd) == fmt.Sprintf("%p", IgnoreElementEnd):
		return nil
	default:
		return p.OnEnd(element)
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

// OnElement adds ElementProcessor to parse specified elementPath.
func (parser *Parser) OnElement(elementPath string, parseWith func() *ElementProcessor) {
	parser.parsers[elementPath] = *parseWith()
}
