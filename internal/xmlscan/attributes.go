package xmlscan

import (
	"bytes"
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
			if tail, repl, ok := matchEntity(src[i:]); ok {
				s.ents = append(s.ents, repl)
				i += tail

				continue
			}
		}

		s.ents = append(s.ents, src[i])
		i++
	}

	return s.ents
}

var entities = map[string]byte{
	"amp;":  '&',
	"lt;":   '<',
	"gt;":   '>',
	"quot;": '"',
	"apos;": '\'',
}

func matchEntity(src []byte) (consumed int, replacement byte, ok bool) {
	for ent, char := range entities {
		if bytes.HasPrefix(src[1:], []byte(ent)) {
			return 1 + len(ent), char, true
		}
	}

	return 0, 0, false
}
