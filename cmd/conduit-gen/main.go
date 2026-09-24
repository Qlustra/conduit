// Command conduit-gen generates typed management wrappers from spec declarations.
//
// Run in the package containing the declaration:
//
//	go run github.com/qlustra/conduit/cmd/conduit-gen -type NotesManagement -output notes_gen.go
//
// Generation reads declarations and type information, never a runtime layout or
// the managed filesystem. Generated bindings and operations delegate to mgmt.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var typeName, output string
	flag.StringVar(&typeName, "type", "", "topology declaration type (required)")
	flag.StringVar(&output, "output", "", "generated Go file (default: <type>_gen.go)")
	flag.Parse()
	if typeName == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if output == "" {
		output = typeName + "_gen.go"
	}
	dir, err := os.Getwd()
	if err == nil {
		err = generateFile(dir, typeName, output)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "conduit-gen:", err)
		os.Exit(1)
	}
}

func generateFile(dir, typeName, output string) error {
	path := output
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	if filepath.Dir(filepath.Clean(path)) != filepath.Clean(dir) || filepath.Ext(path) != ".go" {
		return fmt.Errorf("output must be a .go file in the declaration package")
	}
	pkg, err := loadPackage(dir, filepath.Base(path))
	if err != nil {
		return err
	}
	data, err := generate(pkg, typeName)
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil && !isGenerated(old) {
		return fmt.Errorf("refusing to overwrite non-generated file %s", path)
	}
	return os.WriteFile(path, data, 0o644)
}
