package idl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/arf-rpc/idl/ast"
)

var reservedNames = map[string]struct{}{
	"package":   {},
	"import":    {},
	"as":        {},
	"struct":    {},
	"enum":      {},
	"service":   {},
	"optional":  {},
	"map":       {},
	"array":     {},
	"stream":    {},
	"string":    {},
	"int8":      {},
	"int16":     {},
	"int32":     {},
	"int64":     {},
	"uint8":     {},
	"uint16":    {},
	"uint32":    {},
	"uint64":    {},
	"float32":   {},
	"float64":   {},
	"bool":      {},
	"bytes":     {},
	"timestamp": {},
}

var primitives = map[string]struct{}{
	"string":    {},
	"int8":      {},
	"int16":     {},
	"int32":     {},
	"int64":     {},
	"uint8":     {},
	"uint16":    {},
	"uint32":    {},
	"uint64":    {},
	"float32":   {},
	"float64":   {},
	"bool":      {},
	"bytes":     {},
	"timestamp": {},
}

var camelCaseRegex = regexp.MustCompile(`^[A-Z]+[a-zA-Z0-9]*$`)
var snakeCaseRegex = regexp.MustCompile(`^[a-z]+[a-z_0-9]*$`)
var screamingSnakeCaseRegex = regexp.MustCompile(`^[A-Z]+[A-Z_0-9]*$`)

func parse(filepath string, tokens []token, onError func(error)) (*ast.File, []error) {
	var errors []error
	p := parser{
		tokens: tokens,
		length: len(tokens),
		onError: func(err error) {
			errors = append(errors, err)
			if onError != nil {
				onError(err)
			}
		},
		file: ast.File{
			Structs:       nil,
			Enums:         nil,
			Services:      nil,
			Package:       &ast.Package{},
			Imports:       nil,
			ImportAliases: map[string]string{},
			Path:          filepath,
		},
	}
	p.parse()
	if len(errors) > 0 {
		return nil, errors
	}
	return &p.file, nil
}

type parser struct {
	tokens      []token
	pos         int
	length      int
	file        ast.File
	comments    []token
	annotations []ast.Annotation
	onError     func(error)
}

func (p *parser) tokenPos(t *token) ast.Position {
	return ast.Position{
		File:     &p.file,
		Filename: p.file.Path,
		Line:     t.Line,
		Column:   t.Column,
	}
}

func (p *parser) errorf(format string, args ...interface{}) {
	p.onError(fmt.Errorf(format, args...))
}

func (p *parser) peek() token {
	return p.tokens[p.pos]
}

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) eof() bool {
	return p.pos >= len(p.tokens) || p.peek().Type == tokenTypeEOF
}

func (p *parser) expect(expected tokenType) *token {
	pk := p.peek()
	if pk.Type != expected {
		extra := ""
		msg := fmt.Sprintf("Expected %s but got %s at line %d column %d%s", expected, pk.Type, pk.Line, pk.Column, extra)
		p.errorf("%s", msg)
		return nil
	}
	p.pos++
	return &pk
}

func (p *parser) discardComments() {
	for p.peek().Type == tokenTypeComment {
		p.advance()
	}
}

func (p *parser) consumeUntilSemiOrLinebreak() {
	currentLine := p.peek().Line
	for {
		if p.peek().Type == tokenTypeSemi {
			p.advance()
			break
		}
		if p.peek().Line != currentLine {
			break
		}
		p.advance()
	}
}

func (p *parser) parsePackage() {
	pkg := p.expect(tokenTypeIdentifier)
	if pkg == nil {
		return
	}
	var components []string
	if pkg.Value != "package" {
		p.errorf("Expected package but got %s at line %d, column %d", pkg.Value, pkg.Line, pkg.Column)
		return
	}

	for !p.eof() {
		pk := p.peek()
		if pk.Type != tokenTypeIdentifier {
			p.errorf("Expected identifier at line %d column %d", pk.Line, pk.Column)
			p.consumeUntilSemiOrLinebreak()
			return
		}
		components = append(components, pk.Value)
		p.advance()
		if p.peek().Type != tokenTypePeriod {
			break
		}
		p.advance()
	}

	for _, v := range components {
		if !snakeCaseRegex.MatchString(v) {
			p.errorf("Invalid package component %s at line %d, column %d, expected snake_case", v, pkg.Line, pkg.Column)
		}
	}

	if p.expect(tokenTypeSemi) != nil {
		p.file.Package.Position = p.tokenPos(pkg)
		p.file.Package.Components = components
		p.file.Package.Value = strings.Join(components, ".")
	}
}

