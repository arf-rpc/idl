package idl

import (
	"errors"
	"fmt"

	"github.com/arf-rpc/idl/ast"
)

func validatePhase3(files map[string]*ast.File, entrypoint string) error {
	f, ok := files[entrypoint]
	if !ok {
		return fmt.Errorf("BUG: validation entrypoint %s not found", entrypoint)
	}

	v := &validatorP3{}

	for _, s := range f.Services {
		v.detectDuplicatedMethods(s)
	}

	return errors.Join(v.errors...)
}

type validatorP3 struct {
	errors []error
}

func (p *validatorP3) Errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Errorf(format, args...))
}

func (p *validatorP3) detectDuplicatedMethods(s *ast.Service) {
	methods := make(map[string]*ast.ServiceMethod)
	for _, m := range s.Methods {
		if ex, ok := methods[m.Name]; ok {
			if p.areMethodsDivergent(m, ex) {
				p.methodNameClash(m, ex.Pos())
			}
			continue
		}
		methods[m.Name] = m
	}
	return
}

func (p *validatorP3) areMethodsDivergent(m *ast.ServiceMethod, ex *ast.ServiceMethod) bool {
	if len(m.Params) != len(ex.Params) || len(m.Returns) != len(ex.Returns) {
		return true
	}

	for i, va := range m.Params {
		vb := ex.Params[i]
		if !va.Eql(vb) {
			return true
		}
	}

	for i, va := range m.Returns {
		vb := ex.Returns[i]
		if !va.Eql(vb) {
			return true
		}
	}

	return false
}

func (p *validatorP3) methodNameClash(m *ast.ServiceMethod, ex *ast.Position) {
	p.Errorf("%s is already defined for %s at %s, line %d, column %d", m.Name, m.Service.Name, ex.File.Path, ex.Line, ex.Column)
}
