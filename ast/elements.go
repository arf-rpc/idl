package ast

import (
	"fmt"
	"slices"
	"strings"
)

type Tree struct {
	Package    *Package
	Imports    []*Import
	Structures []*Struct
	Enums      []*Enum
	Services   []*Service
}

func (t Tree) String() string {
	var value []string

	if t.Package != nil {
		value = append(value, fmt.Sprintf("Package: %s", t.Package.Name))
	}

	for _, i := range t.Imports {
		value = append(value, i.String())
	}

	for _, e := range t.Enums {
		value = append(value, e.String())
	}

	for _, s := range t.Structures {
		value = append(value, s.String())
	}

	for _, s := range t.Services {
		value = append(value, s.String())
	}

	return fmt.Sprintf("Tree{%s}", strings.Join(value, ", "))
}

// Package represents a `package` declaration in a source file.
type Package struct {
	Offset Offset
	Name   string
}

// Import represents a `import` statement, which includes a path to be loaded.
type Import struct {
	Offset Offset
	Path   string
}

func (i Import) String() string {
	return fmt.Sprintf("Import{%s}", i.Path)
}

// Offset represents the offset in which a given structure appears in the source
// file. It includes Position for both the point in which it starts, and the
// point in which it ends.
type Offset struct {
	StartsAt Position
	EndsAt   Position
}

// Position represents a given Line/Column position within a source file.
type Position struct {
	Line   int
	Column int
}

type Annotation struct {
	Offset    Offset
	Name      string
	Arguments []any
}

func (a Annotation) String() string {
	args := make([]string, len(a.Arguments))
	for i, arg := range a.Arguments {
		args[i] = fmt.Sprint(arg)
	}
	separator := ""
	if len(a.Arguments) > 0 {
		separator = " "
	}
	return fmt.Sprintf("Annotation{%s%s%s}", a.Name, separator, strings.Join(args, ","))
}

type Annotations []*Annotation

func (a Annotations) ByName(name string) *Annotation {
	for _, v := range a {
		if v.Name == name {
			return v
		}
	}

	return nil
}

func (a Annotations) String() string {
	vals := make([]string, len(a))
	for i, a := range a {
		vals[i] = a.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(vals, ", "))
}

type Field struct {
	Offset      Offset
	Annotations Annotations
	Union       *UnionField
	Plain       *PlainField
	Parent      *Struct
}

func (f *Field) Path() string { return pathOf(f) }

func (f *Field) String() string {
	var vals []string
	if f.Union != nil {
		vals = append(vals, fmt.Sprintf("Union: %s", f.Union.String()))
	} else {
		vals = append(vals, fmt.Sprintf("Plain: %s", f.Plain.String()))
	}
	vals = append(vals, fmt.Sprintf("Annotations: %s", f.Annotations.String()))
	return fmt.Sprintf("Field{%s}", strings.Join(vals, ", "))
}

type PlainField struct {
	Name   string
	Type   Type
	Index  int
	Parent *Field
}

func (f PlainField) Path() string { return pathOf(f) }

func (f PlainField) String() string {
	return fmt.Sprintf("PlainField{Name: %s, Type: %s, Index: %d}", f.Name, f.Type, f.Index)
}

type UnionField struct {
	Fields []*Field
	Parent *Field
	Name   string
}

func (f UnionField) Path() string { return pathOf(f) }

func (f UnionField) String() string {
	fields := make([]string, len(f.Fields))
	for i, f := range f.Fields {
		fields[i] = f.String()
	}
	return fmt.Sprintf("UnionField{Name: %s, Fields: [%s]}", f.Name, strings.Join(fields, ", "))
}

type Method struct {
	Offset      Offset
	Name        string
	Input       []*MethodParam
	Output      []Type
	Annotations Annotations
	Parent      *Service
}

func (m Method) Path() string { return pathOf(m) }

func (m Method) String() string {
	in := make([]string, len(m.Input))
	for i, p := range m.Input {
		in[i] = p.String()
	}
	out := make([]string, len(m.Output))
	for i, p := range m.Output {
		out[i] = p.String()
	}
	return fmt.Sprintf("Method{Name: %s, Input: [%s], Output: [%s], Annotations: %s}",
		m.Name,
		strings.Join(in, ", "),
		strings.Join(out, ", "),
		m.Annotations)
}

type Struct struct {
	Offset      Offset
	Name        string
	Fields      []*Field
	Enums       []*Enum
	Structs     []*Struct
	Annotations Annotations
	Parent      *Struct
}

func (s Struct) Path() string { return pathOf(s) }

