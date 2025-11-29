package idl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arf-rpc/idl/ast"
)

func validatePhase1(files map[string]*ast.File, entrypoint string) error {
	f, ok := files[entrypoint]
	if !ok {
		return fmt.Errorf("BUG: validation entrypoint %s not found", entrypoint)
	}

	v := &validatorP1{
		files:      files,
		errors:     nil,
		objectsPos: make(map[string]*ast.Position),
		objects:    make(map[string]ast.Object),
		f:          f,
	}

	v.processImports()
	if v.errors != nil {
		return errors.Join(v.errors...)
	}

	for _, s := range f.Structs {
		v.validateStruct(s)
	}

	for _, e := range f.Enums {
		v.validateEnum(e)
	}

	for _, s := range f.Services {
		v.detectDuplicatedService(s)
	}

	return errors.Join(v.errors...)
}

type validatorP1 struct {
	files      map[string]*ast.File
	errors     []error
	objectsPos map[string]*ast.Position
	objects    map[string]ast.Object
	f          *ast.File
}

func (p *validatorP1) Errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Errorf(format, args...))
}

func (p *validatorP1) processImports() {
	for _, imp := range p.f.Imports {
		// TODO: defineImportAlias should return whether the name was synthetised
		//       so we can improve the error message
		p.defineImportAlias(imp)
		if _, ok := p.f.ImportAliases[imp.Alias]; ok {
			pos := imp.Pos()
			p.Errorf("duplicate import alias %s at %s, line %d, column %d", imp.Alias, p.f.Path, pos.Line, pos.Column)
			continue
		}
		p.f.ImportAliases[imp.Alias] = imp.ResolvedValue
	}
}

func (p *validatorP1) nameClash(fqn string, pos *ast.Position) {
	comps := strings.Split(fqn, ".")
	name := comps[len(comps)-1]
	p.Errorf("%s is already defined at %s, line %d, column %d", name, pos.Filename, pos.Line, pos.Column)
}

func (p *validatorP1) structFieldClash(f *ast.StructField, pos *ast.Position) {
	p.Errorf("%s is already defined for %s at line %d, column %d", f.Name, pos.Filename, pos.Line, pos.Column)
}

func (p *validatorP1) detectDuplicatedService(s *ast.Service) {
	fqn := s.FQN()
	if ex, ok := p.objects[fqn]; ok {
		p.nameClash(fqn, ex.Pos())
		return
	}

	p.objects[fqn] = s

	// We don't check for duplicated methods here, as we need resolved types
	// to make sure duplicated methods are divergent.
	for _, m := range s.Methods {
		p.validateMethodParams(m)
	}
}

func (p *validatorP1) validateMethodParams(m *ast.ServiceMethod) {
	inputNames := makeSet[string]()
	hasStreamingInput := false
	for _, param := range m.Params {
		if param.Name != nil {
			if inputNames.has(*param.Name) {
				p.Errorf("duplicate parameter name %s for method %s at %s, line %d, column %d", *param.Name, m.Name, param.Position.Filename, param.Position.Line, param.Position.Column)
			}
			if !snakeCaseRegex.MatchString(*param.Name) {
				p.Errorf("invalid parameter name %s for method %s: must be snake_case at %s, line %d, column %d", *param.Name, m.Name, param.Position.Filename, param.Position.Line, param.Position.Column)
			}
		}

		if param.Stream && hasStreamingInput {
			p.Errorf("method %s can only have one stream param at %s, line %d, column %d", m.Name, param.Position.Filename, param.Position.Line, param.Position.Column)
		} else if param.Stream {
			hasStreamingInput = true
		}
	}

	hasStreamingOutput := false
	for _, r := range m.Returns {
		if r.Stream && hasStreamingOutput {
			p.Errorf("method %s can only have one stream return at %s, line %d, column %d", m.Name, r.Position.Filename, r.Position.Line, r.Position.Column)
		} else if r.Stream {
			hasStreamingOutput = true
		}
	}
}

func (p *validatorP1) validateEnum(e *ast.Enum) {
	fqn := e.FQN()
	if ex, ok := p.objects[fqn]; ok {
		p.nameClash(fqn, ex.Pos())
		return
	}
	p.objects[fqn] = e

	if len(e.Members) == 0 {
		p.Errorf("Enum %s must have at least one member at %s, line %d, column %d", e.Name, e.Position.Filename, e.Position.Line, e.Position.Column)
		return
	}

	p.detectDuplicatedEnumValues(e)
}

func (p *validatorP1) validateStruct(s *ast.Struct) {
	fqn := s.FQN()
	if ex, ok := p.objects[fqn]; ok {
		p.nameClash(fqn, ex.Pos())
		return
	}
	p.objects[fqn] = s
	p.detectDuplicatedFields(s)

	for _, ss := range s.Structs {
		p.validateStruct(ss)
	}

	for _, e := range s.Enums {
		p.validateEnum(e)
	}
}

func (p *validatorP1) detectDuplicatedFields(s *ast.Struct) {
	fields := make(posSet)
	for _, f := range s.Fields {
		if ex, ok := fields[f.Name]; ok {
			p.structFieldClash(f, ex)
			continue
		}
		fields[f.Name] = f.Pos()
	}
}

func (p *validatorP1) detectDuplicatedEnumValues(e *ast.Enum) {
	fields := make(posSet)
	for _, f := range e.Members {
		if ex, ok := fields[f.Name]; ok {
			p.nameClash(f.Name, ex)
			continue
		}
		fields[f.Name] = f.Pos()
	}
	return
}

func (p *validatorP1) defineImportAlias(imp *ast.Import) {
	if imp.Alias != "" {
		return
	}
	f, ok := p.files[imp.ResolvedValue]
	if !ok {
		panic("BUG: resolved import not found")
	}
	imp.Alias = f.Package.Components[len(f.Package.Components)-1]
}