func (p *parser) parse() {
	p.discardComments()
	p.parsePackage()

	for !p.eof() {
		switch p.peek().Type {
		case tokenTypeComment:
			p.parseComments()
		case tokenTypeAtSign:
			p.parseAnnotations()
		case tokenTypeIdentifier:
			p.parseRootItem()
		default:
			p.errorf("Unexpected %s; expected comment, import, annotation, enum, struct, or service", p.peek().Value)
			p.consumeUntilSemiOrLinebreak()
		}
	}
}

func (p *parser) parseComments() {
	p.comments = []token{}
	var lastComment token
	for p.peek().Type == tokenTypeComment {
		if lastComment.Type != tokenTypeInvalid && p.peek().Line-lastComment.Line != 1 {
			p.comments = []token{}
		}
		lastComment = p.advance()
		p.comments = append(p.comments, lastComment)
	}
}

func (p *parser) parseAnnotations() {
	p.annotations = []ast.Annotation{}
	for p.peek().Type == tokenTypeAtSign {
		p.parseAnnotation()
	}
}

func (p *parser) parseAnnotation() {
	atSym := p.advance() // Consume @
	name := p.expect(tokenTypeIdentifier)
	if name == nil {
		p.consumeUntilSemiOrLinebreak()
		return
	}
	if p.peek().Type != tokenTypeLeftParen {
		p.annotations = append(p.annotations, ast.Annotation{
			Position: p.tokenPos(&atSym),
			Name:     name.Value,
		})
		return
	}

	p.advance() // Consume LeftParen
	var params []any
	for p.peek().Type == tokenTypeString {
		params = append(params, p.advance().Value)
		if p.peek().Type != tokenTypeComma {
			break
		}
	}
	if p.peek().Type != tokenTypeRightParen && p.peek().Type != tokenTypeString {
		p.errorf("Expected ) or string, got %s at line %d, column %d", p.peek().Value, p.peek().Line, p.peek().Column)
	}
	p.expect(tokenTypeRightParen)
	p.annotations = append(p.annotations, ast.Annotation{
		Position:  p.tokenPos(&atSym),
		Name:      name.Value,
		Arguments: params,
	})
}

func (p *parser) parseRootItem() {
	switch p.peek().Value {
	case "struct":
		p.file.Structs = append(p.file.Structs, p.parseStruct())
	case "enum":
		p.file.Enums = append(p.file.Enums, p.parseEnum())
	case "service":
		svc := p.parseService()
		for svcID, v := range p.file.Services {
			if v.Name == svc.Name {
				for _, meth := range svc.Methods {
					v.AppendMethod(meth)
				}
				p.file.Services[svcID] = v
				return
			}
		}
		p.file.Services = append(p.file.Services, &svc)
	case "import":
		p.file.Imports = append(p.file.Imports, p.parseImport())
	default:
		p.errorf("Unexpected %s; expected struct, enum, or service", p.peek().Value)
		p.consumeUntilSemiOrLinebreak()
	}
}

func (p *parser) takeAnnotations() []ast.Annotation {
	a := p.annotations
	p.annotations = []ast.Annotation{}
	return a
}

func (p *parser) takeComments() []token {
	c := p.comments
	p.comments = []token{}
	return c
}

func (p *parser) parseImport() *ast.Import {
	tk := p.advance() // consume "import"
	str := p.expect(tokenTypeString)
	if str == nil {
		p.consumeUntilSemiOrLinebreak()
		return &ast.Import{}
	}
	alias := ""
	if peek := p.peek(); peek.Type == tokenTypeIdentifier {
		if peek.Value != "as" {
			p.errorf("Expected 'as' or ';' after import path, got %s at line %d, column %d", peek.Value, peek.Line, peek.Column)
			p.consumeUntilSemiOrLinebreak()
			return &ast.Import{}
		}
		p.advance() // consume "as"
		alias = p.expect(tokenTypeIdentifier).Value
		if !snakeCaseRegex.MatchString(alias) {
			p.errorf("Invalid alias %s at line %d, column %d, expected snake_case", alias, peek.Line, peek.Column)
		}
	}
	p.expect(tokenTypeSemi)
	return &ast.Import{
		Position: p.tokenPos(&tk),
		Value:    str.Value,
		Alias:    alias,
	}
}

