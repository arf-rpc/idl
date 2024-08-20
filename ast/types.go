package ast

import "fmt"

//go:generate stringer -type=PrimitiveType -output=types_string.go

type Type interface {
	Streaming() Type
	Optional() Type
	Repeated() Type
}

type UserType struct {
	Name         string
	ResolvedType any
}

func (u *UserType) String() string {
	return fmt.Sprintf("UserType{%s}", u.Name)
}

func (u *UserType) Streaming() Type {
	return &StreamingType{u}
}

func (u *UserType) Optional() Type {
	return &OptionalType{u}
}

func (u *UserType) Repeated() Type {
	return &RepeatedType{u}
}

type MapType struct {
	Key, Value Type
}

func (m *MapType) String() string {
	return fmt.Sprintf("MapType{%s, %s}", m.Key, m.Value)
}

func (m *MapType) Streaming() Type {
	return &StreamingType{m}
}

func (m *MapType) Optional() Type {
	return &OptionalType{m}
}

func (m *MapType) Repeated() Type {
	return &RepeatedType{m}
}

type StreamingType struct {
	Type Type
}

func (s *StreamingType) String() string {
	return fmt.Sprintf("StreamingType{%s}", s.Type)
}

func (s *StreamingType) Streaming() Type {
	return s
}

func (s *StreamingType) Optional() Type {
	panic("StreamingType cannot be made optional")
}

func (s *StreamingType) Repeated() Type {
	panic("StreamingType cannot be made repeated")
}

type OptionalType struct {
	Type Type
}

func (o *OptionalType) String() string {
	return fmt.Sprintf("OptionalType{%s}", o.Type)
}

func (o *OptionalType) Streaming() Type {
	panic("OptionalType cannot be made streaming")
}

func (o *OptionalType) Optional() Type {
	panic("OptionalType cannot be made optional")
}

func (o *OptionalType) Repeated() Type {
	panic("OptionalType cannot be made repeated")
}

type RepeatedType struct {
	Type Type
}

func (r *RepeatedType) String() string {
	return fmt.Sprintf("RepeatedType{%s}", r.Type)
}

func (r *RepeatedType) Streaming() Type {
	return &StreamingType{r}
}

func (r *RepeatedType) Optional() Type {
	panic("RepeatedType cannot be made optional")
}

func (r *RepeatedType) Repeated() Type {
	return r
}

// PrimitiveType represents a single type recognized by ARF
type PrimitiveType int

func (i PrimitiveType) Streaming() Type {
	return &StreamingType{Type: i}
}

func (i PrimitiveType) Optional() Type {
	return &OptionalType{Type: i}
}

func (i PrimitiveType) Repeated() Type {
	return &RepeatedType{Type: i}
}

const (
	Invalid PrimitiveType = iota
	Uint8
	Uint16
	Uint32
	Uint64
	Int8
	Int16
	Int32
	Int64
	Float32
	Float64
	Bool
	String
	Bytes
)

func IntoType(name string) Type {
	switch name {
	case "uint8":
		return Uint8
	case "uint16":
		return Uint16
	case "uint32":
		return Uint32
	case "uint64":
		return Uint64
	case "int8":
		return Int8
	case "int16":
		return Int16
	case "int32":
		return Int32
	case "int64":
		return Int64
	case "float32":
		return Float32
	case "float64":
		return Float64
	case "bool":
		return Bool
	case "string":
		return String
	case "bytes":
		return Bytes
	default:
		return &UserType{Name: name}
	}
}
