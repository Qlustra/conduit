package layout

import (
	"fmt"
	"path/filepath"
)

// CopyToPath copies the file payload onto the exact destination path.
func (f File) CopyToPath(path string, opts CopyOptions) error {
	dst := filepath.Clean(path)
	if samePath(f.Path(), dst) {
		return fmt.Errorf("source and destination must differ: %s", f.Path())
	}
	if err := GuardNodeKind(f.Path(), expectFile); err != nil {
		return err
	}

	return newCopier(opts).copyFile(f.Path(), dst)
}

// CopyToFile copies the file payload onto dst.Path().
func (f File) CopyToFile(dst File, opts CopyOptions) error {
	return f.CopyToPath(dst.Path(), opts)
}

// CopyIntoDir copies the file under dir using the source basename.
func (f File) CopyIntoDir(dir Dir, opts CopyOptions) error {
	return f.CopyToPath(dir.File(f.Base()).Path(), opts)
}
