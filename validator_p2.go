package idl

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/arf-rpc/idl/ast"
)

func validatePhase2(files map[string]*ast.File, entrypoint string) error {
	f, ok := files[entrypoint]
	if !ok {
		return fmt.Errorf("BUG: validation entrypoint %s not found", entrypoint)
	}

	v := &validatorP2{
		files:  files,
		errors: nil,
		f:      f,
	}

	for _, s := range f.Structs {
		v.validateStruct(s)
	}

	// No need to validate enums, as they are not allowed to reference other types.

	for _, s := range f.Services {
		v.validateService(s)
	}

	return errors.Join(v.errors...)
}

type validatorP2 struct {
	files  map[string]*ast.File
	errors []error
	f      *ast.File
}

func (v *validatorP2) Errorf(format string, args ...interface{}) {
	v.errors = append(v.errors, fmt.Errorf(format, args...))
}

func (v *validatorP2) validateStruct(s *ast.Struct) {
	for _, ss := range s.Structs {
		v.validateStruct(ss)
	}

	for _, f := range s.Fields {
		v.resolveType(s, f.Type)
	}

	// No need to validate enums, as they are not allowed to reference other types.
}

func (v *validatorP2) resolveType(parent ast.Object, t ast.Type) {
	switch tt := t.(type) {
	case *ast.OptionalType:
		v.resolveType(parent, tt.Type)
	case *ast.ArrayType:
		v.resolveType(parent, tt.Type)
	case *ast.MapType:
		v.resolveType(parent, tt.Key)
		v.resolveType(parent, tt.Value)
		v.validateMapKey(tt)
	case *ast.SimpleUserType:
		v.preResolveType(parent, tt.Name, tt)
	case *ast.FullQualifiedType:
		v.preResolveType(parent, tt.FullName, tt)
	case *ast.PrimitiveType:
		// NOOP
	default:
		v.Errorf("Bug: Invalid type %T", tt)
	}
}

func (v *validatorP2) preResolveType(parent ast.Object, name string, rt ast.ResolvableType) {
	var obj ast.Object
	if str, ok := parent.(*ast.Struct); ok {
		obj = v.lookupType(str, name)
	} else {
		obj = v.lookupType(v.f, name)
	}

	if obj == nil {
		pos := rt.Pos()
		v.Errorf("Undefined type %s at %s, line %d, column %d", name, pos.Filename, pos.Line, pos.Column)
		return
	}

	rt.SetResolved(obj)
	rt.SetFQN(obj.FQN())
}

func (v *validatorP2) lookupType(parent ast.Container, name string) ast.Object {
	components := strings.Split(name, ".")

	// If the first component starts with a lower case, it must be referencing
	// an alias. Just make sure to check if it's also not referencing the
	// same local compilation unit.
	if unicode.IsLower([]rune(components[0])[0]) {
		if alias, ok := v.f.ImportAliases[components[0]]; ok {
			components[0] = v.files[alias].Package.Value
		} else if components[0] == v.f.Package.Components[0] {
			components[0] = v.f.Package.Value
		}

		// At this point, all types are already loaded, so we can try to resolve
		// it.
		components = strings.Split(strings.Join(components, "."), ".")
		obj := v.lookupFQN(components)
		if obj != nil {
			return obj
		}
	}

	if len(components) == 1 {
		if e := v.f.FindEnum(components[0]); e != nil {
			return e
		}
		if s := v.f.FindStruct(components[0]); s != nil {
			return s
		}
	}

	// Nothing so far, and the type is not a single component. First try to
	// resolve it against the current container, then against the package.

	if obj := v.findScopedType(parent, components); obj != nil {
		return obj
	}

	components = strings.Split(strings.Join(append([]string{v.f.Package.Value}, components...), "."), ".")
	return v.lookupFQN(components)
}

func (v *validatorP2) findScopedType(ctx ast.Container, components []string) ast.Object {
	comp := components
	for {
		switch len(comp) {
		case 1:
			if e := ctx.FindEnum(comp[0]); e != nil {
				return e
			}
			if s := ctx.FindStruct(comp[0]); s != nil {
				return s
			}
			return nil
		default:
			next := ctx.FindStruct(comp[0])
			if next == nil {
				return nil
			}
			ctx = next
			comp = comp[1:]
		}
	}
}

func (v *validatorP2) findPackage(name string) *ast.File {
	for _, f := range v.files {
		if f.Package.Value == name {
			return f
		}
	}
	return nil
}

func (v *validatorP2) lookupFQN(components []string) ast.Object {
	var target ast.Container
	var i int
	for i = range components {
		fullPkg := strings.Join(components[:i+1], ".")

		if p := v.findPackage(fullPkg); p != nil {
			target = p
			break
		}
		if v.f.Package.Value == fullPkg {
			target = v.f
			break
		}
	}
	if target == nil {
		return nil
	}
	name := components[i+1:]

	for {
		switch len(name) {
		case 1:
			if e := target.FindEnum(name[0]); e != nil {
				return e
			}
			if s := target.FindStruct(name[0]); s != nil {
				return s
			}
			return nil
		default:
			next := target.FindStruct(name[0])
			if next == nil {
				return nil
			}
			target = next
			name = name[1:]
		}
	}
}

func (v *validatorP2) validateMapKey(m *ast.MapType) {
	switch t := m.Key.(type) {
	case ast.ResolvableType:
		// NOOP
	case *ast.PrimitiveType:
		// NOOP
	case *ast.OptionalType:
		v.invalidMapKeyType(t, m)
	case *ast.ArrayType:
		v.invalidMapKeyType(t, m)
	case *ast.MapType:
		v.invalidMapKeyType(t, m)
	}
}

func (v *validatorP2) invalidMapKeyType(t ast.Type, m *ast.MapType) {
	pos := m.Position
	v.Errorf("Cannot use %s as a map key at %s, line %d, column %d", t.Kind(), pos.Filename, pos.Line, pos.Column)
}

func (v *validatorP2) validateService(s *ast.Service) {
	// At this point, the service has passed initial validation, so we can
	// focus on type checks for each of its methods.

	for _, m := range s.Methods {
		v.validateMethod(m)
	}
}

func (v *validatorP2) validateMethod(m *ast.ServiceMethod) {
	for _, p := range m.Params {
		v.validateMethodParam(p.Type, &p.Position)
	}
	for _, p := range m.Returns {
		v.validateMethodParam(p.Type, &p.Position)
	}
}

func (v *validatorP2) validateMethodParam(t ast.Type, pos *ast.Position) {
	switch tt := t.(type) {
	case ast.ResolvableType:
		v.resolveType(v.f, tt)
	default:
		v.Errorf("Types used within methods are required to be user-defined structures. Cannot use %s at %s, line %d, column %d", t.Kind(), pos.Filename, pos.Line, pos.Column)
	}
}
