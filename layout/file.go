package layout

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// File is a stateless handle to a regular file path.
//
// A File may be created directly or attached to a composed layout. When it is
// attached through Compose, it can also report declared-path and
// compose-base-relative metadata in addition to its filesystem path.
type File struct {
	path         string
	composeBase  string
	composedBase bool
	declaredPath string
	hasDeclared  bool
}

// NewFile returns a standalone file handle for path.
func NewFile(path string) File {
	return newFileWithCompose(path, "", false)
}

func newFileWithCompose(path string, composeBase string, composed bool) File {
	file := File{path: filepath.Clean(path)}
	if composed {
		file.composeBase = filepath.Clean(composeBase)
		file.composedBase = true
	}
	return file
}

// Path returns the bound filesystem path.
func (f File) Path() string {
	return f.path
}

// Base returns the final path element.
func (f File) Base() string {
	return filepath.Base(f.path)
}

// Ext returns the final extension including the leading dot.
func (f File) Ext() string {
	_, ext := splitBaseExt(f.Base())
	return ext
}

// Stem returns the final path element without its final extension.
func (f File) Stem() string {
	stem, _ := splitBaseExt(f.Base())
	return stem
}

// ComposedBaseDir returns the root directory that anchored composition, when
// the handle belongs to a composed tree.
func (f File) ComposedBaseDir() (Dir, bool) {
	if !f.composedBase {
		return Dir{}, false
	}
	return newDirWithCompose(f.composeBase, f.composeBase, true), true
}

// DeclaredPath returns the node's own layout tag fragment when the handle was
// attached through Compose.
func (f File) DeclaredPath() (string, bool) {
	if !f.hasDeclared {
		return "", false
	}
	return f.declaredPath, true
}

// JoinDeclaredPath joins parts onto the node's declared layout fragment.
func (f File) JoinDeclaredPath(parts ...string) (string, bool) {
	declared, ok := f.DeclaredPath()
	if !ok {
		return "", false
	}
	return joinDeclaredPath(declared, parts...), true
}

// ComposedRelativePath returns the path relative to the tree's compose base.
func (f File) ComposedRelativePath() (string, bool) {
	if !f.composedBase {
		return "", false
	}
	rel, err := filepath.Rel(f.composeBase, f.path)
	if err != nil {
		return "", false
	}
	return rel, true
}

// JoinComposedPath joins parts onto the compose-base-relative path.
func (f File) JoinComposedPath(parts ...string) (string, bool) {
	rel, ok := f.ComposedRelativePath()
	if !ok {
		return "", false
	}
	if len(parts) == 0 {
		return rel, true
	}
	return filepath.Join(append([]string{rel}, parts...)...), true
}

// RelTo returns the path relative to base.
func (f File) RelTo(base Pather) (string, error) {
	if base == nil {
		return "", fmt.Errorf("base path must not be nil")
	}
	return filepath.Rel(base.Path(), f.Path())
}

