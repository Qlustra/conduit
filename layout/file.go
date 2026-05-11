package layout

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	path         string
	composeBase  string
	composedBase bool
	declaredPath string
	hasDeclared  bool
}

func NewFile(path string) File {
	return newFileWithCompose(path, "", false)
}

func newFileWithCompose(path string, composeBase string, composed bool) File {
	file := File{path: filepath.Clean(path)}
	if composed {
		file.composeBase = filepath.Clean(composeBase)
		file.composedBase = true
	}
	return file
}

func (f File) Path() string {
	return f.path
}

func (f File) Base() string {
	return filepath.Base(f.path)
}

func (f File) Ext() string {
	_, ext := splitBaseExt(f.Base())
	return ext
}

func (f File) Stem() string {
	stem, _ := splitBaseExt(f.Base())
	return stem
}

func (f File) ComposedBaseDir() (Dir, bool) {
	if !f.composedBase {
		return Dir{}, false
	}
	return newDirWithCompose(f.composeBase, f.composeBase, true), true
}

func (f File) DeclaredPath() (string, bool) {
	if !f.hasDeclared {
		return "", false
	}
	return f.declaredPath, true
}

func (f File) JoinDeclaredPath(parts ...string) (string, bool) {
	declared, ok := f.DeclaredPath()
	if !ok {
		return "", false
	}
	return joinDeclaredPath(declared, parts...), true
}

func (f File) ComposedRelativePath() (string, bool) {
	if !f.composedBase {
		return "", false
	}
	rel, err := filepath.Rel(f.composeBase, f.path)
	if err != nil {
		return "", false
	}
	return rel, true
}

func (f File) JoinComposedPath(parts ...string) (string, bool) {
	rel, ok := f.ComposedRelativePath()
	if !ok {
		return "", false
	}
	if len(parts) == 0 {
		return rel, true
	}
	return filepath.Join(append([]string{rel}, parts...)...), true
}

func (f File) RelTo(base Pather) (string, error) {
	if base == nil {
		return "", fmt.Errorf("base path must not be nil")
	}
	return filepath.Rel(base.Path(), f.Path())
}

func (f File) JoinRelTo(base Pather, parts ...string) (string, error) {
	rel, err := f.RelTo(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

func (f File) RelToPath(base string) (string, error) {
	return filepath.Rel(filepath.Clean(base), f.Path())
}

func (f File) JoinRelToPath(base string, parts ...string) (string, error) {
	rel, err := f.RelToPath(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

func (f File) Exists() bool {
	_, err := os.Stat(f.Path())
	return err == nil
}

func (f File) WriteBytes(data []byte, dirMode os.FileMode, fileMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(f.path), dirMode); err != nil {
		return err
	}
	return os.WriteFile(f.path, data, fileMode)
}

func (f File) ReadBytes() ([]byte, error) {
	return os.ReadFile(f.path)
}

func (f File) DeleteIfExists() error {
	_, err := os.Stat(f.Path())
	if err != nil {
		return nil
	}
	return os.Remove(f.Path())
}

func (f File) ReadBytesIfExists() ([]byte, bool, error) {
	data, err := f.ReadBytes()
	if err == nil {
		return data, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, err
}

// Compose

func (f *File) ComposePath(path string) {
	f.path = filepath.Clean(path)
	f.composeBase = ""
	f.composedBase = false
}

func (f *File) setComposeBase(path string) {
	f.composeBase = filepath.Clean(path)
	f.composedBase = true
}

func (f *File) setDeclaredPath(path string) {
	f.declaredPath = path
	f.hasDeclared = true
}

// Ensure

func (f File) Ensure(ctx Context) error {
	if err := os.MkdirAll(filepath.Dir(f.path), ctx.DirMode); err != nil {
		return err
	}

	handle, err := os.OpenFile(f.path, os.O_CREATE, ctx.FileMode)
	if err != nil {
		return err
	}

	return handle.Close()
}

func splitBaseExt(base string) (stem string, ext string) {
	switch base {
	case "", ".", "..":
		return base, ""
	}

	if strings.HasPrefix(base, ".") && strings.Count(base, ".") == 1 {
		return base, ""
	}

	ext = filepath.Ext(base)
	return base[:len(base)-len(ext)], ext
}

func joinDeclaredPath(base string, parts ...string) string {
	if len(parts) == 0 {
		return base
	}
	if base == "." || base == "" {
		return filepath.Join(parts...)
	}
	return filepath.Join(append([]string{base}, parts...)...)
}
