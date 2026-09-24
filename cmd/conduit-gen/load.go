package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type listedPackage struct {
	Dir        string
	ImportPath string
	Name       string
	GoFiles    []string
	CgoFiles   []string
	Export     string
}

// go list supplies build-tag-aware source selection and compiler export data.
// Checking our source separately avoids requiring a previously generated file.
// Unrelated unresolved references to that future file are tolerated; every
// declaration and field type actually used by the generator is checked below.
func loadPackage(dir, output string) (*types.Package, error) {
	cmd := exec.Command("go", "list", "-e", "-deps", "-export", "-json", ".")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("load package: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	exports := map[string]string{}
	var selected listedPackage
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var listed listedPackage
		if err := decoder.Decode(&listed); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode go list: %w", err)
		}
		if listed.Export != "" {
			exports[listed.ImportPath] = listed.Export
		}
		if filepath.Clean(listed.Dir) == filepath.Clean(dir) {
			selected = listed
		}
	}
	if selected.Name == "" {
		return nil, fmt.Errorf("no Go package found in %s", dir)
	}
	if len(selected.CgoFiles) != 0 {
		return nil, fmt.Errorf("cgo declaration packages are not supported")
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range selected.GoFiles {
		if name == output {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.AllErrors)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	lookup := func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no compiler export data for %s; ensure this dependency builds", path)
		}
		return os.Open(file)
	}
	var typeErrors []error
	conf := types.Config{
		Importer: importer.ForCompiler(fset, "gc", lookup),
		Error:    func(err error) { typeErrors = append(typeErrors, err) },
	}
	pkg, err := conf.Check(selected.ImportPath, fset, files, nil)
	if pkg == nil {
		return nil, fmt.Errorf("check package: %w", err)
	}
	// Import failures must never be mistaken for unsupported topology fields.
	for _, err := range typeErrors {
		if strings.Contains(err.Error(), "could not import") {
			return nil, err
		}
	}
	return pkg, nil
}
