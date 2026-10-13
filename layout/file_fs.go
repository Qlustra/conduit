package layout

import (
	"os"
	"time"

	safety "github.com/qlustra/conduit/path"
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
	if err := safety.GuardPathMutation(f.Path(), ctx.pathSafetyPolicy(), safety.ExpectFile); err != nil {
		return err
	}
	return os.Chmod(f.Path(), mode)
}

// Chtimes changes the file access and modification times using os.Chtimes.
func (f File) Chtimes(atime time.Time, mtime time.Time, ctx Context) error {
	if err := safety.GuardPathMutation(f.Path(), ctx.pathSafetyPolicy(), safety.ExpectFile); err != nil {
		return err
	}
	return os.Chtimes(f.Path(), atime, mtime)
}
