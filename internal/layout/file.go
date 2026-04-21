package layout

import (
	"errors"
	"os"
	"path/filepath"
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

func (f File) Ensure(dirMode os.FileMode, fileMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(f.path), dirMode); err != nil {
		return err
	}

	handle, err := os.OpenFile(f.path, os.O_CREATE, fileMode)
	if err != nil {
		return err
	}

	return handle.Close()
}
