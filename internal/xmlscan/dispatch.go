package xmlscan

import "strconv"

func (s *Scanner) dispatch() error {
	t := s.parse()

	if t.closing {
		return s.closeTag(t)
	}

	switch {
	case t.is("grammemes"):
		s.section = sectGrammemes
	case t.is("lemmata"):
		s.section = sectLemmata
	case t.is("dictionary"), t.is("restrictions"), t.is("link_types"), t.is("links"):
		s.section = sectOther
	case t.is("grammeme"):
		s.inGrammeme = true
		s.tmpName = s.tmpName[:0]

		parent := s.attr("parent")
		s.tmpParent = append(s.tmpParent[:0], parent...)

		if t.selfClosing {
			s.inGrammeme = false

			return s.h.OnGrammeme(parent, nil)
		}
	case t.is("name"):
		if s.inGrammeme {
			s.captureText = true
			s.text = s.text[:0]
		}
	case t.is("alias"), t.is("description"):
	case t.is("lemma"):
		id, err := strconv.ParseUint(string(s.attr("id")), 10, 32)
		if err != nil {
			return err
		}

		s.lemmaID = uint32(id)

	case t.is("l"):
		if s.section != sectLemmata {
			return nil
		}

		s.inWord = true

		if err := s.h.OnLemma(s.lemmaID, s.attr("t")); err != nil {
			return err
		}

		if t.selfClosing {
			s.inWord = false

			return s.h.OnLemmaEnd()
		}

	case t.is("f"):
		if s.section != sectLemmata {
			return nil
		}

		s.inWord = true

		if err := s.h.OnForm(s.attr("t")); err != nil {
			return err
		}

		if t.selfClosing {
			s.inWord = false

			return s.h.OnFormEnd()
		}

	case t.is("g"):
		if s.section == sectLemmata && s.inWord {
			return s.h.OnGrammemeRef(s.attr("v"))
		}
	}

	return nil
}

func (s *Scanner) closeTag(t parsedTag) error {
	switch {
	case t.is("grammemes"), t.is("lemmata"):
		s.section = sectOther
	case t.is("grammeme"):
		if s.inGrammeme {
			s.inGrammeme = false

			return s.h.OnGrammeme(s.tmpParent, s.tmpName)
		}

	case t.is("name"):
		if s.captureText {
			s.captureText = false
			s.tmpName = append(s.tmpName[:0], s.text...)
		}

	case t.is("l"):
		if s.section == sectLemmata && s.inWord {
			s.inWord = false

			return s.h.OnLemmaEnd()
		}

	case t.is("f"):
		if s.inWord {
			return s.h.OnFormEnd()
		}
	}

	return nil
}
