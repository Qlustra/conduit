package layout

import (
	"os"
	"time"
)

// Stat returns os.Stat for the file path.
func (f File) Stat() (os.FileInfo, error) {
	return os.Stat(f.Path())
}

// Lstat returns os.Lstat for the file path.
func (f File) Lstat() (os.FileInfo, error) {
	return os.Lstat(f.Path())
}

// Chmod changes the file mode using os.Chmod.
func (f File) Chmod(mode os.FileMode, ctx Context) error {
	if err := guardPathMutation(f.Path(), ctx.pathSafetyPolicy(), expectFile); err != nil {
		return err
	}
	return os.Chmod(f.Path(), mode)
}

// Chtimes changes the file access and modification times using os.Chtimes.
func (f File) Chtimes(atime time.Time, mtime time.Time, ctx Context) error {
	if err := guardPathMutation(f.Path(), ctx.pathSafetyPolicy(), expectFile); err != nil {
		return err
	}
	return os.Chtimes(f.Path(), atime, mtime)
}

// Stat returns os.Stat for the directory path.
func (d Dir) Stat() (os.FileInfo, error) {
	return os.Stat(d.Path())
}

// Lstat returns os.Lstat for the directory path.
func (d Dir) Lstat() (os.FileInfo, error) {
	return os.Lstat(d.Path())
}

// Open opens the directory path and returns its file descriptor.
func (d Dir) Open(ctx Context) (*os.File, error) {
	if err := guardPathMutation(d.Path(), ctx.pathSafetyPolicy(), expectDir); err != nil {
		return nil, err
	}
	return os.Open(d.Path())
}

// OpenRoot opens the directory as an os.Root.
func (d Dir) OpenRoot(ctx Context) (*os.Root, error) {
	if err := guardPathMutation(d.Path(), ctx.pathSafetyPolicy(), expectDir); err != nil {
		return nil, err
	}
	return os.OpenRoot(d.Path())
}

// Chmod changes the directory mode using os.Chmod.
func (d Dir) Chmod(mode os.FileMode, ctx Context) error {
	if err := guardPathMutation(d.Path(), ctx.pathSafetyPolicy(), expectDir); err != nil {
		return err
	}
	return os.Chmod(d.Path(), mode)
}

// Chtimes changes the directory access and modification times using os.Chtimes.
func (d Dir) Chtimes(atime time.Time, mtime time.Time, ctx Context) error {
	if err := guardPathMutation(d.Path(), ctx.pathSafetyPolicy(), expectDir); err != nil {
		return err
	}
	return os.Chtimes(d.Path(), atime, mtime)
}

// Lstat returns os.Lstat for the symlink path.
func (l Link) Lstat() (os.FileInfo, error) {
	return os.Lstat(l.Path())
}

// Readlink returns os.Readlink for the symlink path.
func (l Link) Readlink() (string, error) {
	return os.Readlink(l.Path())
}
