package idl

import (
	"errors"
	"fmt"
	"github.com/arf-rpc/idl/ast"
	"os"
	"path/filepath"
	"strings"
)

type frontEnd struct {
	imported   map[string]bool
	files      []*ast.File
	onError    func(error)
	entrypoint string
	validator  *validator
	tree       ast.Tree
}

func (f *frontEnd) run() (*ast.Tree, error) {
	var err error
	if !filepath.IsAbs(f.entrypoint) {
		f.entrypoint, err = filepath.Abs(f.entrypoint)
		if err != nil {
			return nil, err
		}
	}

	err = f.parsePath(f.entrypoint)
	if err != nil {
		return nil, err
	}

	f.validator.runDeferredChecks()
	for _, file := range f.files {
		f.validator.runLoopChecks(file)
	}

	if len(f.validator.errs) > 0 {
		return nil, errors.Join(f.validator.errs...)
	}

	for _, file := range f.files {
		f.tree.AddFile(file)
	}
	return &f.tree, nil
}

func (f *frontEnd) didImport(path string) {
	f.imported[path] = true
}

func (f *frontEnd) shouldImport(path string) bool {
	return !f.imported[path]
}

func (f *frontEnd) parsePath(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tokens, errs := lexFile(data, f.onError)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	file, errs := parse(path, tokens, f.onError)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	f.validator.validateFile(file)
	f.files = append(f.files, file)
	f.didImport(path)
	return errors.Join(f.parseImports(file)...)
}

func (f *frontEnd) parseImports(file *ast.File) []error {
	if len(file.Imports) == 0 {
		return nil
	}
	var errs []string
	for i, imp := range file.Imports {
		p := imp.Value
		if !filepath.IsAbs(p) {
			p = filepath.Join(filepath.Dir(file.Path), p)
		}

		p, err := filepath.Abs(p)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Cannot get absolute path for %s: %s; at %s, line %d, column %d", imp.Value, err.Error(), imp.Position.Filename, imp.Position.Line, imp.Position.Column))
			continue
		}

		if !strings.HasSuffix(p, ".arf") {
			p = p + ".arf"
		}

		stat, err := os.Stat(p)
		if os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("Cannot import %s: %s does not exist; at %s, line %d, column %d", imp.Value, p, imp.Position.Filename, imp.Position.Line, imp.Position.Column))
			continue
		} else if err != nil {
			errs = append(errs, fmt.Sprintf("Cannot stat %s (%s): %s; at %s, line %d, column %d", imp.Value, p, err.Error(), imp.Position.Filename, imp.Position.Line, imp.Position.Column))
			continue
		}
		if stat.IsDir() {
			errs = append(errs, fmt.Sprintf("Cannot import %s: is a directory; at %s, line %d, column %d", imp.Value, imp.Position.Filename, imp.Position.Line, imp.Position.Column))
			continue
		}

		err = f.parsePath(p)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Cannot import %s: %s; at %s, line %d, column %d", imp.Value, err.Error(), imp.Position.Filename, imp.Position.Line, imp.Position.Column))
			continue
		}
		file.Imports[i].ResolvedValue = p
	}

	ret := make([]error, len(errs))
	for i, err := range errs {
		ret[i] = errors.New(err)
	}
	return ret
}

func ParseFile(path string, onError func(error)) (*ast.Tree, error) {
	f := &frontEnd{
		onError:    onError,
		entrypoint: path,
		imported:   make(map[string]bool),
		validator:  newValidator(),
	}
	return f.run()
}
