package layout

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopySymlinkPolicy controls how copy helpers handle symlinks encountered in
// the source tree.
type CopySymlinkPolicy uint8

const (
	// CopySymlinkPreserve recreates symlinks as symlinks using the raw source
	// target string.
	CopySymlinkPreserve CopySymlinkPolicy = iota

	// CopySymlinkFollow copies the payload reached through the symlink instead of
	// preserving the symlink entry.
	CopySymlinkFollow

	// CopySymlinkReject fails when a symlink is encountered.
	CopySymlinkReject
)

// CopyOverwritePolicy controls what copy helpers do when the destination
// already exists.
type CopyOverwritePolicy uint8

const (
	// CopyOverwriteFail returns an error when the destination already exists.
	CopyOverwriteFail CopyOverwritePolicy = iota

	// CopyOverwriteReplace removes the existing destination before copying.
	CopyOverwriteReplace
)

// CopyOptions configures File and Dir copy helpers.
//
// The zero value is treated as DefaultCopyOptions.
type CopyOptions struct {
	// Overwrite controls whether an existing destination is rejected or
	// replaced.
	Overwrite CopyOverwritePolicy

	// Symlinks controls how symlinks in the source tree are handled.
	Symlinks CopySymlinkPolicy

	// PreserveMode controls whether source modes are reused on created files and
	// directories.
	PreserveMode bool

	// FileMode is the fallback mode for created files when PreserveMode is
	// false.
	FileMode os.FileMode

	// DirMode is the fallback mode for created directories when PreserveMode is
	// false.
	DirMode os.FileMode
}

// DefaultCopyOptions preserves source modes and symlinks and fails if the
// destination already exists.
var DefaultCopyOptions = CopyOptions{
	Overwrite:    CopyOverwriteFail,
	Symlinks:     CopySymlinkPreserve,
	PreserveMode: true,
}

type copier struct {
	opts           CopyOptions
	activeRealDirs map[string]struct{}
}

func newCopier(opts CopyOptions) copier {
	return copier{
		opts:           opts.normalized(),
		activeRealDirs: make(map[string]struct{}),
	}
}

func (c copier) copyFile(src string, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("source path %s is a directory", src)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return c.copySymlink(src, dst, fileKind)
	}

	return c.copyRegularFile(src, dst, info)
}

func (c copier) copyDir(src string, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return c.copySymlink(src, dst, dirKind)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path %s is not a directory", src)
	}

	return c.copyDirectoryTree(src, dst, info)
}

type sourceKind uint8

const (
	fileKind sourceKind = iota + 1
	dirKind
)

func (c copier) copySymlink(src string, dst string, kind sourceKind) error {
	switch c.opts.Symlinks {
	case CopySymlinkPreserve:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := c.prepareDestination(dst); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), c.opts.dirMode()); err != nil {
			return err
		}
		return os.Symlink(target, dst)

	case CopySymlinkFollow:
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if kind == fileKind {
			if info.IsDir() {
				return fmt.Errorf("source path %s resolves to a directory", src)
			}
			return c.copyRegularFile(src, dst, info)
		}
		if !info.IsDir() {
			return fmt.Errorf("source path %s does not resolve to a directory", src)
		}
		return c.copyDirectoryTree(src, dst, info)

	case CopySymlinkReject:
		return fmt.Errorf("source path %s contains a symlink", src)

	default:
		return fmt.Errorf("unsupported symlink policy %d", c.opts.Symlinks)
	}
}

func (c copier) copyRegularFile(src string, dst string, info os.FileInfo) error {
	if err := c.prepareDestination(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), c.opts.dirMode()); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	mode := c.opts.fileMode(info.Mode())
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	if err := os.Chmod(dst, mode); err != nil {
		return err
	}
	return nil
}

func (c copier) copyDirectoryTree(src string, dst string, info os.FileInfo) error {
	realPath, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	if _, exists := c.activeRealDirs[realPath]; exists {
		return fmt.Errorf("symlink cycle detected at %s", src)
	}
	c.activeRealDirs[realPath] = struct{}{}
	defer delete(c.activeRealDirs, realPath)

	if err := c.prepareDestination(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(dst, c.opts.dirMode(info.Mode())); err != nil {
		return err
	}
	if err := os.Chmod(dst, c.opts.dirMode(info.Mode())); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childSrc := filepath.Join(src, entry.Name())
		childDst := filepath.Join(dst, entry.Name())

		childInfo, err := os.Lstat(childSrc)
		if err != nil {
			return err
		}

		switch {
		case childInfo.Mode()&os.ModeSymlink != 0:
			childKind := fileKind
			if c.opts.Symlinks == CopySymlinkFollow {
				targetInfo, err := os.Stat(childSrc)
				if err != nil {
					return err
				}
				if targetInfo.IsDir() {
					childKind = dirKind
				}
			}
			if err := c.copySymlink(childSrc, childDst, childKind); err != nil {
				return err
			}

		case childInfo.IsDir():
			if err := c.copyDirectoryTree(childSrc, childDst, childInfo); err != nil {
				return err
			}

		default:
			if err := c.copyRegularFile(childSrc, childDst, childInfo); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c copier) prepareDestination(dst string) error {
	info, err := os.Lstat(dst)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	switch c.opts.Overwrite {
	case CopyOverwriteFail:
		return fmt.Errorf("destination path %s already exists", dst)
	case CopyOverwriteReplace:
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return os.RemoveAll(dst)
		}
		return os.Remove(dst)
	default:
		return fmt.Errorf("unsupported overwrite policy %d", c.opts.Overwrite)
	}
}

func (opts CopyOptions) normalized() CopyOptions {
	if opts == (CopyOptions{}) {
		return DefaultCopyOptions
	}
	if !opts.PreserveMode {
		if opts.FileMode == 0 {
			opts.FileMode = DefaultContext.FileMode
		}
		if opts.DirMode == 0 {
			opts.DirMode = DefaultContext.DirMode
		}
	}
	return opts
}

func (opts CopyOptions) fileMode(src os.FileMode) os.FileMode {
	if opts.PreserveMode {
		return src & os.ModePerm
	}
	return opts.FileMode
}

func (opts CopyOptions) dirMode(src ...os.FileMode) os.FileMode {
	if opts.PreserveMode {
		if len(src) != 0 {
			return src[0] & os.ModePerm
		}
		return DefaultContext.DirMode
	}
	return opts.DirMode
}

func samePath(a string, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func pathWithin(path string, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)

	if path == root {
		return true
	}

	sep := string(os.PathSeparator)
	if !strings.HasSuffix(root, sep) {
		root += sep
	}
	return strings.HasPrefix(path, root)
}
