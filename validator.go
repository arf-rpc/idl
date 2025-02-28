package idl

import (
	"fmt"
	"github.com/arf-rpc/idl/ast"
	"strings"
)

type typeCheck struct {
	t      ast.Type
	source ast.Object
}

type mapKeyCheck struct {
	t      ast.Type
	key    *ast.MapType
	source ast.Object
}

type validator struct {
	objectsPos          map[string]*ast.Position
	objects             map[string]ast.Object
	errs                []error
	deferredTypeCheck   []*typeCheck
	deferredMapKeyCheck []*mapKeyCheck
}

func newValidator() *validator {
	return &validator{
		objectsPos:          make(map[string]*ast.Position),
		objects:             make(map[string]ast.Object),
		errs:                nil,
		deferredTypeCheck:   nil,
		deferredMapKeyCheck: nil,
	}
}

func (v *validator) errorf(format string, args ...interface{}) {
	v.errs = append(v.errs, fmt.Errorf(format, args...))
}

func (v *validator) nameClash(fqn string, pos *ast.Position) {
	comps := strings.Split(fqn, ".")
	name := comps[len(comps)-1]
	v.errorf("%s is already defined at %s, line %d, column %d", name, pos.Filename, pos.Line, pos.Column)
}

func (v *validator) structFieldClash(f ast.StructField, pos *ast.Position) {
	v.errorf("%s is already defined for %s at line %d, column %d", f.Name, pos.Filename, pos.Line, pos.Column)
}

func (v *validator) validateFile(f *ast.File) {
	for _, s := range f.Structs {
		v.validateStruct(&s)
	}
	for _, e := range f.Enums {
		v.validateEnum(&e)
	}
	for _, s := range f.Services {
		v.validateService(&s)
	}
}

func (v *validator) validateStruct(s *ast.Struct) {
	fqn := s.FQN()
	if pos, ok := v.objectsPos[fqn]; ok {
		v.nameClash(fqn, pos)
	} else {
		v.objectsPos[fqn] = pos
		v.objects[fqn] = s
	}

	fs := map[string]*ast.Position{}
	for _, f := range s.Fields {
		if pos, ok := fs[f.Name]; ok {
			v.structFieldClash(f, pos)
		} else {
			fs[f.Name] = pos
		}
	}

	for _, f := range s.Fields {
		v.validateType(f.Type, &f)
	}

	for _, ss := range s.Structs {
		v.validateStruct(&ss)
	}

	for _, e := range s.Enums {
		v.validateEnum(&e)
	}
}

func (v *validator) validateEnum(e *ast.Enum) {
	f := e.FQN()
	if pos, ok := v.objectsPos[f]; ok {
		v.nameClash(e.Name, pos)
	} else {
		v.objectsPos[f] = pos
		v.objects[f] = e
	}

	fs := map[string]*ast.Position{}
	if len(e.Members) == 0 {
		v.errorf("Enum %s must have at least one member at %s, line %d, column %d", e.Name, e.Position.Filename, e.Position.Line, e.Position.Column)
		return
	}

	for _, f := range e.Members {
		if pos, ok := fs[f.Name]; ok {
			v.nameClash(f.Name, pos)
		} else {
			fs[f.Name] = pos
		}
	}
}

func (v *validator) validateService(s *ast.Service) {
	fqn := s.FQN()
	if pos, ok := v.objectsPos[fqn]; ok {
		v.nameClash(fqn, pos)
	} else {
		v.objectsPos[fqn] = pos
		v.objects[fqn] = s
	}

	methods := map[string]*ast.Position{}
	for _, m := range s.Methods {
		if pos, ok := methods[m.Name]; ok {
			v.nameClash(m.Name, pos)
		} else {
			methods[m.Name] = pos
		}

		v.validateMethod(m)
	}
}

