package idl

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/arf-rpc/idl/ast"
)

func Parse(entrypoint string) (*ast.Tree, error) {
	fe, err := New(entrypoint)
	if err != nil {
		return nil, err
	}
	return fe.Run()
}

type Frontend interface {
	Run() (*ast.Tree, error)
}

type frontend struct {
	entrypoint     string
	workingDir     string
	processedPaths map[string]struct{}
	files          map[string]*ast.File
}

func New(entrypoint string) (Frontend, error) {
	stat, err := os.Stat(entrypoint)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("%s: is a directory", entrypoint)
	}
	absPath, err := filepath.Abs(entrypoint)
	if err != nil {
		return nil, err
	}

	return &frontend{
		entrypoint:     absPath,
		workingDir:     path.Dir(absPath),
		processedPaths: map[string]struct{}{},
		files:          map[string]*ast.File{},
	}, nil
}

func (f *frontend) Run() (*ast.Tree, error) {
	if err := f.parse(f.entrypoint); err != nil {
		return nil, err
	}
	if err := validatePhase1(f.files, f.entrypoint); err != nil {
		return nil, err
	}
	if err := validatePhase2(f.files, f.entrypoint); err != nil {
		return nil, err
	}
	if err := validatePhase3(f.files, f.entrypoint); err != nil {
		return nil, err
	}

	tree := &ast.Tree{}
	for _, f := range f.files {
		tree.AddFile(f)
	}

	return tree, nil
}

func (f *frontend) parse(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tokens, errs := lexFile(data, nil)
	if errs != nil {
		return errors.Join(errs...)
	}

	astFile, errs := parse(path, tokens, nil)
	if errs != nil {
		return errors.Join(errs...)
	}

	for i, imp := range astFile.Imports {
		val := imp.Value
		if !strings.HasSuffix(strings.ToLower(val), ".arf") {
			val = val + ".arf"
		}

		clean, err := filepath.Abs(filepath.Join(filepath.Dir(path), val))
		if err != nil {
			return err
		}

		if _, ok := f.processedPaths[clean]; !ok {
			if err = f.parse(clean); err != nil {
				return err
			}
		}
		astFile.Imports[i].ResolvedValue = clean
	}

	f.files[path] = astFile
	f.processedPaths[path] = struct{}{}

	return nil
}
