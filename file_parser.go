package idl

import (
	"fmt"
	"github.com/arf-rpc/idl/ast"
	"strconv"
	"strings"
)

type fileParser struct {
	data []*Token
	cur  int
	len  int
	tree ast.Tree
	path string
}

func (p *fileParser) isAtEnd() bool {
	return p.peek().is(EOF)
}

func (p *fileParser) peek() *Token {
	return p.data[p.cur]
}

func (p *fileParser) Parse(filePath string, input []*Token) (*ast.Tree, error) {
	p.data = input
	p.cur = 0
	p.len = len(input)
	p.path = filePath

	if err := p.parsePackage(); err != nil {
		return nil, err
	}

	if err := p.parseImports(); err != nil {
		return nil, err
	}

	for !p.isAtEnd() {
		p.consumeBlanks()
		obj, err := p.parseStructEnumOrService()
		if err != nil {
			return nil, err
		}

		switch v := obj.(type) {
		case *ast.Service:
			p.tree.Services = append(p.tree.Services, v)
		case *ast.Enum:
			p.tree.Enums = append(p.tree.Enums, v)
		case *ast.Struct:
			p.tree.Structures = append(p.tree.Structures, v)
		}
		p.consumeBlanks()
	}

	return &p.tree, nil
}

func (p *fileParser) requireIdentifierNamed(named string) (*Token, error) {
	if !p.peek().is(Identifier) {
		return nil, p.errorf("expected identifier")
	}
	if p.peek().Value != named {
		return nil, p.errorf("expected '%s'", named)
	}

	return p.advance(), nil
}

func (p *fileParser) requireIdentifier() (*Token, error) {
	if !p.peek().is(Identifier) {
		return nil, p.errorf("expected identifier")
	}
	return p.advance(), nil
}

func (p *fileParser) requireNumber() (*Token, error) {
	if !p.peek().is(Number) {
		return nil, fmt.Errorf("expected number, found '%s'", p.peek().Value)
	}

	return p.advance(), nil
}

func (p *fileParser) advance() *Token {
	defer func() { p.cur++ }()
	return p.peek()
}

func (p *fileParser) consumeBlanks() {
	for {
		pk := p.peek()
		if pk.is(LineBreak) || pk.is(Indentation) || pk.is(Comment) {
			p.advance()
		} else {
			return
		}
	}
}

func (p *fileParser) errorf(format string, args ...any) error {
	return ParseError{
		Path:    p.path,
		Token:   p.peek(),
		Message: fmt.Sprintf(format, args...),
	}
}

func (p *fileParser) parsePackage() error {
	p.consumeBlanks()
	start, err := p.requireIdentifierNamed("package")
	if err != nil {
		return err
	}

	pName := []string{p.advance().Value}
	for p.peek().is(Identifier) || p.peek().is(Dot) {
		pName = append(pName, p.advance().Value)
	}
	if !p.peek().is(Semi) {
		return p.errorf("expected ';'")
	}
	end := p.advance()
	p.tree.Package = &ast.Package{
		Offset: offsetBetween(start, end),
		Name:   strings.Join(pName, ""),
	}

	return nil
}

func (p *fileParser) parseImports() error {
	for {
		p.consumeBlanks()
		if !p.peek().is(Identifier) || p.peek().Value != "import" {
			break
		}
		imp, err := p.parseImport()
		if err != nil {
			return err
		}
		p.tree.Imports = append(p.tree.Imports, imp)
	}

	return nil
}

func (p *fileParser) requireSemi() (*Token, error) {
	if !p.peek().is(Semi) {
		return nil, p.errorf("expected semicolon")
	}
	return p.advance(), nil
}

func (p *fileParser) require(el Element) (*Token, error) {
	if !p.peek().is(el) {
		return nil, p.errorf("expected '%s'", el)
	}

	return p.advance(), nil
}