func mapFn[T any, C []T, U any](c C, fn func(T) U) []U {
	result := make([]U, len(c))
	for i, u := range c {
		result[i] = fn(u)
	}
	return result
}

func (p *parser) commentsAsStrings() []string {
	cmm := p.takeComments()
	return mapFn(cmm, func(t token) string { return t.Value })
}

func (p *parser) parseStruct() *ast.Struct {
	tk := p.advance() // Consume "struct"
	str := ast.Struct{
		Position:    p.tokenPos(&tk),
		Name:        "",
		Comment:     p.commentsAsStrings(),
		Annotations: p.takeAnnotations(),
		Fields:      nil,
		Structs:     nil,
		Enums:       nil,
		Parent:      nil,
	}

	if name := p.expect(tokenTypeIdentifier); name == nil {
		p.consumeUntilSemiOrLinebreak()
	} else {
		str.Name = name.Value
		if !camelCaseRegex.MatchString(name.Value) {
			p.errorf("Invalid struct name %s at line %d, column %d, expected CamelCase", name.Value, name.Line, name.Column)
		}
	}

	p.expect(tokenTypeLeftCurly)

loop:
	for !p.eof() {
		pk := p.peek()
		switch pk.Type {
		case tokenTypeIdentifier:
			switch pk.Value {
			case "struct":
				str.AppendStruct(p.parseStruct())
			case "enum":
				str.AppendEnum(p.parseEnum())
			case "service":
				p.errorf("Invalid service declaration at line %d, column %d: Services cannot be declared inside structs", pk.Line, pk.Column)
				p.parseService()
			default:
				v := pk.Value
				if _, ok := reservedNames[v]; ok {
					p.errorf("Unexpected %s at line %d, column %d, expected identifier", pk.Value, pk.Line, pk.Column)
					p.consumeUntilSemiOrLinebreak()
					continue
				}
				str.AppendField(p.parseStructField())
			}
		case tokenTypeAtSign:
			p.parseAnnotations()
		case tokenTypeComment:
			p.parseComments()
		case tokenTypeRightCurly:
			break loop
		default:
			p.errorf("unexpected %s at line %d, column %d, expected identifier", pk.Type, pk.Line, pk.Column)
			p.consumeUntilSemiOrLinebreak()
		}
	}

	p.expect(tokenTypeRightCurly)

	return &str
}

func (p *parser) parseStructField() ast.StructField {
	n := p.advance()
	f := ast.StructField{
		Position:    p.tokenPos(&n),
		Annotations: p.takeAnnotations(),
		Comment:     p.commentsAsStrings(),
		Name:        n.Value,
		Type:        nil,
		Parent:      nil,
	}

	if !snakeCaseRegex.MatchString(f.Name) {
		p.errorf("Invalid field name %s at line %d, column %d, expected snake_case", f.Name, f.Position.Line, f.Position.Column)
	}

	if fieldType := p.parseType(); p == nil {
		return f
	} else {
		f.Type = fieldType
	}

	if p.expect(tokenTypeSemi) == nil {
		p.consumeUntilSemiOrLinebreak()
		return f
	}
	return f
}

