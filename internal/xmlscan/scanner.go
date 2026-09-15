// Package xmlscan provides a streaming byte-level scanner for the OpenCorpora dict.xml schema.
package xmlscan

import (
	"fmt"
	"io"
)

// Handler receives dictionary events. Slices point into scanner-owned
// buffers and are valid only until the next handler call.
type Handler interface {
	OnGrammeme(parent []byte, name []byte) error
	OnGrammemeRef(value []byte) error
	OnLemma(id uint32, text []byte) error
	// OnLemmaHeadEnd fires when <l> (the lemma's headword element) closes
	// — not when the enclosing <lemma> closes, which has no dedicated
	// event of its own. Implementations needing true end-of-lemma
	// behavior should reset their state on the next OnLemma call instead.
	OnLemmaHeadEnd() error
	OnForm(text []byte) error
	OnFormEnd() error
}

type section int

const (
	sectOther section = iota
	sectGrammemes
	sectLemmata
)

const defaultBufSize = 1 << 18

// Scanner reads dict.xml from r and emits events to h.
// It is not safe for concurrent use.
type Scanner struct {
	r   io.Reader
	h   Handler
	buf []byte
	pos int
	end int

	tag       []byte
	text      []byte
	ents      []byte
	tmpParent []byte
	tmpName   []byte
	attrs     []byte

	section     section
	inGrammeme  bool
	captureText bool
	inWord      bool
	lemmaID     uint32
}

func New(r io.Reader, h Handler) *Scanner {
	return NewBufferSize(r, h, defaultBufSize)
}

func NewBufferSize(r io.Reader, h Handler, size int) *Scanner {
	if size < 16 {
		size = 16
	}

	return &Scanner{
		r:   r,
		h:   h,
		buf: make([]byte, size),
		tag: make([]byte, 0, 512),
	}
}

// Scan processes the whole input. Handler errors abort the scan and are
// returned as is.
func (s *Scanner) Scan() error {
	for {
		if s.pos >= s.end {
			switch err := s.fill(); err {
			case nil:
			case io.EOF:
				return nil
			default:
				return err
			}
		}

		c := s.buf[s.pos]

		if c != '<' {
			s.pos++

			if s.captureText {
				s.text = append(s.text, c)
			}

			continue
		}

		// Buffer up to the longest prefix we distinguish below ("<!--") before
		// deciding, so a boundary landing right after "<!" doesn't get
		// misread as a bare "<!...>" declaration and truncate a comment at
		// its first '>' instead of its real "-->" close.
		s.ensureLookahead(4)

		switch {
		case s.hasPrefix("<?"):
			if err := s.skipUntil("?>"); err != nil {
				return err
			}
		case s.hasPrefix("<!--"):
			if err := s.skipUntil("-->"); err != nil {
				return err
			}
		case s.hasPrefix("<!"):
			if err := s.skipUntil(">"); err != nil {
				return err
			}
		default:
			if err := s.readTag(); err != nil {
				return err
			}

			if err := s.dispatch(); err != nil {
				return err
			}
		}
	}
}

// ensureLookahead tries to buffer at least n bytes starting at s.pos,
// pulling more from the reader as needed. It stops as soon as a fill call
// makes no further progress (reader exhausted) so it never loops forever.
func (s *Scanner) ensureLookahead(n int) {
	for s.end-s.pos < n {
		avail := s.end - s.pos
		if err := s.fill(); err != nil {
			return
		}

		if s.end-s.pos == avail {
			return
		}
	}
}

func (s *Scanner) hasPrefix(p string) bool {
	return s.end-s.pos >= len(p) && string(s.buf[s.pos:s.pos+len(p)]) == p
}

func (s *Scanner) fill() error {
	n := copy(s.buf, s.buf[s.pos:s.end])
	s.pos = 0
	s.end = n

	if n >= len(s.buf) {
		// The retained (not-yet-consumed) tail already fills the buffer, so
		// io.ReadFull below would read into a zero-length slice and return
		// (0, nil) forever — a silent infinite loop for the caller. This is
		// not reachable with today's minimum buffer size (16) and delimiters
		// (longest is 3 bytes), but fail loudly instead of hanging if that
		// ever changes.
		return fmt.Errorf("xmlscan: unconsumed token exceeds buffer size (%d bytes); use a larger NewBufferSize", len(s.buf))
	}

	read, err := io.ReadFull(s.r, s.buf[n:])
	s.end += read

	if err == io.ErrUnexpectedEOF || err == io.EOF {
		if s.end > s.pos {
			return nil
		}

		return io.EOF
	}

	if err != nil {
		return err
	}

	return nil
}

func (s *Scanner) skipUntil(delim string) error {
	for {
		for i := s.pos; i+len(delim) <= s.end; i++ {
			if string(s.buf[i:i+len(delim)]) == delim {
				s.pos = i + len(delim)

				return nil
			}
		}

		if len(delim) > 1 {
			s.pos = max(s.pos, s.end-len(delim)+1)
		} else {
			s.pos = s.end
		}

		if err := s.fill(); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}

			return err
		}
	}
}

func (s *Scanner) readTag() error {
	s.tag = s.tag[:0]

	var quote byte

	for {
		if s.pos >= s.end {
			if err := s.fill(); err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}

				return err
			}
		}

		c := s.buf[s.pos]
		s.pos++
		s.tag = append(s.tag, c)

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			return nil
		}
	}
}

type parsedTag struct {
	name        []byte
	selfClosing bool
	closing     bool
}

func (s *Scanner) parse() parsedTag {
	inner := s.tag[1 : len(s.tag)-1]

	t := parsedTag{}

	if len(inner) > 0 && inner[0] == '/' {
		t.closing = true
		inner = inner[1:]
	} else {
		t.selfClosing = detectSelfClosing(inner)
	}

	n := 0
	for n < len(inner) && isNameByte(inner[n]) {
		n++
	}

	t.name = inner[:n]
	s.attrs = inner[n:]

	return t
}

func detectSelfClosing(inner []byte) bool {
	var quote byte

	last := -1

	for i, c := range inner {
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '/':
			last = i
		}
	}

	return last == len(inner)-1
}

func isNameByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return true
	case c >= '0' && c <= '9':
		return true
	default:
		return c == '_' || c == '-' || c == ':' || c == '.'
	}
}

func (t parsedTag) is(name string) bool {
	return string(t.name) == name
}
