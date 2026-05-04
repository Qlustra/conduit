package layout

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	path string
}

func NewFile(path string) File {
	return File{path: filepath.Clean(path)}
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
