package idl

import "fmt"

// SyntaxError indicates that a provided file does not contain a valid ARF
// Interface Description File.
type SyntaxError struct {
	Path    string
	Message string
	Line    int
	Column  int
}

func (s SyntaxError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s at line %d, column %d", s.Path, s.Line, s.Column, s.Message, s.Line, s.Column)
}

// ParseError indicates that one or more productions from the scanner does not
// define a valid IDL file.
type ParseError struct {
	Token   *Token
	Message string
	Path    string
}

func (p ParseError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s at %#v on line %d, column %d", p.Path, p.Token.Line, p.Token.Column, p.Message, p.Token.Value, p.Token.Line, p.Token.Column)
}
