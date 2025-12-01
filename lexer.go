package idl

import "fmt"

type lexer struct {
	data      []rune
	len       int
	pos       int
	startPos  int
	startLine int
	startCol  int

	line   int
	column int

	onError func(error)
	tokens  []token
}

func lexFile(data []byte, onError func(error)) ([]token, []error) {
	var errors []error
	runes := []rune(string(data))
	s := &lexer{
		data:   runes,
		len:    len(runes),
		line:   1,
		column: 1,
		onError: func(err error) {
			errors = append(errors, err)
			if onError != nil {
				onError(err)
			}
		},
	}

	s.scan()

	return s.tokens, errors
}

func (s *lexer) eof() bool {
	return s.pos >= s.len
}

func (s *lexer) peek() rune {
	return s.data[s.pos]
}

func (s *lexer) peek1() rune {
	if s.pos+1 >= s.len {
		return 0
	}
	return s.data[s.pos+1]
}

func (s *lexer) mark() {
	s.startPos = s.pos
	s.startLine = s.line
	s.startCol = s.column
}

func (s *lexer) marked() string {
	return string(s.data[s.startPos:s.pos])
}

func (s *lexer) advance() rune {
	v := s.data[s.pos]
	s.pos++
	s.column++
	if v == '\n' {
		s.line++
		s.column = 1
	}
	return v
}

func (s *lexer) errorf(msg string, args ...interface{}) {
	s.onError(fmt.Errorf("%s at %d:%d", fmt.Sprintf(msg, args...), s.startLine, s.startCol))
}

func (s *lexer) match(r rune) bool {
	if s.peek() == r {
		s.advance()
		return true
	}
	s.errorf("Unexpected '%c'", s.peek())
	return false
}

func (s *lexer) pushToken(t tokenType) {
	s.tokens = append(s.tokens, token{
		Type:   t,
		Value:  s.marked(),
		Pos:    s.startPos,
		Line:   s.startLine,
		Column: s.startCol,
	})
}

func (s *lexer) pushSimple(t tokenType) {
	s.mark()
	s.advance()
	s.pushToken(t)
}

func isAscii(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isAlpha(r rune) bool {
	return isAscii(r) || isDigit(r)
}

var simpleTokens = map[rune]tokenType{
	'=': tokenTypeEqual,
	';': tokenTypeSemi,
	'(': tokenTypeLeftParen,
	')': tokenTypeRightParen,
	'{': tokenTypeLeftCurly,
	'}': tokenTypeRightCurly,
	'<': tokenTypeLeftAngled,
	'>': tokenTypeRightAngled,
	',': tokenTypeComma,
	'@': tokenTypeAtSign,
	'.': tokenTypePeriod,
}

func (s *lexer) scan() {
	for !s.eof() {
		p := s.peek()
		switch p {
		case ' ', '\n', '\t', '\r':
			s.advance()
		case '#':
			s.advance()
			s.mark()
			for !s.eof() && s.peek() != '\n' {
				s.advance()
			}
			s.pushToken(tokenTypeComment)
		case '"', '\'':
			s.parseString(p)
		case '-':
			s.mark()
			s.advance()
			if s.match('>') {
				s.pushToken(tokenTypeArrow)
				continue
			}
		default:
			if simple, ok := simpleTokens[p]; ok {
				s.pushSimple(simple)
			} else if isDigit(p) {
				if s.peek1() == 'x' {
					s.parseHex()
				} else {
					s.parseNumber()
				}
			} else if isAscii(p) {
				s.parseIdentifier()
			} else {
				s.errorf("Unexpected '%c'", p)
				s.advance()
			}
		}
	}
	s.mark()
	s.tokens = append(s.tokens, token{Type: tokenTypeEOF, Pos: s.startPos, Line: s.line, Column: s.column})
}

func (s *lexer) parseString(q rune) {
	startPos := s.pos
	startLine := s.startLine
	startCol := s.startCol
	s.advance() // Consume first quote
	var data []rune
	escaping := false
	for !s.eof() {
		p := s.peek()
		if escaping {
			escaping = false
			if p == q {
				data = append(data, s.advance())
			} else {
				data = append(data, '\\', s.advance())
			}
			continue
		}
		if s.peek() == '\\' {
			escaping = true
			s.advance()
			continue
		}

		if p == '\n' {
			s.errorf("Invalid line break in string")
			s.advance()
			continue
		}

		if p == q {
			s.advance()
			break
		}

		data = append(data, s.advance())
	}

	s.tokens = append(s.tokens, token{
		Type:   tokenTypeString,
		Value:  string(data),
		Pos:    startPos,
		Line:   startLine,
		Column: startCol,
	})
}

func (s *lexer) parseNumber() {
	s.mark()
	for isDigit(s.peek()) {
		s.advance()
	}
	s.pushToken(tokenTypeNumber)
}

func (s *lexer) parseHex() {
	s.mark()
	s.advance() // consume 0
	s.advance() // consume x
	for isHex(s.peek()) {
		s.advance()
	}
	s.pushToken(tokenTypeHex)
}

func (s *lexer) parseIdentifier() {
	s.mark()
	for isAlpha(s.peek()) {
		s.advance()
	}
	s.pushToken(tokenTypeIdentifier)
}
