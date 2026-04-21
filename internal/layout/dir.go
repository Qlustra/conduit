package layout

import (
	"os"
	"path/filepath"
)

type Dir struct {
	path string
}

func NewDir(path string) Dir {
	return Dir{path: filepath.Clean(path)}
}

func (d Dir) Path() string {
	return d.path
}

func (d Dir) Exists() bool {
	_, err := os.Stat(d.Path())
	return err == nil
}

func (d Dir) Join(parts ...string) string {
	all := append([]string{d.path}, parts...)
	return filepath.Join(all...)
}

func (d Dir) Dir(name string) Dir {
	return NewDir(filepath.Join(d.path, name))
}

func (d Dir) File(name string) File {
	return NewFile(filepath.Join(d.path, name))
}

func (d Dir) DeleteIfExists() error {
	_, err := os.Stat(d.Path())
	if err != nil {
		return nil
	}
	return os.RemoveAll(d.Path())
}

// Compose

func (d *Dir) ComposePath(path string) {
	d.path = filepath.Clean(path)
}

// Ensure

func (d Dir) Ensure(ctx Context) error {
	return os.MkdirAll(d.path, ctx.DirMode)
}