// JoinRelTo joins parts onto the path relative to base.
func (f File) JoinRelTo(base Pather, parts ...string) (string, error) {
	rel, err := f.RelTo(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

// RelToPath returns the path relative to base.
func (f File) RelToPath(base string) (string, error) {
	return filepath.Rel(filepath.Clean(base), f.Path())
}

// JoinRelToPath joins parts onto the path relative to base.
func (f File) JoinRelToPath(base string, parts ...string) (string, error) {
	rel, err := f.RelToPath(base)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return rel, nil
	}
	return filepath.Join(append([]string{rel}, parts...)...), nil
}

// Exists reports whether a filesystem entry currently exists at Path.
func (f File) Exists() bool {
	_, err := os.Stat(f.Path())
	return err == nil
}

// Chown applies os.Chown to the file path.
func (f File) Chown(uid int, gid int) error {
	return os.Chown(f.Path(), uid, gid)
}

// IsExecutable reports whether Path currently points to an executable regular
// file.
func (f File) IsExecutable() bool {
	info, err := os.Stat(f.Path())
	if err != nil {
		return false
	}
	return info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

// Truncate resizes the file in place using os.Truncate.
func (f File) Truncate(size int64) error {
	return os.Truncate(f.Path(), size)
}

// AppendReader creates parent directories if needed and appends bytes read
// from src.
func (f File) AppendReader(src io.Reader, dirMode os.FileMode, fileMode os.FileMode) error {
	if src == nil {
		return fmt.Errorf("append source must not be nil")
	}

	out, err := f.openAppendDestination(dirMode, fileMode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// AppendBytes creates parent directories if needed and appends raw bytes.
func (f File) AppendBytes(data []byte, dirMode os.FileMode, fileMode os.FileMode) error {
	return f.AppendReader(bytes.NewReader(data), dirMode, fileMode)
}

// AppendString creates parent directories if needed and appends string
// content.
func (f File) AppendString(content string, dirMode os.FileMode, fileMode os.FileMode) error {
	return f.AppendReader(strings.NewReader(content), dirMode, fileMode)
}

// AppendFile creates parent directories if needed and appends the source file
// payload.
func (f File) AppendFile(src File, dirMode os.FileMode, fileMode os.FileMode) (err error) {
	if samePath(f.Path(), src.Path()) {
		return fmt.Errorf("source and destination must differ: %s", f.Path())
	}

	in, err := os.Open(src.Path())
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := in.Close(); err == nil {
			err = closeErr
		}
	}()

	err = f.AppendReader(in, dirMode, fileMode)
	return err
}

// AppendFiles creates parent directories if needed and appends each source
// file payload in order.
func (f File) AppendFiles(dirMode os.FileMode, fileMode os.FileMode, srcs ...File) error {
	for _, src := range srcs {
		if err := f.AppendFile(src, dirMode, fileMode); err != nil {
			return err
		}
	}
	return nil
}

// WriteBytes creates parent directories if needed and rewrites the file
// contents.
func (f File) WriteBytes(data []byte, dirMode os.FileMode, fileMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(f.path), dirMode); err != nil {
		return err
	}
	return os.WriteFile(f.path, data, fileMode)
}

// ReadBytes reads the file contents.
func (f File) ReadBytes() ([]byte, error) {
	return os.ReadFile(f.path)
}

// DeleteIfExists removes the file when it exists.
func (f File) DeleteIfExists() error {
	_, err := os.Stat(f.Path())
	if err != nil {
		return nil
	}
	return os.Remove(f.Path())
}

// ReadBytesIfExists reads the file when present and returns ok == false when
// it is missing.
func (f File) ReadBytesIfExists() ([]byte, bool, error) {
	data, err := f.ReadBytes()
	if err == nil {
		return data, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, err
}

// Compose

// ComposePath binds the handle to path and resets composition metadata.
func (f *File) ComposePath(path string) {
	f.path = filepath.Clean(path)
	f.composeBase = ""
	f.composedBase = false
}

func (f *File) setComposeBase(path string) {
	f.composeBase = filepath.Clean(path)
	f.composedBase = true
}

func (f *File) setDeclaredPath(path string) {
	f.declaredPath = path
	f.hasDeclared = true
}

// Ensure

// Ensure creates parent directories and creates the file when it is missing.
//
// Existing contents are preserved.
func (f File) Ensure(ctx Context) error {
	if !ctx.ensurePolicy().allowsFile() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(f.path), ctx.DirMode); err != nil {
		return err
	}

	handle, err := os.OpenFile(f.path, os.O_CREATE, ctx.FileMode)
	if err != nil {
		return err
	}

	return handle.Close()
}

func splitBaseExt(base string) (stem string, ext string) {
	switch base {
	case "", ".", "..":
		return base, ""
	}

	if strings.HasPrefix(base, ".") && strings.Count(base, ".") == 1 {
		return base, ""
	}

	ext = filepath.Ext(base)
	return base[:len(base)-len(ext)], ext
}

func joinDeclaredPath(base string, parts ...string) string {
	if len(parts) == 0 {
		return base
	}
	if base == "." || base == "" {
		return filepath.Join(parts...)
	}
	return filepath.Join(append([]string{base}, parts...)...)
}

func (f File) openAppendDestination(dirMode os.FileMode, fileMode os.FileMode) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(f.path), dirMode); err != nil {
		return nil, err
	}
	return os.OpenFile(f.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileMode)
}

// Copy

// CopyToPath copies the file payload onto the exact destination path.
func (f File) CopyToPath(path string, opts CopyOptions) error {
	dst := filepath.Clean(path)
	if samePath(f.Path(), dst) {
		return fmt.Errorf("source and destination must differ: %s", f.Path())
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