func (p *parser) parseEnum() *ast.Enum {
	tk := p.advance() // Consume "enum"
	en := ast.Enum{
		Position:    p.tokenPos(&tk),
		Comment:     p.commentsAsStrings(),
		Annotations: p.takeAnnotations(),
	}

	if name := p.expect(tokenTypeIdentifier); name == nil {
		p.consumeUntilSemiOrLinebreak()
	} else {
		en.Name = name.Value
		if !camelCaseRegex.MatchString(name.Value) {
			p.errorf("Invalid enum name %s at line %d, column %d, expected CamelCase", name.Value, name.Line, name.Column)
		}
	}

	p.expect(tokenTypeLeftCurly)

loop:
	for !p.eof() {
		pk := p.peek()
		switch pk.Type {
		case tokenTypeIdentifier:
			switch pk.Value {
			case "struct":
				p.errorf("Invalid struct declaration at line %d, column %d: Structs cannot be declared inside enums", pk.Line, pk.Column)
				p.parseStruct()
			case "enum":
				p.errorf("Invalid enum declaration at line %d, column %d: Enums cannot be declared inside enums", pk.Line, pk.Column)
				p.parseEnum()
			case "service":
				p.errorf("Invalid service declaration at line %d, column %d: Services cannot be declared inside enums", pk.Line, pk.Column)
				p.parseService()
			default:
				v := pk.Value
				if _, ok := reservedNames[v]; ok {
					p.errorf("Unexpected %s at line %d, column %d, expected identifier", pk.Value, pk.Line, pk.Column)
					p.consumeUntilSemiOrLinebreak()
					continue
				}
				en.AppendMember(p.parseEnumMember())
			}
		case tokenTypeAtSign:
			p.parseAnnotations()
		case tokenTypeComment:
			p.parseComments()
		case tokenTypeRightCurly:
			break loop
		default:
			p.errorf("Unexpected %s at line %d, column %d, expected identifier", pk.Type, pk.Line, pk.Column)
			p.consumeUntilSemiOrLinebreak()
		}
	}

	p.expect(tokenTypeRightCurly)

	return &en
}

func (p *parser) parseEnumMember() ast.EnumMember {
	member := ast.EnumMember{
		Comment:     p.commentsAsStrings(),
		Annotations: p.takeAnnotations(),
	}

	if name := p.expect(tokenTypeIdentifier); name == nil {
		p.consumeUntilSemiOrLinebreak()
		return member
	} else {
		member.Position = p.tokenPos(name)
		member.Name = name.Value
		if !screamingSnakeCaseRegex.MatchString(member.Name) {
			p.errorf("Invalid enum member name %s at line %d, column %d, expected SCREAMING_SNAKE_CASE", member.Name, member.Position.Line, member.Position.Column)
		}
	}

	if p.expect(tokenTypeEqual) == nil {
		p.consumeUntilSemiOrLinebreak()
		return member
	}

	if value := p.expect(tokenTypeNumber); value == nil {
		p.consumeUntilSemiOrLinebreak()
		return member
	} else {
		valueInt, err := strconv.Atoi(value.Value)
		if err != nil {
			p.errorf("failed parsing enum member value %s at line %d, column %d: %s", value.Value, value.Line, value.Column, err)
		} else {
			member.Value = valueInt
		}
	}

	if p.expect(tokenTypeSemi) == nil {
		p.consumeUntilSemiOrLinebreak()
	}

	return member
}

func (p *parser) parseService() ast.Service {
	tk := p.advance() // Consume "service"
	svc := ast.Service{
		Position:    p.tokenPos(&tk),
		Comment:     p.commentsAsStrings(),
		Annotations: p.takeAnnotations(),
	}

	if name := p.expect(tokenTypeIdentifier); name == nil {
		p.consumeUntilSemiOrLinebreak()
	} else {
		svc.Name = name.Value
		if !camelCaseRegex.MatchString(name.Value) {
			p.errorf("Invalid service name %s at line %d, column %d, expected CamelCase", name.Value, name.Line, name.Column)
		}
	}

	p.expect(tokenTypeLeftCurly)

loop:
	for !p.eof() {
		pk := p.peek()
		switch pk.Type {
		case tokenTypeIdentifier:
			switch pk.Value {
			case "struct":
				p.errorf("Invalid struct declaration at line %d, column %d: Structs cannot be declared inside services", pk.Line, pk.Column)
				p.parseStruct()
			case "enum":
				p.errorf("Invalid enum declaration at line %d, column %d: Enums cannot be declared inside services", pk.Line, pk.Column)
				p.parseEnum()
			case "service":
				p.errorf("Invalid service declaration at line %d, column %d: Services cannot be declared inside services", pk.Line, pk.Column)
				p.parseService()
			default:
				v := pk.Value
				if _, ok := reservedNames[v]; ok {
					p.errorf("Unexpected %s at line %d, column %d, expected identifier", pk.Value, pk.Line, pk.Column)
					p.consumeUntilSemiOrLinebreak()
					continue
				}
				svc.AppendMethod(p.parseServiceMethod())
			}
		case tokenTypeAtSign:
			p.parseAnnotations()
		case tokenTypeComment:
			p.parseComments()
		case tokenTypeRightCurly:
			break loop
		default:
			p.errorf("Unexpected %s at line %d, column %d, expected identifier", pk.Type, pk.Line, pk.Column)
			p.consumeUntilSemiOrLinebreak()
		}
	}

	p.expect(tokenTypeRightCurly)

	return svc
}

