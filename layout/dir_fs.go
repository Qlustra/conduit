package layout

import (
	"os"
	"time"

	safety "github.com/qlustra/conduit/path"
)

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
	if err := safety.GuardPathMutation(d.Path(), ctx.pathSafetyPolicy(), safety.ExpectDir); err != nil {
		return nil, err
	}
	return os.Open(d.Path())
}

// OpenRoot opens the directory as an os.Root.
func (d Dir) OpenRoot(ctx Context) (*os.Root, error) {
	if err := safety.GuardPathMutation(d.Path(), ctx.pathSafetyPolicy(), safety.ExpectDir); err != nil {
		return nil, err
	}
	return os.OpenRoot(d.Path())
}

// Chmod changes the directory mode using os.Chmod.
func (d Dir) Chmod(mode os.FileMode, ctx Context) error {
	if err := safety.GuardPathMutation(d.Path(), ctx.pathSafetyPolicy(), safety.ExpectDir); err != nil {
		return err
	}
	return os.Chmod(d.Path(), mode)
}

// Chtimes changes the directory access and modification times using os.Chtimes.
func (d Dir) Chtimes(atime time.Time, mtime time.Time, ctx Context) error {
	if err := safety.GuardPathMutation(d.Path(), ctx.pathSafetyPolicy(), safety.ExpectDir); err != nil {
		return err
	}
	return os.Chtimes(d.Path(), atime, mtime)
}