func (s Struct) String() string {
	fields := make([]string, len(s.Fields))
	enums := make([]string, len(s.Enums))
	structs := make([]string, len(s.Structs))

	for i, f := range s.Fields {
		fields[i] = f.String()
	}

	for i, e := range s.Enums {
		enums[i] = e.String()
	}

	for i, s := range s.Structs {
		structs[i] = s.String()
	}

	return fmt.Sprintf("Struct{Name: %s, Fields: [%s], Enums: [%s], Structs: [%s], Annotations: %s}", s.Name, strings.Join(fields, ", "), strings.Join(enums, ", "), strings.Join(structs, ", "), s.Annotations)
}

type Service struct {
	Offset      Offset
	Name        string
	Methods     []*Method
	Annotations Annotations
}

func (s Service) Path() string { return pathOf(s) }

func (s Service) String() string {
	methods := make([]string, len(s.Methods))
	for i, m := range s.Methods {
		methods[i] = m.String()
	}

	return fmt.Sprintf("Service{Name: %s, Methods: [%s], Annotations: %s}", s.Name, strings.Join(methods, ", "), s.Annotations)
}

type Enum struct {
	Offset      Offset
	Name        string
	Options     []*EnumOption
	Annotations Annotations
	Parent      *Struct
}

func (e Enum) Path() string {
	return pathOf(e)
}

func (e Enum) String() string {
	options := make([]string, len(e.Options))
	for i, opt := range e.Options {
		options[i] = opt.String()
	}

	return fmt.Sprintf("Enum{Name: %s, Options: [%s], Annotations: %s}", e.Name, strings.Join(options, ", "), e.Annotations)
}

type EnumOption struct {
	Offset      Offset
	Name        string
	Index       int
	Annotations Annotations
	Parent      *Enum
}

func (e EnumOption) Path() string {
	return pathOf(e)
}

func (e EnumOption) String() string {
	return fmt.Sprintf("EnumOption{Name: %s, Index: %d, Annotations: %s}", e.Name, e.Index, e.Annotations)
}

type MethodParam struct {
	Name  string
	Named bool
	Type  Type
}

func (m MethodParam) String() string {
	return fmt.Sprintf("MethodParam{Name: %q, Named: %t, Type: %s}", m.Name, m.Named, m.Type)
}

func pathOf(val any) string {
	var parent any
	var comps []string

	switch v := val.(type) {
	case *EnumOption:
		if v.Parent == nil {
			parent = nil
		} else {
			parent = v.Parent
		}
		comps = append(comps, v.Name)
	case *Enum:
		if v.Parent == nil {
			parent = nil
		} else {
			parent = v.Parent
		}
		comps = append(comps, v.Name)
	case *Struct:
		if v.Parent == nil {
			parent = nil
		} else {
			parent = v.Parent
		}
		comps = append(comps, v.Name)
	case *Service:
		parent = nil
		comps = append(comps, v.Name)
	case *UnionField:
		if v.Parent == nil {
			parent = nil
		} else {
			parent = v.Parent
		}
	case *PlainField:
		if v.Parent == nil {
			parent = nil
		} else {
			parent = v.Parent
		}
		comps = append(comps, v.Name)
	case *Field:
		if v.Parent == nil {
			parent = nil
		} else {
			parent = v.Parent
		}
		if v.Plain != nil {
			comps = append(comps, v.Plain.Name)
		} else {
			comps = append(comps, "union")
		}
	default:
		panic(fmt.Sprintf("unknown type %T", val))
	}

	for parent != nil {
		if parent == (*Struct)(nil) {
			fmt.Printf("LOLSIES\n")
		}
		switch v := parent.(type) {
		case *EnumOption:
			if v.Parent == nil {
				parent = nil
			} else {
				parent = v.Parent
			}
			comps = append(comps, v.Name)
		case *Enum:
			if v.Parent == nil {
				parent = nil
			} else {
				parent = v.Parent
			}

			comps = append(comps, v.Name)
		case *Struct:
			if v.Parent == nil {
				parent = nil
			} else {
				parent = v.Parent
			}
			comps = append(comps, v.Name)
		case *Service:
			parent = nil
			comps = append(comps, v.Name)
		case *UnionField:
			if v.Parent == nil {
				parent = nil
			} else {
				parent = v.Parent
			}
		case *PlainField:
			if v.Parent == nil {
				parent = nil
			} else {
				parent = v.Parent
			}
			comps = append(comps, v.Name)
		case *Field:
			parent = v.Parent
			if v.Plain != nil {
				comps = append(comps, v.Plain.Name)
			} else {
				comps = append(comps, "union")
			}
		default:
			panic(fmt.Sprintf("unknown type %T", v))
		}
	}

	slices.Reverse(comps)
	return strings.Join(comps, ".")
}