func (p *parser) parseServiceMethod() *ast.ServiceMethod {
	method := &ast.ServiceMethod{
		Comment:     p.commentsAsStrings(),
		Annotations: p.takeAnnotations(),
	}

	if name := p.expect(tokenTypeIdentifier); name == nil {
		p.consumeUntilSemiOrLinebreak()
		return method
	} else {
		method.Name = name.Value
		method.Position = p.tokenPos(name)
		if !camelCaseRegex.MatchString(method.Name) {
			p.errorf("Invalid method name %s at line %d, column %d, expected CamelCase", method.Name, method.Position.Line, method.Position.Column)
		}
	}

	if p.expect(tokenTypeLeftParen) == nil {
		p.consumeUntilSemiOrLinebreak()
		return method
	}

	if p.peek().Type != tokenTypeRightParen {
		for _, param := range p.parseMethodParams() {
			method.AppendParam(&param)
		}
	}

	if p.expect(tokenTypeRightParen) == nil {
		p.consumeUntilSemiOrLinebreak()
		return method
	}

	if p.peek().Type == tokenTypeArrow {
		p.advance() // Consume arrow
		for _, r := range p.parseMethodReturns() {
			method.AppendReturn(&r)
		}
	}

	streamFound := false
	for _, param := range method.Params {
		if streamFound {
			p.errorf("Stream must be the last parameter of a method at %s, line %d, column %d", param.Position.Filename, param.Position.Line, param.Position.Column)
			break
		}
		if param.Stream {
			streamFound = true
		}
	}

	streamFound = false
	for _, ret := range method.Returns {
		if streamFound {
			p.errorf("Stream must be the last return value of a method at %s, line %d, column %d", ret.Position.Filename, ret.Position.Line, ret.Position.Column)
			break
		}
		if ret.Stream {
			streamFound = true
		}
	}

	p.expect(tokenTypeSemi)
	return method
}

func (p *parser) parseMethodParams() []ast.MethodParam {
	res := []ast.MethodParam{p.parseMethodParam()}
	for p.peek().Type == tokenTypeComma {
		p.advance() // Consume comma
		res = append(res, p.parseMethodParam())
	}
	return res
}

func (p *parser) parseMethodParam() ast.MethodParam {
	param := ast.MethodParam{}
	if name := p.expect(tokenTypeIdentifier); name == nil {
		return param
	} else {
		param.Position = p.tokenPos(name)
		if name.Value == "stream" {
			param.Stream = true
		} else {
			param.Name = &name.Value
		}
	}
	param.Type = p.parseType()
	return param
}

func (p *parser) parseMethodReturns() []ast.MethodReturn {
	pk := p.peek()
	switch {
	case pk.Type == tokenTypeIdentifier:
		return []ast.MethodReturn{p.parseMethodReturn()}
	case pk.Type == tokenTypeLeftParen:
		p.advance()
		if p.peek().Type == tokenTypeRightParen {
			p.advance()
			return nil
		}
		ret := []ast.MethodReturn{p.parseMethodReturn()}
		for p.peek().Type == tokenTypeComma {
			p.advance() // consume comma
			ret = append(ret, p.parseMethodReturn())
		}
		p.expect(tokenTypeRightParen)
		return ret

	default:
		p.errorf("Unexpected %s at line %d, column %d, expected identifier", pk.Type.String(), pk.Line, pk.Column)
		p.consumeUntilSemiOrLinebreak()
		return nil
	}
}