func (p *fileParser) parseImport() (*ast.Import, error) {
	start := p.advance() // consume "import"
	if !p.peek().is(StringElement) {
		return nil, p.errorf("expected string")
	}
	path := p.advance()
	// Consume semi
	semi, err := p.requireSemi()
	if err != nil {
		return nil, err
	}

	return &ast.Import{
		Offset: offsetBetween(start, semi),
		Path:   path.Value,
	}, nil
}

func (p *fileParser) parseStructEnumOrService() (any, error) {
	p.consumeBlanks()

	if !p.peek().is(Identifier) {
		return nil, p.errorf("expected identifier")
	}

	annotations, err := p.parseAnnotations()
	if err != nil {
		return nil, err
	}

	val := p.peek().Value
	switch val {
	case "service":
		srv, err := p.parseService()
		if err != nil {
			return nil, err
		}
		srv.Annotations = annotations
		return srv, nil

	case "struct":
		str, err := p.parseStruct()
		if err != nil {
			return nil, err
		}
		str.Annotations = annotations
		return str, nil

	case "enum":
		en, err := p.parseEnum()
		if err != nil {
			return nil, err
		}
		en.Annotations = annotations
		return en, nil

	default:
		return nil, p.errorf("expected 'struct', 'enum', or 'service'")
	}
}

func (p *fileParser) parseService() (*ast.Service, error) {
	start, err := p.requireIdentifierNamed("service")
	if err != nil {
		return nil, err
	}

	name, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}

	_, err = p.require(OpenCurly)
	if err != nil {
		return nil, err
	}

	var mets []*ast.Method
	for !p.isAtEnd() {
		ann, err := p.parseAnnotations()
		if err != nil {
			return nil, err
		}

		met, err := p.parseMethod()
		if err != nil {
			return nil, err
		}

		met.Annotations = ann
		mets = append(mets, met)

		p.consumeBlanks()
		if p.peek().is(CloseCurly) {
			break
		}
	}

	end, err := p.require(CloseCurly)
	if err != nil {
		return nil, err
	}

	s := &ast.Service{
		Offset:  offsetBetween(start, end),
		Name:    name.Value,
		Methods: mets,
	}

	for _, met := range mets {
		met.Parent = s
	}

	return s, nil
}

func (p *fileParser) parseStruct() (*ast.Struct, error) {
	p.consumeBlanks()
	start, err := p.requireIdentifierNamed("struct")
	if err != nil {
		return nil, err
	}
	name, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}
	if _, err = p.require(OpenCurly); err != nil {
		return nil, err
	}

	var (
		fields  []*ast.Field
		structs []*ast.Struct
		enum    []*ast.Enum
	)

	for !p.isAtEnd() {
		ann, err := p.parseAnnotations()
		if err != nil {
			return nil, err
		}

		if p.peek().is(CloseCurly) {
			break
		}

		val, err := p.parseStructEnumOrField()
		if err != nil {
			return nil, err
		}

		switch v := val.(type) {
		case *ast.Struct:
			v.Annotations = ann
			structs = append(structs, v)
		case *ast.Enum:
			v.Annotations = ann
			enum = append(enum, v)
		case *ast.Field:
			v.Annotations = ann
			fields = append(fields, v)
		}

		p.consumeBlanks()
		if p.peek().is(CloseCurly) {
			break
		}
	}

	end, err := p.require(CloseCurly)
	if err != nil {
		return nil, err
	}

	s := &ast.Struct{
		Offset:  offsetBetween(start, end),
		Name:    name.Value,
		Fields:  fields,
		Enums:   enum,
		Structs: structs,
	}

	for _, field := range s.Fields {
		field.Parent = s
	}

	for _, enum := range s.Enums {
		enum.Parent = s
	}

	for _, str := range s.Structs {
		str.Parent = s
	}

	return s, nil
}

func (p *fileParser) parseStructEnumOrField() (any, error) {
	p.consumeBlanks()
	tk := p.peek()
	if !tk.is(Identifier) {
		return nil, p.errorf("expected identifier")
	}
	switch tk.Value {
	case "struct":
		return p.parseStruct()
	case "enum":
		return p.parseEnum()
	case "union":
		return p.parseUnion()
	default:
		return p.parseField()
	}
}