func (v *validator) validateMethod(m *ast.ServiceMethod) {
	inputNames := map[string]struct{}{}
	hasStreamingInput := false
	for _, p := range m.Params {
		if p.Name != nil {
			if _, ok := inputNames[*p.Name]; ok {
				v.errorf("duplicate parameter name %s for method %s at %s, line %d, column %d", *p.Name, m.Name, p.Position.Filename, p.Position.Line, p.Position.Column)
			}
		}

		if p.Stream && hasStreamingInput {
			v.errorf("method %s can only have one stream param at %s, line %d, column %d", m.Name, p.Position.Filename, p.Position.Line, p.Position.Column)
		} else if p.Stream {
			hasStreamingInput = true
		}

		switch p.Type.(type) {
		case *ast.SimpleUserType:
			v.deferredTypeCheck = append(v.deferredTypeCheck, &typeCheck{
				t:      p.Type,
				source: p,
			})
		case *ast.FullQualifiedType:
			v.deferredTypeCheck = append(v.deferredTypeCheck, &typeCheck{
				t:      p.Type,
				source: p,
			})
		}
	}

	hasStreamingOutput := false
	for _, r := range m.Returns {
		if r.Stream && hasStreamingOutput {
			v.errorf("method %s can only have one stream return at %s, line %d, column %d", m.Name, r.Position.Filename, r.Position.Line, r.Position.Column)
		} else if r.Stream {
			hasStreamingOutput = true
		}

		switch r.Type.(type) {
		case *ast.SimpleUserType:
			v.deferredTypeCheck = append(v.deferredTypeCheck, &typeCheck{
				t:      r.Type,
				source: r,
			})
		case *ast.FullQualifiedType:
			v.deferredTypeCheck = append(v.deferredTypeCheck, &typeCheck{
				t:      r.Type,
				source: r,
			})
		}
	}
}

func (v *validator) validateType(t ast.Type, source ast.Object) {
	switch tt := t.(type) {
	case *ast.MapType:
		v.validateMapKey(tt, source)
		v.validateType(tt.Value, source)
	case *ast.ArrayType:
		v.validateType(tt.Type, source)
	case *ast.OptionalType:
		v.validateType(tt.Type, source)
	case *ast.PrimitiveType:
		// Noop
	case *ast.SimpleUserType:
		v.deferredTypeCheck = append(v.deferredTypeCheck, &typeCheck{t: t, source: source})
	case *ast.FullQualifiedType:
		v.deferredTypeCheck = append(v.deferredTypeCheck, &typeCheck{t: t, source: source})
	default:
		v.errorf("unknown type %v", t)
	}
}

func (v *validator) validateMapKey(tt *ast.MapType, source ast.Object) {
	switch t := tt.Key.(type) {
	case *ast.PrimitiveType:
		if t.Name == "bytes" {
			v.invalidMapKey(t, source)
		}
	case *ast.SimpleUserType:
		v.deferredMapKeyCheck = append(v.deferredMapKeyCheck, &mapKeyCheck{
			t:      t,
			key:    tt,
			source: source,
		})
	case *ast.FullQualifiedType:
		v.deferredMapKeyCheck = append(v.deferredMapKeyCheck, &mapKeyCheck{
			t:      t,
			key:    tt,
			source: source,
		})
	case *ast.MapType:
		v.invalidMapKey(t, source)
	case *ast.OptionalType:
		v.invalidMapKey(t, source)
	}
}

func (v *validator) invalidMapKey(t ast.Type, source ast.Object) {
	switch src := source.(type) {
	case *ast.StructField:
		v.errorf("%s cannot be used as a map key on struct %s, field %s, at %s line %d, column %d",
			t.Kind(), src.Parent.Name, src.Name, src.Position.Filename, src.Position.Line, src.Position.Column)

	case *ast.MethodParam:
		if src.Name == nil {
			v.errorf("%s cannot be used as a map key on anonymous method param on method %s of service %s, at %s line %d, column %d",
				t.Kind(), src.Method.Name, src.Method.Service.Name, src.Position.Filename, src.Position.Line, src.Position.Column)
		} else {
			v.errorf("%s cannot be used as a map key on method param %s on method %s of service %s, at %s line %d, column %d",
				t.Kind(), *src.Name, src.Method.Name, src.Method.Service.Name, src.Position.Filename, src.Position.Line, src.Position.Column)
		}

	case *ast.MethodReturn:
		v.errorf("%s cannot be used as a map key on method result on method %s of service %s, at %s line %d, column %d",
			t.Kind(), src.Method.Name, src.Method.Service.Name, src.Position.Filename, src.Position.Line, src.Position.Column)
	}
}

