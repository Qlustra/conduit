package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir is a stateless handle to a directory path.
//
// A Dir may be created directly or attached to a composed layout. When it is
// attached through Compose, it can also report declared-path and
// compose-base-relative metadata in addition to its filesystem path.
type Dir struct {
	path         string
	composeBase  string
	composedBase bool
	declaredPath string
	hasDeclared  bool
}

// NewDir returns a standalone directory handle for path.
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

// Path returns the bound filesystem path.
func (d Dir) Path() string {
	return d.path
}

// Base returns the final path element.
func (d Dir) Base() string {
	return filepath.Base(d.path)
}

// Stem returns the final path element without its final extension.
func (d Dir) Stem() string {
	stem, _ := splitBaseExt(d.Base())
	return stem
}

// ComposedBaseDir returns the root directory that anchored composition, when
// the handle belongs to a composed tree.
func (d Dir) ComposedBaseDir() (Dir, bool) {
	if !d.composedBase {
		return Dir{}, false
	}
	return newDirWithCompose(d.composeBase, d.composeBase, true), true
}

// DeclaredPath returns the node's own layout tag fragment when the handle was
// attached through Compose.
func (d Dir) DeclaredPath() (string, bool) {
	if !d.hasDeclared {
		return "", false
	}
	return d.declaredPath, true
}

// JoinDeclaredPath joins parts onto the node's declared layout fragment.
func (d Dir) JoinDeclaredPath(parts ...string) (string, bool) {
	declared, ok := d.DeclaredPath()
	if !ok {
		return "", false
	}
	return joinDeclaredPath(declared, parts...), true
}

// ComposedRelativePath returns the path relative to the tree's compose base.
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

// JoinComposedPath joins parts onto the compose-base-relative path.
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

// RelTo returns the path relative to base.
func (d Dir) RelTo(base Pather) (string, error) {
	if base == nil {
		return "", fmt.Errorf("base path must not be nil")
	}
	return filepath.Rel(base.Path(), d.Path())
}

// JoinRelTo joins parts onto the path relative to base.
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

// RelToPath returns the path relative to base.
func (d Dir) RelToPath(base string) (string, error) {
	return filepath.Rel(filepath.Clean(base), d.Path())
}

// JoinRelToPath joins parts onto the path relative to base.
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

// Exists reports whether a filesystem entry currently exists at Path.
func (d Dir) Exists() bool {
	_, err := os.Stat(d.Path())
	return err == nil
}

// Chown applies os.Chown to the directory path.
func (d Dir) Chown(uid int, gid int) error {
	return os.Chown(d.Path(), uid, gid)
}

// ChangeTo changes the process working directory to Path.
func (d Dir) ChangeTo() error {
	return os.Chdir(d.Path())
}

// Join returns a descendant path under the directory.
func (d Dir) Join(parts ...string) string {
	all := append([]string{d.path}, parts...)
	return filepath.Join(all...)
}

// List returns the direct children of the directory.
func (d Dir) List() ([]os.DirEntry, error) {
	return os.ReadDir(d.Path())
}

// Dir returns a child directory handle under the receiver.
func (d Dir) Dir(name string) Dir {
	return newDirWithCompose(filepath.Join(d.path, name), d.composeBase, d.composedBase)
}

// File returns a child file handle under the receiver.
func (d Dir) File(name string) File {
	return newFileWithCompose(filepath.Join(d.path, name), d.composeBase, d.composedBase)
}

// DeleteIfExists removes the directory tree when it exists.
func (d Dir) DeleteIfExists() error {
	_, err := os.Stat(d.Path())
	if err != nil {
		return nil
	}
	return os.RemoveAll(d.Path())
}

// Compose

// ComposePath binds the handle to path and resets composition metadata.
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

// Ensure creates the directory tree using ctx.DirMode.
func (d Dir) Ensure(ctx Context) error {
	if !ctx.ensurePolicy().allowsDir() {
		return nil
	}
	return os.MkdirAll(d.path, ctx.DirMode)
}

// Copy

// CopyToPath copies the directory tree onto the exact destination path.
//
// The destination must differ from the source and must not be inside the
// source tree.
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

// CopyToDir copies the directory tree onto dst.Path().
func (d Dir) CopyToDir(dst Dir, opts CopyOptions) error {
	return d.CopyToPath(dst.Path(), opts)
}

// CopyIntoDir copies the directory tree under parent using the source
// basename.
func (d Dir) CopyIntoDir(parent Dir, opts CopyOptions) error {
	return d.CopyToPath(parent.Dir(d.Base()).Path(), opts)
}