func (p *fileParser) parseUnion() (*ast.Field, error) {
	start, err := p.requireIdentifierNamed("union")
	if err != nil {
		return nil, err
	}
	name, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}
	_, err = p.require(OpenCurly)
	if err != nil {
		return nil, err
	}

	var fields []*ast.Field
	for !p.isAtEnd() {
		ann, err := p.parseAnnotations()
		if err != nil {
			return nil, err
		}

		f, err := p.parseField()
		if err != nil {
			return nil, err
		}
		f.Annotations = ann
		fields = append(fields, f)
		p.consumeBlanks()
		if p.peek().is(CloseCurly) {
			break
		}
	}

	end, err := p.require(CloseCurly)
	if err != nil {
		return nil, err
	}

	f := &ast.Field{
		Offset: offsetBetween(start, end),
		Union: &ast.UnionField{
			Name:   name.Value,
			Fields: fields,
		},
	}

	f.Union.Parent = f

	return f, nil
}

func (p *fileParser) parseField() (*ast.Field, error) {
	fieldName, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}

	fieldType, err := p.parseType()
	if err != nil {
		return nil, err
	}

	if _, err = p.require(Equal); err != nil {
		return nil, err
	}

	idx, err := p.requireNumber()
	if err != nil {
		return nil, err
	}

	parsedIdx, err := strconv.ParseInt(idx.Value, 10, 32)
	if err != nil {
		return nil, p.errorf("expected integer for field")
	}

	end, err := p.requireSemi()
	if err != nil {
		return nil, err
	}

	f := &ast.Field{
		Offset: offsetBetween(fieldName, end),
		Plain: &ast.PlainField{
			Name:  fieldName.Value,
			Type:  fieldType,
			Index: int(parsedIdx),
		},
	}

	f.Plain.Parent = f

	return f, nil
}

func (p *fileParser) parseEnum() (*ast.Enum, error) {
	start, err := p.requireIdentifierNamed("enum")
	if err != nil {
		return nil, err
	}
	name, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}
	_, err = p.require(OpenCurly)
	if err != nil {
		return nil, err
	}

	var opts []*ast.EnumOption
	for !p.isAtEnd() {
		ann, err := p.parseAnnotations()
		if err != nil {
			return nil, err
		}
		opt, err := p.parseEnumField()
		if err != nil {
			return nil, err
		}
		opt.Annotations = ann
		opts = append(opts, opt)

		if !p.peek().is(Semi) {
			break
		} else {
			p.advance() // consume Semi
		}
		p.consumeBlanks()
		if p.peek().is(CloseCurly) {
			break
		}
	}

	end, err := p.require(CloseCurly)
	if err != nil {
		return nil, err
	}

	e := &ast.Enum{
		Offset:  offsetBetween(start, end),
		Name:    name.Value,
		Options: opts,
	}

	for _, opt := range e.Options {
		opt.Parent = e
	}

	return e, nil
}

func (p *fileParser) parseEnumField() (*ast.EnumOption, error) {
	start, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}
	_, err = p.require(Equal)
	if err != nil {
		return nil, err
	}
	val, err := p.requireNumber()
	if err != nil {
		return nil, err
	}

	parsedIdx, err := strconv.ParseInt(val.Value, 10, 32)
	if err != nil {
		return nil, p.errorf("expected integer for enum option")
	}

	return &ast.EnumOption{
		Offset: offsetBetween(start, val),
		Name:   start.Value,
		Index:  int(parsedIdx),
	}, nil
}

func (p *fileParser) parseAnnotations() (ast.Annotations, error) {
	var res []*ast.Annotation
	for {
		p.consumeBlanks()
		if !p.peek().is(Annotation) {
			break
		}
		name := p.advance()
		var params []any
		endAt := name
		if p.peek().is(OpenCurly) {
			if !p.peek().is(CloseCurly) {
				params = append(params, p.advance())
				for p.peek().is(Comma) {
					p.advance() // consume comma
					params = append(params, p.advance())
				}
			}
			if !p.peek().is(CloseCurly) {
				return nil, p.errorf("expected ')'")
			}
			endAt = p.advance() // consume close curly
		}

		res = append(res, &ast.Annotation{
			Offset:    offsetBetween(name, endAt),
			Name:      name.Value,
			Arguments: params,
		})
	}

	return res, nil
}

