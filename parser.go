package idl

import (
	"bytes"
	"fmt"
	"github.com/arf-rpc/idl/ast"
	"os"
	"path/filepath"
)

type Parser struct {
	loadedFiles    map[string]bool
	set            ast.FileSet
	resolveImports bool
}

func (p *Parser) parsePath(path string) error {
	if p.loadedFiles[path] {
		return nil
	}

	basePath := filepath.Dir(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file %s: %w", path, err)
	}

	tokens, err := scan(path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("could not parse file %s: %w", path, err)
	}

	tree, err := parseTokens(path, tokens)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	p.loadedFiles[path] = true

	if p.resolveImports {
		for _, i := range tree.Imports {
			path, err := filepath.Rel(basePath, i.Path)
			if err != nil {
				return fmt.Errorf("could not resolve import path %q: %w", i.Path, err)
			}

			if err = p.parsePath(path); err != nil {
				return err
			}
		}
	}

	p.set = append(p.set, ast.File{
		Path: path,
		Tree: tree,
	})

	return nil
}

func (p *Parser) processTypes() error {
	for _, file := range p.set {
		if err := p.processTreeTypes(file.Tree); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) processTreeTypes(tree *ast.Tree) error {
	for _, s := range tree.Structures {
		if err := p.processStructureTypes(s); err != nil {
			return err
		}
	}

	for _, s := range tree.Services {
		if err := p.processServiceTypes(s); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) processStructureTypes(s *ast.Struct) error {
	for _, e := range s.Structs {
		if err := p.processStructureTypes(e); err != nil {
			return err
		}
	}

	for _, f := range s.Fields {
		if err := p.processFieldTypes(f); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) processServiceTypes(s *ast.Service) error {
	for _, m := range s.Methods {
		if m.Input != nil {
			if err := p.resolveType(m.Input, s); err != nil {
				return err
			}
		}

		if m.Output != nil {
			if err := p.resolveType(m.Output, s); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) processFieldTypes(f *ast.Field) error {
	if plain := f.Plain; plain != nil {
		return p.resolveType(plain.Type, f.Parent)
	} else {
		for _, v := range f.Union.Fields {
			if err := p.processFieldTypes(v); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) resolveType(t ast.Type, parent any) error {
	switch v := t.(type) {
	case *ast.StreamingType:
		return p.resolveType(v.Type, parent)
	case *ast.RepeatedType:
		return p.resolveType(v.Type, parent)
	case *ast.OptionalType:
		return p.resolveType(v.Type, parent)
	case *ast.MapType:
		if err := p.resolveType(v.Key, parent); err != nil {
			return err
		}
		if err := p.resolveType(v.Value, parent); err != nil {
			return err
		}
		return nil
	case ast.PrimitiveType:
		// Noop
		return nil
	case *ast.UserType:
		typ, err := p.findType(v.Name, parent)
		if err != nil {
			return err
		}
		v.ResolvedType = typ
		return nil
	default:
		panic(fmt.Sprintf("Unsupported type %T", t))
	}
}

func (p *Parser) findType(name string, parent any) (any, error) {
	switch v := parent.(type) {
	case *ast.Struct:
		for _, e := range v.Enums {
			if e.Name == name {
				return e, nil
			}
		}

		for _, s := range v.Structs {
			if s.Name == name {
				return s, nil
			}
		}

		if v.Parent == nil {
			return p.findRootType(name)
		}

		return p.findType(v.Name, v.Parent)

	case *ast.Service:
		return p.findRootType(name)

	default:
		panic(fmt.Sprintf("Unsupported type %T", v))
	}
}

func (p *Parser) findRootType(name string) (any, error) {
	for _, f := range p.set {
		for _, s := range f.Tree.Structures {
			if s.Name == name {
				return s, nil
			}
		}

		for _, e := range f.Tree.Enums {
			return e, nil
		}
	}

	return nil, fmt.Errorf("cannot resolve type %s", name)
}

func Parse(path string, resolveImports bool) (ast.FileSet, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	p := &Parser{
		loadedFiles:    map[string]bool{},
		resolveImports: resolveImports,
	}

	if err = p.parsePath(path); err != nil {
		return nil, err
	}

	if err = p.processTypes(); err != nil {
		return nil, err
	}

	if err = validate(p); err != nil {
		return nil, err
	}

	return p.set, nil
}
