package ast

import (
	"sort"
	"strings"
)

type Container interface {
	FindEnum(name string) *Enum
	FindStruct(name string) *Struct
}

type Tree struct {
	Packages map[string]*PackageTree
}

type PackageTree struct {
	Files      []*File
	Structures []*Struct
	Enums      []*Enum
	Services   []*Service
	Imports    []*Import
	Package    string
}

func (t *Tree) AddFile(file *File) {
	if t.Packages == nil {
		t.Packages = make(map[string]*PackageTree)
	}
	tree, ok := t.Packages[file.Package.Value]
	if !ok {
		t.Packages[file.Package.Value] = new(PackageTree)
		tree = t.Packages[file.Package.Value]
		tree.Package = file.Package.Value
	}
	tree.Files = append(tree.Files, file)
	for _, v := range file.Structs {
		tree.Structures = append(tree.Structures, v)
	}
	for _, v := range file.Enums {
		tree.Enums = append(tree.Enums, v)
	}
	for _, v := range file.Services {
		tree.Services = append(tree.Services, v)
	}
	for _, v := range file.Imports {
		tree.Imports = append(tree.Imports, v)
	}
}

type Position struct {
	Filename string
	Line     int
	Column   int
	File     *File
}

type Object interface {
	Kind() string
	Pos() *Position
	BaseFQN() string
	FQN() string
}

type File struct {
	Structs       []*Struct
	Enums         []*Enum
	Services      []*Service
	Package       *Package
	Imports       []*Import
	ImportAliases map[string]string
	Path          string
}

