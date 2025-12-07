package xmlstream

import (
	"encoding/xml"
)

// OnStartHandler defines function interface to handle xml-tag start.
// It called when next tag is opened and provides xml tag name and attributes to handle in xml.StartElement element.
type OnStartHandler func(element xml.StartElement) error

// OnDataHandler defines function interface to handle string tag data (content between tag open and close).
type OnDataHandler func(data string) error

// OnEndHandler defines function interface to handle tag close.
type OnEndHandler func(element xml.EndElement) error

// ElementProcessor combines tag processing function to define tag start, content and closing together.
type ElementProcessor struct {
	OnStart OnStartHandler
	OnData  OnDataHandler
	OnEnd   OnEndHandler
}

var IgnoreElement = ElementProcessor{
	OnStart: IgnoreElementStart,
	OnData:  IgnoreElementData,
	OnEnd:   IgnoreElementEnd,
}
