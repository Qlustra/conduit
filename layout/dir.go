package layout

import (
	"os"
	"path/filepath"
)

type Dir struct {
	path         string
	composeBase  string
	composedBase bool
}

func NewDir(path string) Dir {
	return newDirWithCompose(path, "", false)
}

func newDirWithCompose(path string, composeBase string, composed bool) Dir {
	dir := Dir{path: filepath.Clean(path)}
	if composed {
		dir.composeBase = filepath.Clean(composeBase)
		dir.composedBase = true
	}
	return dir
}

func (d Dir) Path() string {
	return d.path
}

func (d Dir) Base() string {
	return filepath.Base(d.path)
}

func (d Dir) Stem() string {
	stem, _ := splitBaseExt(d.Base())
	return stem
}

func (d Dir) ComposedBaseDir() (Dir, bool) {
	if !d.composedBase {
		return Dir{}, false
	}
	return newDirWithCompose(d.composeBase, d.composeBase, true), true
}

func (d Dir) ComposedRelativePath() (string, bool) {
	if !d.composedBase {
		return "", false
	}
	rel, err := filepath.Rel(d.composeBase, d.path)
	if err != nil {
		return "", false
	}
	return rel, true
}

func (d Dir) JoinComposedPath(parts ...string) (string, bool) {
	rel, ok := d.ComposedRelativePath()
	if !ok {
		return "", false
	}
	if len(parts) == 0 {
		return rel, true
	}
	return filepath.Join(append([]string{rel}, parts...)...), true
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
	return newDirWithCompose(filepath.Join(d.path, name), d.composeBase, d.composedBase)
}

func (d Dir) File(name string) File {
	return newFileWithCompose(filepath.Join(d.path, name), d.composeBase, d.composedBase)
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
	d.composeBase = ""
	d.composedBase = false
}

func (d *Dir) setComposeBase(path string) {
	d.composeBase = filepath.Clean(path)
	d.composedBase = true
}

// Ensure

func (d Dir) Ensure(ctx Context) error {
	return os.MkdirAll(d.path, ctx.DirMode)
}
