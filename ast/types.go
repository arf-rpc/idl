package ast

type Type interface {
	_type()
	Kind() string
	Eql(other Type) bool
}

type ArrayType struct {
	Position Position
	Type     Type
}

func (a *ArrayType) _type() {}

func (*ArrayType) Kind() string { return "Array" }

func (a *ArrayType) Eql(other Type) bool {
	if ot, ok := other.(*ArrayType); ok {
		return a.Type.Eql(ot.Type)
	}
	return false
}

type MapType struct {
	Position   Position
	Key, Value Type
}

func (m *MapType) _type() {}

func (*MapType) Kind() string { return "Map" }

func (m *MapType) Eql(other Type) bool {
	if ot, ok := other.(*MapType); ok {
		return m.Key.Eql(ot.Key) && m.Value.Eql(ot.Value)
	}
	return false
}

type OptionalType struct {
	Position Position
	Type     Type
}

func (o *OptionalType) _type() {}

func (*OptionalType) Kind() string { return "Optional" }

func (o *OptionalType) Eql(other Type) bool {
	if ot, ok := other.(*OptionalType); ok {
		return o.Type.Eql(ot.Type)
	}
	return false
}

type PrimitiveType struct {
	Position Position
	Name     string
}

func (p *PrimitiveType) _type() {}

func (*PrimitiveType) Kind() string { return "Primitive" }

func (p *PrimitiveType) Eql(other Type) bool {
	if ot, ok := other.(*PrimitiveType); ok {
		return p.Name == ot.Name
	}
	return false
}

type SimpleUserType struct {
	Position          Position
	Name              string
	ResolvedType      Object
	FullQualifiedName string
}

func (u *SimpleUserType) _type() {}

func (*SimpleUserType) Kind() string { return "SimpleUser" }

func (u *SimpleUserType) Pos() Position { return u.Position }

func (u *SimpleUserType) SetResolved(obj Object) { u.ResolvedType = obj }

func (u *SimpleUserType) Resolved() Object { return u.ResolvedType }

func (u *SimpleUserType) SetFQN(fqn string) { u.FullQualifiedName = fqn }

func (u *SimpleUserType) FQN() string { return u.FullQualifiedName }

func (u *SimpleUserType) Eql(other Type) bool {
	switch ot := other.(type) {
	case *SimpleUserType:
		return ot.FullQualifiedName == u.FullQualifiedName
	case *FullQualifiedType:
		return ot.FullQualifiedName == u.FullQualifiedName
	default:
		return false
	}
}

type FullQualifiedType struct {
	Position          Position
	Package           string
	Name              string
	FullName          string
	Components        []string
	ResolvedType      Object
	FullQualifiedName string
}

func (q *FullQualifiedType) _type() {}

func (*FullQualifiedType) Kind() string { return "FullQualified" }

func (q *FullQualifiedType) Pos() Position { return q.Position }

func (q *FullQualifiedType) SetResolved(obj Object) { q.ResolvedType = obj }

func (q *FullQualifiedType) Resolved() Object { return q.ResolvedType }

func (q *FullQualifiedType) SetFQN(fqn string) { q.FullQualifiedName = fqn }

func (q *FullQualifiedType) FQN() string { return q.FullQualifiedName }

func (q *FullQualifiedType) Eql(other Type) bool {
	switch ot := other.(type) {
	case *SimpleUserType:
		return ot.FullQualifiedName == q.FullQualifiedName
	case *FullQualifiedType:
		return ot.FullQualifiedName == q.FullQualifiedName
	default:
		return false
	}
}

type ResolvableType interface {
	Type
	Pos() Position
	SetResolved(Object)
	Resolved() Object
	SetFQN(fqn string)
	FQN() string
}
