package xmlstream

import "encoding/xml"

// IgnoreElementStart defines stub to do nothing on element parsing start.
func IgnoreElementStart(_ xml.StartElement) error { return nil }

// IgnoreElementData defines stub to do nothing on element content parsing.
func IgnoreElementData(_ string) error { return nil }

// IgnoreElementEnd defines stub to do nothing on element closing parsing.
func IgnoreElementEnd(_ xml.EndElement) error { return nil }
