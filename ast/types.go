package ast

type Type interface {
	_type()
	Kind() string
}

type ArrayType struct {
	Position Position
	Type     Type
}

func (a *ArrayType) _type() {}

func (*ArrayType) Kind() string { return "Array" }

type MapType struct {
	Position   Position
	Key, Value Type
}

func (m *MapType) _type() {}

func (*MapType) Kind() string { return "Map" }

type OptionalType struct {
	Position Position
	Type     Type
}

func (o *OptionalType) _type() {}

func (*OptionalType) Kind() string { return "Optional" }

type PrimitiveType struct {
	Position Position
	Name     string
}

func (p *PrimitiveType) _type() {}

func (*PrimitiveType) Kind() string { return "Primitive" }

type SimpleUserType struct {
	Position     Position
	Name         string
	ResolvedType Object
}

func (u *SimpleUserType) _type() {}

func (*SimpleUserType) Kind() string { return "SimpleUser" }

type FullQualifiedType struct {
	Position     Position
	Package      string
	Name         string
	FullName     string
	Components   []string
	ResolvedType Object
}

func (q *FullQualifiedType) _type() {}

func (*FullQualifiedType) Kind() string { return "FullQualified" }
