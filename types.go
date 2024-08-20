package idl

import (
	"fmt"
	"github.com/arf-rpc/idl/ast"
)

//go:generate stringer -type=Element -output=types_string.go

// Element represents a single Token element kind in a source file
type Element int

const (
	InvalidElement Element = iota
	Identifier             // [a-z][0-9a-z_]
	OpenCurly              // {
	CloseCurly             // }
	OpenParen              // (
	CloseParen             // )
	OpenAngled             // <
	CloseAngled            // >
	Comma                  // ,
	Dot                    // .
	LineBreak              // \n
	Equal                  // =
	Number                 // 0-9+
	Arrow                  // ->
	Semi                   // ;
	Comment                // Anything from # onwards
	Annotation             // Anything from @ until next space
	StringElement          // Anything between "
	Indentation
	EOF
)

// Token represents a single token present in a source file
type Token struct {
	Type   Element
	Value  string
	Line   int
	Column int

	AnnotationParams []*Token
}

func (t Token) is(o Element) bool { return t.Type == o }
func (t Token) String() string {
	return fmt.Sprintf("Token{Type=%d (%s), Value=%#v, Line=%d, Column=%d}", t.Type, t.Type.String(), t.Value, t.Line, t.Column)
}

func offsetBetween(a, b *Token) ast.Offset {
	return ast.Offset{
		StartsAt: ast.Position{
			Line:   a.Line,
			Column: a.Column,
		},
		EndsAt: ast.Position{
			Line:   b.Line,
			Column: b.Column,
		},
	}
}