func (v *validator) findObject(name string) ast.Object {
	comps := strings.Split(name, ".")
	name = comps[len(comps)-1]
	comps = comps[:len(comps)-1]

	for len(comps) > 0 {
		lookup := strings.Join(append(comps, name), ".")
		if obj, ok := v.objects[lookup]; ok {
			return obj
		}
		comps = comps[:len(comps)-1]
	}

	return nil
}

func (v *validator) lookupType(source ast.Object, t ast.Type) (string, ast.Object) {
	var (
		name string
		obj  ast.Object
	)
	switch tt := t.(type) {
	case *ast.SimpleUserType:
		name = tt.Name
		if tt.ResolvedType == nil {
			tt.ResolvedType = v.findObject(source.BaseFQN() + "." + tt.Name)
		}
		obj = tt.ResolvedType
	case *ast.FullQualifiedType:
		name = tt.FullName
		if tt.ResolvedType == nil {
			tt.ResolvedType = v.findObject(tt.FullName)
			if tt.ResolvedType == nil {
				// Vito: This case allows an edge case where a type may be indicated as
				// Foo.Bar, but be referring to a type in the current package. Considering
				// how we want this to work, it's a weird construction, but well... /shrug
				tt.ResolvedType = v.findObject(source.BaseFQN() + "." + tt.FullName)
			}
		}
		obj = tt.ResolvedType
	}

	return name, obj
}

func (v *validator) runDeferredChecks() {
	for _, t := range v.deferredTypeCheck {
		name, obj := v.lookupType(t.source, t.t)
		if obj == nil {
			p := t.source.Pos()
			v.errorf("Unknown type %s at %s, line %d, column %d", name, p.Filename, p.Line, p.Column)
			continue
		}

		switch obj.(type) {
		case *ast.Struct:
		case *ast.Enum:
		default:
			p := t.source.Pos()
			v.errorf("Cannot use %s as a type at %s, line %d, column %d", obj.Kind(), p.Filename, p.Line, p.Column)
		}
	}

	for _, t := range v.deferredMapKeyCheck {
		name, obj := v.lookupType(t.source, t.t)

		if obj == nil {
			p := t.source.Pos()
			v.errorf("Unknown type %s at %s, line %d, column %d", name, p.Filename, p.Line, p.Column)
			continue
		}

		switch obj.(type) {
		case *ast.Enum:
		default:
			p := t.source.Pos()
			v.errorf("Cannot use %s as a type at %s, line %d, column %d", obj.Kind(), p.Filename, p.Line, p.Column)
		}
	}
}

func (v *validator) runLoopChecks(f *ast.File) {
	for _, s := range f.Structs {
		v.runStructLoopCheck(&s)
	}
}

func (v *validator) runStructLoopCheck(s *ast.Struct) {
	for _, f := range s.Fields {
		_, obj := v.lookupType(&f, f.Type)
		if obj == nil {
			continue
		}

		if obj.FQN() == s.FQN() {
			v.errorf("%s cannot reference itself as a type at %s, line %d, column %d", s.Name, f.Position.Filename, f.Position.Line, f.Position.Column)
			continue
		}

		if ss, ok := obj.(*ast.Struct); ok {
			// Check if ss makes direct reference to s
			for _, ff := range ss.Fields {
				_, ssObj := v.lookupType(ss, ff.Type)
				if ssObj == nil {
					continue
				}
				if ssObj.FQN() == s.FQN() {
					v.errorf("%s cannot directly reference %s as it would create a cyclic reference at %s, line %d, column %d",
						s.Name, ss.Name, f.Position.Filename, f.Position.Line, f.Position.Column)
				}
			}
		}
	}
}