func (p *parser) parseMethodReturn() ast.MethodReturn {
	pk := p.peek()
	switch {
	case pk.Type == tokenTypeIdentifier && pk.Value == "stream":
		p.advance()
		if p.peek().Type == tokenTypeLeftParen {
			p.errorf("Unexpected %s at line %d, column %d; cannot stream tuples", pk.Value, pk.Line, pk.Column)
			for !p.eof() && p.peek().Type != tokenTypeRightParen {
				p.advance()
			}
			return ast.MethodReturn{}
		}
		return ast.MethodReturn{Position: p.tokenPos(&pk), Type: p.parseType(), Stream: true}
	case pk.Type == tokenTypeIdentifier:
		return ast.MethodReturn{Position: p.tokenPos(&pk), Type: p.parseType(), Stream: false}
	case pk.Type == tokenTypeLeftParen:
		p.errorf("Unexpected %s at line %d, column %d; expected identifier", pk.Type, pk.Line, pk.Column)
		p.advance()
		if p.peek().Type == tokenTypeRightParen {
			p.advance()
		}
		return ast.MethodReturn{}
	default:
		p.errorf("Unexpected %s at line %d, column %d, expected identifier", pk.Type, pk.Line, pk.Column)
		p.consumeUntilSemiOrLinebreak()
		return ast.MethodReturn{}
	}
}

func (p *parser) parseType() ast.Type {
	typeName := p.expect(tokenTypeIdentifier)
	if typeName == nil {
		p.consumeUntilSemiOrLinebreak()
		return nil
	}
	switch typeName.Value {
	case "map":
		if p.expect(tokenTypeLeftAngled) == nil {
			p.consumeUntilSemiOrLinebreak()
			return nil
		}
		k := p.parseType()
		if p.expect(tokenTypeComma) == nil {
			p.consumeUntilSemiOrLinebreak()
			return nil
		}
		v := p.parseType()
		if p.expect(tokenTypeRightAngled) == nil {
			p.consumeUntilSemiOrLinebreak()
			return nil
		}
		return &ast.MapType{
			Position: p.tokenPos(typeName),
			Key:      k,
			Value:    v,
		}
	case "array":
		if p.expect(tokenTypeLeftAngled) == nil {
			p.consumeUntilSemiOrLinebreak()
			return nil
		}
		t := p.parseType()
		if p.expect(tokenTypeRightAngled) == nil {
			p.consumeUntilSemiOrLinebreak()
			return nil
		}
		return &ast.ArrayType{
			Position: p.tokenPos(typeName),
			Type:     t,
		}
	case "optional":
		if p.expect(tokenTypeLeftAngled) == nil {
			p.consumeUntilSemiOrLinebreak()
			return nil
		}
		t := p.parseType()
		if p.expect(tokenTypeRightAngled) == nil {
			p.consumeUntilSemiOrLinebreak()
			return nil
		}
		return &ast.OptionalType{
			Position: p.tokenPos(typeName),
			Type:     t,
		}
	default:
		if _, ok := primitives[typeName.Value]; ok {
			return &ast.PrimitiveType{
				Position: p.tokenPos(typeName),
				Name:     typeName.Value,
			}
		}
		if p.peek().Type == tokenTypePeriod {
			// Kind is composed
			typeParts := []token{*typeName}
			p.advance()
			for !p.eof() {
				if next := p.expect(tokenTypeIdentifier); next == nil {
					p.consumeUntilSemiOrLinebreak()
					return nil
				} else {
					typeParts = append(typeParts, *next)
				}
				if p.peek().Type == tokenTypePeriod {
					p.advance()
					continue
				}
				break
			}

			comps := mapFn(typeParts, func(t token) string { return t.Value })
			return &ast.FullQualifiedType{
				Position:   p.tokenPos(typeName),
				Package:    strings.Join(comps[0:len(comps)-1], "."),
				Name:       comps[len(comps)-1],
				FullName:   strings.Join(comps, "."),
				Components: comps,
			}
		}

		return &ast.SimpleUserType{Position: p.tokenPos(typeName), Name: typeName.Value}
	}
}
