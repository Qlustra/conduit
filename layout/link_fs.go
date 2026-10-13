package layout

import "os"

// Lstat returns os.Lstat for the symlink path.
func (l Link) Lstat() (os.FileInfo, error) {
	return os.Lstat(l.Path())
}

// Readlink returns os.Readlink for the symlink path.
func (l Link) Readlink() (string, error) {
	return os.Readlink(l.Path())
}
