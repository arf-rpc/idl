package idl

import (
	"fmt"
	"github.com/arf-rpc/idl/ast"
)

type def struct {
	path   string
	offset ast.Offset
}

func validate(p *Parser) error {
	v := validator{
		p:           p,
		rootObjects: map[string]def{},
	}
	if err := v.processTreeAnnotations(); err != nil {
		return err
	}

	if err := v.processMethodDefinitions(); err != nil {
		return err
	}

	if err := v.processClashes(); err != nil {
		return err
	}

	return nil
}

type validator struct {
	p           *Parser
	rootObjects map[string]def
}

func (v *validator) processTreeAnnotations() error {
	for _, s := range v.p.set {
		tree := s.Tree
		for _, e := range tree.Enums {
			if err := v.processEnumAnnotations(s, e); err != nil {
				return err
			}
		}

		for _, sv := range tree.Services {
			if err := v.processServiceAnnotations(s, sv); err != nil {
				return err
			}
		}

		for _, st := range tree.Structures {
			if err := v.processStructureAnnotations(s, st); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *validator) processEnumAnnotations(file ast.File, s *ast.Enum) error {
	for _, a := range s.Annotations {
		if a.Name == "repeated" || a.Name == "optional" {
			return fmt.Errorf("%s:%d:%d: enum %s cannot be annotated with %s", file.Path, s.Offset.StartsAt.Line, s.Offset.StartsAt.Column, s.Path(), a.Name)
		}
	}

	for _, o := range s.Options {
		if err := v.processEnumOptionAnnotation(file, o); err != nil {
			return err
		}
	}

	return nil
}

func (v *validator) processServiceAnnotations(file ast.File, s *ast.Service) error {
	for _, a := range s.Annotations {
		if a.Name == "repeated" || a.Name == "optional" {
			return fmt.Errorf("%s:%d:%d: service %s cannot be annotated with %s", file.Path, s.Offset.StartsAt.Line, s.Offset.StartsAt.Column, s.Path(), a.Name)
		}
	}

	for _, m := range s.Methods {
		if err := v.processServiceMethodAnnotations(file, m); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) processStructureAnnotations(file ast.File, s *ast.Struct) error {
	for _, a := range s.Annotations {
		if a.Name == "repeated" || a.Name == "optional" {
			return fmt.Errorf("%s:%d:%d: struct %s cannot be annotated with %s", file.Path, s.Offset.StartsAt.Line, s.Offset.StartsAt.Column, s.Path(), a.Name)
		}
	}

	for _, e := range s.Enums {
		if err := v.processEnumAnnotations(file, e); err != nil {
			return err
		}
	}

	for _, s := range s.Structs {
		if err := v.processStructureAnnotations(file, s); err != nil {
			return err
		}
	}

	for _, f := range s.Fields {
		if err := v.processStructureFieldAnnotations(file, f); err != nil {
			return err
		}
	}

	return nil
}

func (v *validator) processEnumOptionAnnotation(file ast.File, o *ast.EnumOption) error {
	for _, a := range o.Annotations {
		if a.Name == "repeated" || a.Name == "optional" {
			return fmt.Errorf("%s:%d:%d: enum option %s cannot be annotated with %s", file.Path, o.Offset.StartsAt.Line, o.Offset.StartsAt.Column, o.Path(), a.Name)
		}
	}

	return nil
}

func (v *validator) processServiceMethodAnnotations(file ast.File, m *ast.Method) error {
	for _, a := range m.Annotations {
		if a.Name == "repeated" || a.Name == "optional" {
			return fmt.Errorf("%s:%d:%d: method %s cannot be annotated with %s", file.Path, m.Offset.StartsAt.Line, m.Offset.StartsAt.Column, m.Path(), a.Name)
		}
	}
	return nil
}

func (v *validator) processStructureFieldAnnotations(file ast.File, f *ast.Field) error {
	optional := f.Annotations.ByName("optional")
	repeated := f.Annotations.ByName("repeated")

	if optional != nil && f.Union != nil {
		return fmt.Errorf("%s:%d:%d: union cannot be annotated as optional", file.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column)
	}

	if repeated != nil && f.Union != nil {
		return fmt.Errorf("%s:%d:%d: union cannot be annotated as repeated", file.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column)
	}

	if repeated != nil && optional != nil {
		return fmt.Errorf("%s:%d:%d: field %s cannot be annotated with both optional and repeated", file.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column, f.Path())
	}

	if optional != nil && len(optional.Arguments) != 0 {
		return fmt.Errorf("%s:%d:%d: optional annotation on field %s does not take parameters", file.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column, f.Path())
	}

	if repeated != nil && len(repeated.Arguments) != 0 {
		return fmt.Errorf("%s:%d:%d: repeated annotation on field %s does not take parameters", file.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column, f.Path())
	}

	if f.Union != nil {
		for _, f := range f.Union.Fields {
			if err := v.processUnionFieldAnnotations(file, f); err != nil {
				return err
			}
		}
	} else {
		if optional != nil {
			f.Plain.Type = f.Plain.Type.Optional()
		}
		if repeated != nil {
			f.Plain.Type = f.Plain.Type.Repeated()
		}
	}

	return nil
}

func (v *validator) processUnionFieldAnnotations(file ast.File, f *ast.Field) error {
	optional := f.Annotations.ByName("optional")
	repeated := f.Annotations.ByName("repeated")

	if optional != nil {
		return fmt.Errorf("%s:%d:%d: union field %s cannot be annotated as optional", file.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column, f.Path())
	}

	if repeated != nil {
		return fmt.Errorf("%s:%d:%d: union field %s cannot be annotated as repeated", file.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column, f.Path())
	}

	return nil
}

func (v *validator) processClashes() error {
	for _, s := range v.p.set {
		tree := s.Tree

		for _, e := range tree.Enums {
			if other, ok := v.rootObjects[e.Name]; ok {
				return fmt.Errorf("name %s already defined in %s, line %d, column %d", e.Name, other.path, other.offset.StartsAt.Line, other.offset.StartsAt.Column)
			}
			v.rootObjects[e.Name] = def{s.Path, e.Offset}

			if err := v.processEnumClashes(s, e); err != nil {
				return err
			}
		}

		for _, serv := range tree.Services {
			if other, ok := v.rootObjects[serv.Name]; ok {
				return fmt.Errorf("name %s already defined in %s, line %d, column %d", serv.Name, other.path, other.offset.StartsAt.Line, other.offset.StartsAt.Column)
			}
			v.rootObjects[serv.Name] = def{s.Path, serv.Offset}

			if err := v.processServiceClashes(s, serv); err != nil {
				return err
			}
		}

		for _, str := range tree.Structures {
			if other, ok := v.rootObjects[str.Name]; ok {
				return fmt.Errorf("name %s already defined in %s, line %d, column %d", str.Name, other.path, other.offset.StartsAt.Line, other.offset.StartsAt.Column)
			}
			v.rootObjects[str.Name] = def{s.Path, str.Offset}
			if err := v.processStructureClashes(s, str); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *validator) processEnumClashes(s ast.File, e *ast.Enum) error {
	known := map[string]bool{}
	for _, option := range e.Options {
		if known[option.Name] {
			return fmt.Errorf("%s:%d:%d: enum %s has a duplicated option %s", s.Path, option.Offset.StartsAt.Line, option.Offset.StartsAt.Column, e.Name, option.Name)
		}
		known[option.Name] = true
	}

	return nil
}

func (v *validator) processServiceClashes(s ast.File, serv *ast.Service) error {
	known := map[string]bool{}
	for _, method := range serv.Methods {
		if known[method.Name] {
			return fmt.Errorf("%s:%d:%d: enum %s has a duplicated option %s", s.Path, method.Offset.StartsAt.Line, method.Offset.StartsAt.Column, serv.Name, method.Name)
		}
		known[method.Name] = true
	}

	return nil
}

func (v *validator) processStructureClashes(s ast.File, str *ast.Struct) error {
	rootObjects := map[string]bool{}
	fields := map[string]bool{}
	indexes := map[int]bool{}

	for _, st := range str.Structs {
		if rootObjects[st.Name] {
			return fmt.Errorf("%s:%d:%d: struct %s has a duplicated nested object %s", s.Path, st.Offset.StartsAt.Line, st.Offset.StartsAt.Column, str.Name, st.Name)
		}
		rootObjects[st.Name] = true
	}

	for _, en := range str.Enums {
		if rootObjects[en.Name] {
			return fmt.Errorf("%s:%d:%d: struct %s has a duplicated nested object %s", s.Path, en.Offset.StartsAt.Line, en.Offset.StartsAt.Column, str.Name, en.Name)
		}
		rootObjects[en.Name] = true
	}

	for _, f := range str.Fields {
		if f.Plain != nil {
			if fields[f.Plain.Name] {
				return fmt.Errorf("%s:%d:%d: struct %s has a duplicated field %s", s.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column, str.Name, f.Plain.Name)
			}
			fields[f.Plain.Name] = true
			if indexes[f.Plain.Index] {
				return fmt.Errorf("%s:%d:%d: struct %s has a duplicated index %d", s.Path, f.Offset.StartsAt.Line, f.Offset.StartsAt.Column, str.Name, f.Plain.Index)
			}
			indexes[f.Plain.Index] = true
		} else {
			for _, v := range f.Union.Fields {
				if fields[v.Plain.Name] {
					return fmt.Errorf("%s:%d:%d: struct %s has a duplicated field %s", s.Path, v.Offset.StartsAt.Line, v.Offset.StartsAt.Column, str.Name, v.Plain.Name)
				}
				fields[v.Plain.Name] = true
				if indexes[v.Plain.Index] {
					return fmt.Errorf("%s:%d:%d: struct %s has a duplicated index %d", s.Path, v.Offset.StartsAt.Line, v.Offset.StartsAt.Column, str.Name, v.Plain.Index)
				}
				indexes[v.Plain.Index] = true
			}
		}
	}

	return nil
}

func (v *validator) processMethodDefinitions() error {
	for _, f := range v.p.set {
		for _, s := range f.Tree.Services {
			for _, m := range s.Methods {
				if err := v.processMethodDefinition(m); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (v *validator) processMethodDefinition(m *ast.Method) error {

	// Simple rules:
	// 1. Either no argument is named, or all of them are named.
	// 2. Only one stream allowed in either input or output values.
	// 3. If present, a stream must be the last argument of input/output values.

	isNamed := false
	for _, f := range m.Input {
		isNamed = isNamed || f.Named
	}

	for i, f := range m.Input {
		if !f.Named && isNamed {
			return fmt.Errorf("argument %d of %s must be named (all arguments must be either named, or not)", i, m.Path())
		}
	}

	streamCount := 0
	for i, f := range m.Input {
		if _, ok := f.Type.(*ast.StreamingType); ok {
			streamCount++
			if streamCount > 1 {
				return fmt.Errorf("argument %d of %s is invalid: only one streaming argument is allowed", i, m.Path())
			}
			continue
		}
		if streamCount > 0 {
			return fmt.Errorf("argument %d of %s is invalid: stream arguments must be the last argument in a method definition", i, m.Path())
		}
	}

	streamCount = 0
	for i, f := range m.Output {
		if _, ok := f.(*ast.StreamingType); ok {
			streamCount++
			if streamCount > 1 {
				return fmt.Errorf("output value %d of %s is invalid: only one streaming output value is allowed", i, m.Path())
			}
			continue
		}
		if streamCount > 0 {
			return fmt.Errorf("output value %d of %s is invalid: stream value must be the last value in a method definition", i, m.Path())
		}
	}

	return nil
}
