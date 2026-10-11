package layout

import (
	"hash"
	"io"
	"os"
	"time"
)

// FileAPI is the complete exported API of File, including pointer-receiver methods.
type FileAPI interface {
	AppendBytes(data []byte, ctx Context) error
	AppendFile(src File, ctx Context) (err error)
	AppendFiles(ctx Context, srcs ...File) error
	AppendReader(src io.Reader, ctx Context) error
	AppendString(content string, ctx Context) error
	Base() string
	Chmod(mode os.FileMode, ctx Context) error
	Chown(uid int, gid int, ctx Context) error
	Chtimes(atime time.Time, mtime time.Time, ctx Context) error
	ComposePath(path string)
	ComposedBaseDir() (Dir, bool)
	ComposedRelativePath() (string, bool)
	ConcatBytes(ctx Context, opts ConcatOptions, srcs ...[]byte) error
	ConcatFiles(ctx Context, opts ConcatOptions, srcs ...File) error
	ConcatReaders(ctx Context, opts ConcatOptions, srcs ...io.Reader) error
	ConcatStrings(ctx Context, opts ConcatOptions, srcs ...string) error
	CopyIntoDir(dir Dir, opts CopyOptions) error
	CopyToFile(dst File, opts CopyOptions) error
	CopyToPath(path string, opts CopyOptions) error
	DeclaredPath() (string, bool)
	DeleteIfExists(ctx Context) error
	Ensure(ctx Context) error
	Exists() bool
	Ext() string
	Fingerprint() FileFingerprint
	Hash(ctx Context, h hash.Hash) ([]byte, error)
	HashHex(ctx Context, h hash.Hash) (string, error)
	Inspect(ctx Context, inspect InspectFunc) error
	InspectLines(ctx Context, inspect InspectTokenFunc) error
	InspectTokens(ctx Context, opts TokenOptions, inspect InspectTokenFunc) error
	IsExecutable() bool
	JoinComposedPath(parts ...string) (string, bool)
	JoinDeclaredPath(parts ...string) (string, bool)
	JoinRelPathTo(target string, parts ...string) (string, error)
	JoinRelTo(base Pather, parts ...string) (string, error)
	JoinRelToPath(base string, parts ...string) (string, error)
	Lstat() (os.FileInfo, error)
	Match(ctx Context, match MatchFunc) (bool, error)
	MatchIfExists(ctx Context, match MatchFunc) (bool, error)
	MatchLines(ctx Context, match MatchTokenFunc) (bool, error)
	MatchLinesIfExists(ctx Context, match MatchTokenFunc) (bool, error)
	MatchTokens(ctx Context, opts TokenOptions, match MatchTokenFunc) (bool, error)
	MatchTokensIfExists(ctx Context, opts TokenOptions, match MatchTokenFunc) (bool, error)
	OpenAppend(ctx Context, op OpenPolicy) (*os.File, error)
	OpenFile(ctx Context, flag int, perm os.FileMode) (*os.File, error)
	OpenRead(ctx Context, op OpenPolicy) (*os.File, error)
	OpenReadAppend(ctx Context, op OpenPolicy) (*os.File, error)
	OpenReadRewrite(ctx Context, op OpenPolicy) (*os.File, error)
	OpenReadRewriteAppend(ctx Context, op OpenPolicy) (*os.File, error)
	OpenReadWrite(ctx Context, op OpenPolicy) (*os.File, error)
	OpenRewrite(ctx Context, op OpenPolicy) (*os.File, error)
	OpenRewriteAppend(ctx Context, op OpenPolicy) (*os.File, error)
	OpenWrite(ctx Context, op OpenPolicy) (*os.File, error)
	ParentDir() Dir
	ParentPath() string
	Path() string
	ReadBytes() ([]byte, error)
	ReadBytesIfExists() ([]byte, bool, error)
	RelPathTo(target string) (string, error)
	RelTo(base Pather) (string, error)
	RelToPath(base string) (string, error)
	SetComposeBase(path string)
	SetDeclaredPath(path string)
	Stat() (os.FileInfo, error)
	Stem() string
	Transform(ctx Context, transform TransformFunc) error
	TransformBytes(ctx Context, data []byte, transform TransformFunc) error
	TransformFile(ctx Context, src File, transform TransformFunc) error
	TransformReader(ctx Context, src io.Reader, transform TransformFunc) error
	TransformString(ctx Context, data string, transform TransformFunc) error
	Truncate(size int64, ctx Context) error
	Validate(opts ValidateOptions) error
	WriteBytes(data []byte, ctx Context) error
}
