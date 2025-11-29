package ast

import (
	"bytes"
	"fmt"
	"strings"
)

func Print(file *File) {
	p := printer{}
	p.print(file)
	fmt.Println(p.b.String())
}

type printer struct {
	b   bytes.Buffer
	lvl int
}

func (p *printer) inc() func() {
	p.lvl++
	return p.dec
}

func (p *printer) dec() { p.lvl-- }

func (p *printer) printf(format string, args ...interface{}) {
	p.b.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("  ", p.lvl), fmt.Sprintf(format, args...)))
}

func (p *printer) print(file *File) {
	p.printf("File: %s", file.Path)
	defer p.inc()()
	p.printf("Package: %s", file.Package.Value)
	if len(file.Imports) > 0 {
		p.printf("Imports:")
		p.printImports(file.Imports)
	}
	if len(file.Structs) > 0 {
		p.printf("Structs:")
		p.printStructs(file.Structs)
	}
	if len(file.Enums) > 0 {
		p.printf("Enums:")
		p.printEnums(file.Enums)
	}
	if len(file.Services) > 0 {
		p.printf("Services:")
		p.printServices(file.Services)
	}
}

func (p *printer) printImports(imports []*Import) {
	defer p.inc()()
	for _, imp := range imports {
		if imp.Alias != "" {
			p.printf(" - %s as %s", imp.Value, imp.Alias)
		} else {
			p.printf(" - %s", imp.Value)
		}
	}
}

func (p *printer) printStructs(structs []*Struct) {
	defer p.inc()()
	for _, st := range structs {
		p.printf("- Name: %s", st.Name)
		p.inc() // 1
		p.printComments(st.Comment)
		p.printAnnotations(st.Annotations)
		p.printFields(st.Fields)
		if len(st.Structs) > 0 {
			p.printf("Structs:")
			p.printStructs(st.Structs)
		}
		if len(st.Enums) > 0 {
			p.printf("Enums:")
			p.printEnums(st.Enums)
		}
		p.dec() // 1
	}
}

func (p *printer) printComments(c []string) {
	if len(c) == 0 {
		return
	}
	p.printf("Comment:")
	{
		p.inc() // 2
		for _, v := range c {
			p.printf("- %s", v)
		}
		p.dec() // 2
	}
}

func (p *printer) printAnnotation(v Annotation) {
	if len(v.Arguments) > 0 {
		args := make([]string, 0, len(v.Arguments))
		for _, param := range v.Arguments {
			args = append(args, fmt.Sprintf("%#v", param))
		}
		p.printf("- %s (%s)", v.Name, strings.Join(args, " "))
	} else {
		p.printf("- %s", v.Name)
	}
}

func (p *printer) printAnnotations(annotations []Annotation) {
	if len(annotations) == 0 {
		return
	}
	p.printf("Annotations:")
	{
		p.inc() // 3
		for _, v := range annotations {
			p.printAnnotation(v)
		}
		p.dec() // 3
	}
}

func (p *printer) printFields(fields []*StructField) {
	p.printf("Fields:")
	{
		p.inc()
		for _, v := range fields {
			p.printField(v)
		}
		p.dec()
	}
}

func (p *printer) printField(f *StructField) {
	p.printf("- %s", f.Name)
	defer p.inc()()
	p.printType(f.Type)
	p.printComments(f.Comment)
	p.printAnnotations(f.Annotations)
}

func (p *printer) printType(t Type) {
	switch tt := t.(type) {
	case *ArrayType:
		p.printf("Kind: Array")
		p.inc()
		p.printType(tt.Type)
		p.dec()
	case *MapType:
		p.printf("Kind: Map")
		p.inc()
		{
			p.printf("Key:")
			p.inc()
			p.printType(tt.Key)
			p.dec()
		}
		{
			p.printf("Value:")
			p.inc()
			p.printType(tt.Value)
			p.dec()
		}
		p.dec()
	case *OptionalType:
		p.printf("Kind: Optional")
		p.inc()
		p.printType(tt.Type)
		p.dec()
	case *PrimitiveType:
		p.printf("Kind: %s", tt.Name)
	case *SimpleUserType:
		p.printf("Kind: %s", tt.Name)
	case *FullQualifiedType:
		p.printf("Kind: %s", tt.FullName)
	}
}

func (p *printer) printEnums(enums []*Enum) {
	defer p.inc()()
	for _, e := range enums {
		p.printEnum(e)
	}
}

func (p *printer) printEnum(e *Enum) {
	p.printf("- Name: %s", e.Name)
	defer p.inc()()
	p.printComments(e.Comment)
	p.printAnnotations(e.Annotations)
	p.printf("Members:")
	defer p.inc()()
	for _, m := range e.Members {
		p.printf("- %s: %d", m.Name, m.Value)
		p.inc()
		p.printAnnotations(m.Annotations)
		p.printComments(m.Comment)
		p.dec()
	}
}

func (p *printer) printServices(services []*Service) {
	defer p.inc()()
	for _, s := range services {
		p.printService(s)
	}
}

func (p *printer) printService(s *Service) {
	p.printf("- Name: %s", s.Name)
	defer p.inc()()
	p.printComments(s.Comment)
	p.printAnnotations(s.Annotations)
	p.printf("Methods:")
	defer p.inc()()
	for _, m := range s.Methods {
		p.printServiceMethod(m)
	}
}

func (p *printer) printServiceMethod(m *ServiceMethod) {
	p.printf("- Name: %s", m.Name)
	defer p.inc()()
	p.printComments(m.Comment)
	p.printAnnotations(m.Annotations)
	if len(m.Params) > 0 {
		p.printf("Arguments:")
		p.printMethodParams(m.Params)
	}
	if len(m.Returns) > 0 {
		p.printf("Returns:")
		p.printMethodReturns(m.Returns)
	}
}

func (p *printer) printMethodParams(params []*MethodParam) {
	defer p.inc()()
	for _, param := range params {
		if param.Name != nil {
			p.printf("- Name: %s", *param.Name)
		} else {
			p.printf("- Name: (Anonymous)")
		}
		p.inc()
		p.printf("Stream: %t", param.Stream)
		p.printType(param.Type)
		p.dec()
	}
}

func (p *printer) printMethodReturns(params []*MethodReturn) {
	defer p.inc()()
	for idx, param := range params {
		p.inc()
		p.printf("- Index: %d", idx)
		p.printf("Stream: %t", param.Stream)
		p.printType(param.Type)
		p.dec()
	}
}