func (p *fileParser) parseMethod() (*ast.Method, error) {
	p.consumeBlanks()
	name, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}

	_, err = p.require(OpenParen)
	if err != nil {
		return nil, err
	}

	var in []*ast.MethodParam
	var out []ast.Type

	if !p.peek().is(CloseParen) {
		for {
			param, err := p.parseMethodParam()
			if err != nil {
				return nil, err
			}
			in = append(in, param)

			if p.peek().is(Comma) {
				p.advance()
				continue
			}
			break
		}
	}

	_, err = p.require(CloseParen)
	if err != nil {
		return nil, err
	}

	if p.peek().is(Arrow) {
		p.advance() // Consume arrow
		if p.peek().is(OpenParen) {
			p.advance() // consume openParen
			for {
				t, err := p.parseMethodReturnParam()
				if err != nil {
					return nil, err
				}
				out = append(out, t)

				if p.peek().is(Comma) {
					// consume comma
					p.advance()
					continue
				}
				break
			}
			if _, err = p.require(CloseParen); err != nil {
				return nil, err
			}
		} else {
			outType, err := p.parseType()
			if err != nil {
				return nil, err
			}
			out = append(out, outType)
		}
	}

	end, err := p.requireSemi()
	if err != nil {
		return nil, err
	}

	return &ast.Method{
		Offset: offsetBetween(name, end),
		Name:   name.Value,
		Input:  in,
		Output: out,
	}, nil
}

func (p *fileParser) parseMethodParam() (*ast.MethodParam, error) {
	nameOrType, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}

	// In case we have `stream' or `map`, we surely have a type instead of a
	// name.
	if nameOrType.Value == "stream" || nameOrType.Value == "map" {
		p.cur--
		t, err := p.parseType()
		if err != nil {
			return nil, err
		}
		if nameOrType.Value == "stream" {
			t = t.Streaming()
		} else if nameOrType.Value == "map" {
		}
		return &ast.MethodParam{
			Name:  "",
			Named: false,
			Type:  t,
		}, nil
	}

	// in case the next value is an identifier, we can parse a type.
	if p.peek().is(Identifier) {
		t, err := p.parseType()
		if err != nil {
			return nil, err
		}
		return &ast.MethodParam{
			Name:  nameOrType.Value,
			Named: true,
			Type:  t,
		}, nil
	}

	// otherwise, we have only a type
	return &ast.MethodParam{
		Name:  "",
		Named: false,
		Type:  ast.IntoType(nameOrType.Value),
	}, nil
}

func (p *fileParser) parseMethodReturnParam() (ast.Type, error) {
	return p.parseType()
}

func (p *fileParser) parseType() (ast.Type, error) {
	typeName, err := p.requireIdentifier()
	if err != nil {
		return nil, err
	}

	if typeName.Value == "stream" {
		parsedType, err := p.parseType()
		if err != nil {
			return nil, err
		}

		return parsedType.Streaming(), nil
	}

	if typeName.Value == "map" {
		if _, err = p.require(OpenAngled); err != nil {
			return nil, err
		}

		keyType, err := p.parseType()
		if err != nil {
			return nil, err
		}

		if _, err := p.require(Comma); err != nil {
			return nil, err
		}

		valueType, err := p.parseType()
		if err != nil {
			return nil, err
		}

		if _, err := p.require(CloseAngled); err != nil {
			return nil, err
		}

		return &ast.MapType{Key: keyType, Value: valueType}, nil
	}

	return ast.IntoType(typeName.Value), nil
}

func parseTokens(path string, tokens []*Token) (*ast.Tree, error) {
	p := fileParser{}
	return p.Parse(path, tokens)
}