func (*File) Kind() string      { return "File" }
func (*File) Pos() *Position    { return nil }
func (f *File) BaseFQN() string { return f.Package.Value }
func (f *File) FQN() string     { return f.BaseFQN() }
func (f *File) FindEnum(name string) *Enum {
	for _, e := range f.Enums {
		if e.Name == name {
			return e
		}
	}
	return nil
}
func (f *File) FindStruct(name string) *Struct {
	for _, s := range f.Structs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

type Package struct {
	Position   Position
	Value      string
	Components []string
}

func (p *Package) Kind() string    { return "Package" }
func (p *Package) Pos() *Position  { return &p.Position }
func (p *Package) BaseFQN() string { return p.Position.File.BaseFQN() }
func (p *Package) FQN() string     { return p.BaseFQN() }

type Import struct {
	Position      Position
	Value         string
	ResolvedValue string
	Alias         string
}

func (i *Import) Kind() string    { return "Import" }
func (i *Import) Pos() *Position  { return &i.Position }
func (i *Import) BaseFQN() string { return i.Position.File.BaseFQN() }
func (i *Import) FQN() string     { return i.BaseFQN() }

type Struct struct {
	Position    Position
	Name        string
	Comment     []string
	Annotations AnnotationSet
	Fields      []*StructField
	Structs     []*Struct
	Enums       []*Enum
	Parent      *Struct
}

func (*Struct) Kind() string     { return "Struct" }
func (s *Struct) Pos() *Position { return &s.Position }

func (s *Struct) AppendStruct(st *Struct) {
	st.Parent = s
	s.Structs = append(s.Structs, st)
}

func (s *Struct) AppendEnum(e *Enum) {
	e.Parent = s
	s.Enums = append(s.Enums, e)
}

func (s *Struct) AppendField(f StructField) {
	f.Parent = s
	s.Fields = append(s.Fields, &f)
}

func (s *Struct) FindEnum(name string) *Enum {
	for _, e := range s.Enums {
		if e.Name == name {
			return e
		}
	}
	return nil
}

func (s *Struct) FindStruct(name string) *Struct {
	for _, st := range s.Structs {
		if st.Name == name {
			return st
		}
	}
	return nil
}

func (s *Struct) FQN() string { return s.BaseFQN() + "." + s.Name }

func (s *Struct) BaseFQN() string {
	var comps []string
	p := s.Parent
	for p != nil {
		comps = append(comps, p.Name)
		p = p.Parent
	}
	comps = append(comps, s.Position.File.Package.Value)
	sort.Sort(sort.Reverse(sort.StringSlice(comps)))
	return strings.Join(comps, ".")
}

type StructField struct {
	Position    Position
	Annotations AnnotationSet
	Comment     []string
	Name        string
	Type        Type
	Parent      *Struct
}

func (*StructField) Kind() string      { return "Struct Field" }
func (s *StructField) Pos() *Position  { return &s.Position }
func (s *StructField) BaseFQN() string { return s.Parent.FQN() }
func (s *StructField) FQN() string     { return s.BaseFQN() + "." + s.Name }

type Enum struct {
	Position    Position
	Annotations AnnotationSet
	Comment     []string
	Name        string
	Members     []*EnumMember
	Parent      *Struct
}

func (*Enum) Kind() string     { return "Enum" }
func (e *Enum) Pos() *Position { return &e.Position }
func (e *Enum) BaseFQN() string {
	if e.Parent != nil {
		return e.Parent.BaseFQN()
	}
	return e.Position.File.BaseFQN()
}

func (e *Enum) AppendMember(i EnumMember) {
	i.Enum = e
	e.Members = append(e.Members, &i)
}

func (e *Enum) FQN() string {
	comps := []string{e.Name}
	p := e.Parent
	for p != nil {
		comps = append(comps, p.Name)
		p = p.Parent
	}
	comps = append(comps, e.Position.File.Package.Value)
	sort.Sort(sort.Reverse(sort.StringSlice(comps)))
	return strings.Join(comps, ".")
}

type EnumMember struct {
	Position    Position
	Comment     []string
	Annotations AnnotationSet
	Name        string
	Value       int
	Enum        *Enum
}

func (*EnumMember) Kind() string      { return "Enum Member" }
func (m *EnumMember) Pos() *Position  { return &m.Position }
func (m *EnumMember) BaseFQN() string { return m.Enum.BaseFQN() }
func (m *EnumMember) FQN() string     { return m.Enum.FQN() + "." + m.Name }

type Annotation struct {
	Position  Position
	Name      string
	Arguments []any
}

func (*Annotation) Kind() string      { return "Annotation" }
func (a *Annotation) Pos() *Position  { return &a.Position }
func (a *Annotation) BaseFQN() string { return a.Position.File.BaseFQN() }
func (a *Annotation) FQN() string     { return a.BaseFQN() }

type AnnotationSet []Annotation

func (a AnnotationSet) ByName(name string) *Annotation {
	for _, a := range a {
		if a.Name == name {
			return &a
		}
	}
	return nil
}

type Service struct {
	Position    Position
	Comment     []string
	Annotations AnnotationSet
	Name        string
	Methods     []*ServiceMethod
}

func (*Service) Kind() string      { return "Service" }
func (s *Service) Pos() *Position  { return &s.Position }
func (s *Service) BaseFQN() string { return s.Position.File.BaseFQN() }
func (s *Service) FQN() string     { return s.BaseFQN() + "." + s.Name }

func (s *Service) AppendMethod(m *ServiceMethod) {
	m.Service = s
	s.Methods = append(s.Methods, m)
}

type ServiceMethod struct {
	Position    Position
	Comment     []string
	Annotations AnnotationSet
	Name        string
	Params      []*MethodParam
	Returns     []*MethodReturn
	Service     *Service
}

func (s *ServiceMethod) AppendParam(p *MethodParam) {
	p.Method = s
	s.Params = append(s.Params, p)
}

func (s *ServiceMethod) AppendReturn(r *MethodReturn) {
	r.Method = s
	s.Returns = append(s.Returns, r)
}

func (*ServiceMethod) Kind() string      { return "Service Method" }
func (s *ServiceMethod) Pos() *Position  { return &s.Position }
func (s *ServiceMethod) BaseFQN() string { return s.Service.BaseFQN() }
func (s *ServiceMethod) FQN() string     { return s.Service.FQN() + "." + s.Name }

type MethodParam struct {
	Position Position
	Stream   bool
	Name     *string
	Type     Type
	Method   *ServiceMethod
}

func (*MethodParam) Kind() string      { return "Method Param" }
func (p *MethodParam) Pos() *Position  { return &p.Position }
func (p *MethodParam) BaseFQN() string { return p.Method.BaseFQN() }
func (p *MethodParam) FQN() string     { return p.Method.BaseFQN() }
func (p *MethodParam) Eql(other *MethodParam) bool {
	if p.Name == nil && other.Name != nil || p.Name != nil && other.Name == nil {
		return false
	}
	if p.Name != nil && other.Name != nil && *p.Name != *other.Name {
		return false
	}
	return p.Stream == other.Stream &&
		p.Type.Eql(other.Type)
}

type MethodReturn struct {
	Position Position
	Type     Type
	Stream   bool
	Method   *ServiceMethod
}

func (*MethodReturn) Kind() string      { return "Method Return" }
func (r *MethodReturn) Pos() *Position  { return &r.Position }
func (r *MethodReturn) BaseFQN() string { return r.Method.BaseFQN() }
func (r *MethodReturn) FQN() string     { return r.Method.BaseFQN() }
func (r *MethodReturn) Eql(other *MethodReturn) bool {
	return r.Type.Eql(other.Type) && r.Stream == other.Stream
}
