package layout

import (
	"os"
	"time"
)

// DirAPI is the complete exported API of Dir, including pointer-receiver methods.
type DirAPI interface {
	Base() string
	ChangeTo() error
	Chmod(mode os.FileMode, ctx Context) error
	Chown(uid int, gid int, ctx Context) error
	Chtimes(atime time.Time, mtime time.Time, ctx Context) error
	ComposePath(path string)
	ComposedBaseDir() (Dir, bool)
	ComposedRelativePath() (string, bool)
	CopyIntoDir(parent Dir, opts CopyOptions) error
	CopyToDir(dst Dir, opts CopyOptions) error
	CopyToPath(path string, opts CopyOptions) error
	DeclaredPath() (string, bool)
	DeleteIfExists(ctx Context) error
	Dir(name string) Dir
	DirList() ([]Dir, error)
	Empty(ctx Context) error
	Ensure(ctx Context) error
	Exists() bool
	File(name string) File
	FileList() ([]File, error)
	Join(parts ...string) string
	JoinComposedPath(parts ...string) (string, bool)
	JoinDeclaredPath(parts ...string) (string, bool)
	JoinRelPathTo(target string, parts ...string) (string, error)
	JoinRelTo(base Pather, parts ...string) (string, error)
	JoinRelToPath(base string, parts ...string) (string, error)
	List() ([]os.DirEntry, error)
	Lstat() (os.FileInfo, error)
	Open(ctx Context) (*os.File, error)
	OpenRoot(ctx Context) (*os.Root, error)
	ParentDir() Dir
	ParentPath() string
	Path() string
	RelPathTo(target string) (string, error)
	RelTo(base Pather) (string, error)
	RelToPath(base string) (string, error)
	SetComposeBase(path string)
	SetDeclaredPath(path string)
	Stat() (os.FileInfo, error)
	Stem() string
	Validate(opts ValidateOptions) error
}
