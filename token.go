package idl

import "fmt"

type tokenType int

func (t tokenType) String() string {
	return tokenTypeAsString[t]
}

const (
	tokenTypeInvalid tokenType = iota
	tokenTypeEOF
	tokenTypeComment
	tokenTypeIdentifier
	tokenTypeNumber
	tokenTypeString
	tokenTypeEqual
	tokenTypeLeftCurly
	tokenTypeRightCurly
	tokenTypeLeftParen
	tokenTypeRightParen
	tokenTypeLeftAngled
	tokenTypeRightAngled
	tokenTypeSemi
	tokenTypeComma
	tokenTypePeriod
	tokenTypeAtSign
	tokenTypeArrow
)

var tokenTypeAsString = map[tokenType]string{
	tokenTypeInvalid:     "Invalid",
	tokenTypeEOF:         "EOF",
	tokenTypeComment:     "Comment",
	tokenTypeIdentifier:  "Identifier",
	tokenTypeNumber:      "Number",
	tokenTypeString:      "String",
	tokenTypeEqual:       "Equal",
	tokenTypeLeftCurly:   "LeftCurly",
	tokenTypeRightCurly:  "RightCurly",
	tokenTypeLeftParen:   "LeftParen",
	tokenTypeRightParen:  "RightParen",
	tokenTypeLeftAngled:  "LeftAngled",
	tokenTypeRightAngled: "RightAngled",
	tokenTypeSemi:        "Semi",
	tokenTypeComma:       "Comma",
	tokenTypePeriod:      "Period",
	tokenTypeAtSign:      "AtSign",
	tokenTypeArrow:       "Arrow",
}

type token struct {
	Type   tokenType
	Value  string
	Pos    int
	Line   int
	Column int
}

func (t token) String() string {
	return fmt.Sprintf("idl.token{Kind: %s, Value: %q, Pos: %d, Line: %d, Column: %d}", t.Type, t.Value, t.Pos, t.Line, t.Column)
}
