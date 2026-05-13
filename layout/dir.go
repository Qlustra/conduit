package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

type Dir struct {
	path         string
	composeBase  string
	composedBase bool
	declaredPath string
	hasDeclared  bool
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

func (d Dir) DeclaredPath() (string, bool) {
	if !d.hasDeclared {
		return "", false
	}
	return d.declaredPath, true
}

func (d Dir) JoinDeclaredPath(parts ...string) (string, bool) {
	declared, ok := d.DeclaredPath()
	if !ok {
		return "", false
	}
	return joinDeclaredPath(declared, parts...), true
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

func (d Dir) RelTo(base Pather) (string, error) {
	if base == nil {
		return "", fmt.Errorf("base path must not be nil")
	}
	return filepath.Rel(base.Path(), d.Path())
}

func (d Dir) JoinRelTo(base Pather, parts ...string) (string, error) {
	rel, err := d.RelTo(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

func (d Dir) RelToPath(base string) (string, error) {
	return filepath.Rel(filepath.Clean(base), d.Path())
}

func (d Dir) JoinRelToPath(base string, parts ...string) (string, error) {
	rel, err := d.RelToPath(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

func (d Dir) Exists() bool {
	_, err := os.Stat(d.Path())
	return err == nil
}

func (d Dir) Chown(uid int, gid int) error {
	return os.Chown(d.Path(), uid, gid)
}

func (d Dir) ChangeTo() error {
	return os.Chdir(d.Path())
}

func (d Dir) Join(parts ...string) string {
	all := append([]string{d.path}, parts...)
	return filepath.Join(all...)
}

func (d Dir) List() ([]os.DirEntry, error) {
	return os.ReadDir(d.Path())
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

func (d *Dir) setDeclaredPath(path string) {
	d.declaredPath = path
	d.hasDeclared = true
}

// Ensure

func (d Dir) Ensure(ctx Context) error {
	return os.MkdirAll(d.path, ctx.DirMode)
}

// Copy

func (d Dir) CopyToPath(path string, opts CopyOptions) error {
	dst := filepath.Clean(path)
	if samePath(d.Path(), dst) {
		return fmt.Errorf("source and destination must differ: %s", d.Path())
	}
	if pathWithin(dst, d.Path()) {
		return fmt.Errorf("destination path %s must not be inside source directory %s", dst, d.Path())
	}

	return newCopier(opts).copyDir(d.Path(), dst)
}

func (d Dir) CopyToDir(dst Dir, opts CopyOptions) error {
	return d.CopyToPath(dst.Path(), opts)
}

func (d Dir) CopyIntoDir(parent Dir, opts CopyOptions) error {
	return d.CopyToPath(parent.Dir(d.Base()).Path(), opts)
}
