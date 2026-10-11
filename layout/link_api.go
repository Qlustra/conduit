package layout

import (
	"os"
)

// LinkAPI is the complete exported API of Link, including pointer-receiver methods.
type LinkAPI interface {
	Base() string
	ClearTarget()
	ComposePath(path string)
	ComposedBaseDir() (Dir, bool)
	ComposedRelativePath() (string, bool)
	DeclaredPath() (string, bool)
	Delete(ctx Context) error
	Discover() (DiskState, error)
	DiskState() DiskState
	Exists() bool
	Ext() string
	HasBeenLoaded() bool
	HasContent() bool
	HasKnownDiskState() bool
	HasTarget() bool
	IsDangling() (bool, error)
	IsDirty() bool
	JoinComposedPath(parts ...string) (string, bool)
	JoinDeclaredPath(parts ...string) (string, bool)
	JoinRelPathTo(target string, parts ...string) (string, error)
	JoinRelTo(base Pather, parts ...string) (string, error)
	JoinRelToPath(base string, parts ...string) (string, error)
	Load() (bool, error)
	Lstat() (os.FileInfo, error)
	MemoryState() MemoryState
	MustTarget() string
	Path() string
	Readlink() (string, error)
	RelPathTo(target string) (string, error)
	RelTo(base Pather) (string, error)
	RelToPath(base string) (string, error)
	ResolvedTargetPath() (string, bool)
	Scan() (DiskState, error)
	SetComposeBase(path string)
	SetDeclaredPath(path string)
	SetDefaultTarget(target string) bool
	SetTarget(target string)
	Stem() string
	Sync(ctx Context) (ResultCode, error)
	Target() (string, bool)
	TargetExists() (bool, error)
	Unload()
	Validate(opts ValidateOptions) error
	WasObservedOnDisk() bool
}

// FileLinkAPI is the complete exported API of FileLink and its embedded Link.
type FileLinkAPI interface {
	LinkAPI
	MustTargetFile() File
	TargetFile() (File, bool)
}

// DirLinkAPI is the complete exported API of DirLink and its embedded Link.
type DirLinkAPI interface {
	LinkAPI
	MustTargetDir() Dir
	TargetDir() (Dir, bool)
}
