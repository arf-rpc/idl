package idl

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

var simpleTokens = map[rune]Element{
	'(':  OpenParen,
	')':  CloseParen,
	'<':  OpenAngled,
	'>':  CloseAngled,
	'{':  OpenCurly,
	'}':  CloseCurly,
	',':  Comma,
	'.':  Dot,
	'=':  Equal,
	';':  Semi,
	'\n': LineBreak,
}

// scan takes an io.Reader and returns a list of Token from it, or an error, in
// case the file is invalid. This is a convenience function that creates a new
// scanner, reads the provided io.Reader into it, and returns the resulting
// value. scan does not close the provided io.Reader.
func scan(path string, r io.Reader) ([]*Token, error) {
	s, err := newScanner(path, r)
	if err != nil {
		return nil, err
	}
	return s.Run()
}

func newScanner(path string, r io.Reader) (*scanner, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &scanner{
		path:    path,
		data:    []rune(string(buf)),
		dataLen: len(buf),
		start:   0,
		current: 0,
	}, nil
}

type scanner struct {
	data            []rune
	dataLen         int
	start           int
	current         int
	lastIsLineBreak bool
	path            string
}

func (s *scanner) advance() rune {
	r := s.data[s.current]
	s.current++
	return r
}

func (s *scanner) peek() rune {
	if s.isAtEnd() {
		return 0x00
	}
	return s.data[s.current]
}

func (s *scanner) peekNext() rune {
	if s.current+1 >= s.dataLen {
		return 0x00
	}
	return s.data[s.current+1]
}

func (s *scanner) pos() (int, int) {
	line := 1
	column := 0
	for i := 0; i < s.current; i++ {
		if s.data[i] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}

func (s *scanner) isAtEnd() bool {
	return s.current >= len(s.data)
}

func (s *scanner) error(msg string, a ...interface{}) (*Token, error) {
	l, c := s.pos()
	return nil, SyntaxError{
		Path:    s.path,
		Message: fmt.Sprintf(msg, a...),
		Line:    l,
		Column:  c,
	}
}

func (s *scanner) Run() ([]*Token, error) {
	var tokens []*Token
	for !s.isAtEnd() {
		s.start = s.current
		if tok, err := s.scanToken(); err != nil {
			return nil, err
		} else {
			tokens = append(tokens, tok)
		}
	}
	eof, _ := s.makeToken(EOF, "")
	tokens = append(tokens, eof)
	return tokens, nil
}

func (s *scanner) makeToken(k Element, v string) (*Token, error) {
	l, c := s.pos()
	return &Token{
		Type:   k,
		Value:  v,
		Line:   l,
		Column: c,
	}, nil
}

func (s *scanner) scanToken() (*Token, error) {
	if s.lastIsLineBreak {
		s.lastIsLineBreak = false
		if p := s.peek(); unicode.IsSpace(p) && p != '\n' {
			return s.consumeIndent()
		}
	}

	r := s.advance()
	switch r {
	case '@':
		return s.annotation()

	case '-':
		if s.peek() != '>' {
			unkChar := s.advance()
			return s.error("Unexpected `%c', expected `>'", unkChar)
		}

		v, err := s.makeToken(Arrow, "->")
		if err != nil {
			return nil, err
		}
		// We advance later here so we can point the arrow to
		// the beginning of it instead of the end.
		s.advance()
		return v, nil

	case '\r', ' ', '\t':
		// Just consume it. We don't care about spaces
		s.start = s.current
		return s.scanToken()
	case '\n':
		tk, err := s.makeToken(LineBreak, string(r))
		if err != nil {
			return nil, err
		}
		s.lastIsLineBreak = true
		return tk, nil
	case '"':
		return s.string()

	case '#':
		return s.comment()

	default:
		if k, ok := simpleTokens[r]; ok {
			return s.makeToken(k, string(r))
		} else if unicode.IsDigit(r) {
			return s.number()
		} else if unicode.IsGraphic(r) {
			return s.identifier()
		} else {
			return s.error("Unexpected `%c'", r)
		}
	}
}

func (s *scanner) annotation() (*Token, error) {
	l, c := s.pos()
	for p := s.peek(); p != ' ' && p != '('; p = s.peek() {
		s.advance()
	}

	name := string(s.data[s.start+1 : s.current])

	var err error
	var params []*Token
	if s.peek() == '(' {
		s.advance()
		params, err = s.annotationParams()
		if err != nil {
			return nil, err
		}
	}

	consumed := s.current - s.start
	if consumed == 1 {
		return s.error("Unexpected `%c', expected identifier", s.peek())
	}
	return &Token{
		Type:             Annotation,
		Value:            strings.TrimSpace(name),
		Line:             l,
		Column:           c,
		AnnotationParams: params,
	}, nil
}

func (s *scanner) comment() (*Token, error) {
	for s.peek() != '\n' && !s.isAtEnd() {
		s.advance()
	}
	return s.makeToken(Comment, string(s.data[s.start+1:s.current]))
}

func (s *scanner) number() (*Token, error) {
	for unicode.IsDigit(s.peek()) {
		s.advance()
	}
	return s.makeToken(Number, string(s.data[s.start:s.current]))
}

func (s *scanner) string() (*Token, error) {
	escaping := false
	var val []rune
	for !s.isAtEnd() {
		if s.peek() == '\\' {
			if escaping {
				val = append(val, s.peek())
				escaping = false
			} else {
				escaping = true
			}
			s.advance()
			continue
		}
		if escaping {
			escaping = false
			val = append(val, s.advance())
			continue
		}
		if s.peek() == '"' {
			s.advance()
			return s.makeToken(StringElement, string(val))
		}

		val = append(val, s.advance())
	}

	return s.makeToken(StringElement, string(val))
}

func (s *scanner) identifier() (*Token, error) {
	l, col := s.pos()
	c := s.peek()
	if (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c == '_') {
		s.advance()
	}
	c = s.peek()
	for (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c == '_') ||
		(c >= '0' && c <= '9') {
		s.advance()
		c = s.peek()
	}

	return &Token{
		Type:   Identifier,
		Value:  string(s.data[s.start:s.current]),
		Line:   l,
		Column: col,
	}, nil
}

func (s *scanner) annotationParams() ([]*Token, error) {
	var tokens []*Token

	for {
		p := s.peek()
		if p == ')' || p == 0x00 {
			s.advance()
			return tokens, nil
		}
		t, err := s.scanToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
}

func (s *scanner) consumeIndent() (*Token, error) {
	for unicode.IsSpace(s.peek()) {
		s.advance()
	}

	l, col := s.pos()

	return &Token{
		Type:   Indentation,
		Value:  string(s.data[s.start:s.current]),
		Line:   l,
		Column: col,
	}, nil
}
