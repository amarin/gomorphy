package xmlscan

import (
	"bytes"
	"strconv"
	"unicode/utf8"
)

func (s *Scanner) attr(name string) []byte {
	rest := s.attrs

	for len(rest) > 0 {
		rest = trimSpaces(rest)
		if len(rest) == 0 {
			return nil
		}

		n := 0
		for n < len(rest) && rest[n] != '=' && !isSpace(rest[n]) && rest[n] != '/' {
			n++
		}

		key := rest[:n]
		rest = rest[n:]

		if len(rest) > 0 && (isSpace(rest[0]) || rest[0] == '/') {
			if string(key) == name {
				return nil
			}

			continue
		}

		if len(rest) == 0 || rest[0] != '=' || len(rest) < 2 || rest[1] != '"' {
			continue
		}

		rest = rest[2:]

		v := 0
		for v < len(rest) && rest[v] != '"' {
			v++
		}

		value := rest[:v]
		rest = rest[min(v+1, len(rest)):]

		if string(key) == name {
			return s.decodeEntities(value)
		}
	}

	return nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func trimSpaces(b []byte) []byte {
	for len(b) > 0 && isSpace(b[0]) {
		b = b[1:]
	}

	return b
}

func (s *Scanner) decodeEntities(src []byte) []byte {
	if !bytes.ContainsRune(src, '&') {
		return src
	}

	s.ents = s.ents[:0]

	for i := 0; i < len(src); {
		if src[i] == '&' {
			if tail, r, ok := matchEntity(src[i:]); ok {
				s.ents = utf8.AppendRune(s.ents, r)
				i += tail

				continue
			}
		}

		s.ents = append(s.ents, src[i])
		i++
	}

	return s.ents
}

var namedEntities = map[string]rune{
	"amp;":  '&',
	"lt;":   '<',
	"gt;":   '>',
	"quot;": '"',
	"apos;": '\'',
}

// matchEntity recognizes the 5 predefined XML entities (&amp; &lt; &gt;
// &quot; &apos;) plus numeric character references (&#39; decimal,
// &#x27; hex). src[0] must be '&'.
func matchEntity(src []byte) (consumed int, r rune, ok bool) {
	if len(src) > 1 && src[1] == '#' {
		return matchNumericEntity(src)
	}

	for ent, char := range namedEntities {
		if bytes.HasPrefix(src[1:], []byte(ent)) {
			return 1 + len(ent), char, true
		}
	}

	return 0, 0, false
}

func matchNumericEntity(src []byte) (consumed int, r rune, ok bool) {
	i := 2 // past "&#"
	base := 10

	if i < len(src) && (src[i] == 'x' || src[i] == 'X') {
		base = 16
		i++
	}

	start := i
	for i < len(src) && isDigitInBase(src[i], base) {
		i++
	}

	if i == start || i >= len(src) || src[i] != ';' {
		return 0, 0, false
	}

	val, err := strconv.ParseInt(string(src[start:i]), base, 32)
	if err != nil || val < 0 || val > utf8.MaxRune {
		return 0, 0, false
	}

	return i + 1, rune(val), true
}

func isDigitInBase(c byte, base int) bool {
	switch {
	case c >= '0' && c <= '9':
		return true
	case base == 16 && (c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'):
		return true
	default:
		return false
	}
}
