# Layout API

This page documents the exported `github.com/qlustra/conduit/layout` package.

## Types

### `Dir`

```go
type Dir struct{}
```

Description:

- stateless handle to a directory path

Methods:

- `Path() string`: returns the bound path
- `Base() string`: returns the final path element
- `Stem() string`: returns the final path element without its final extension
- `ParentPath() string`: returns `filepath.Dir(Path())`
- `ParentDir() Dir`: returns the parent directory handle
- `RelTo(base Pather) (string, error)`: returns the path relative to another node with a `Path()`
- `JoinRelTo(base Pather, parts ...string) (string, error)`: joins path parts onto the relative path from another node
- `RelToPath(base string) (string, error)`: returns the receiver path relative to a raw base path
- `JoinRelToPath(base string, parts ...string) (string, error)`: joins path parts onto the receiver path relative to a raw base path
- `RelPathTo(target string) (string, error)`: returns a target path relative to the receiver path
- `JoinRelPathTo(target string, parts ...string) (string, error)`: joins path parts onto a target path relative to the receiver path
- `DeclaredPath() (string, bool)`: returns the node's own declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the handle belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the composed-relative path
- `ComposePath(path string)`: binds the handle to a concrete path and resets composition metadata
- `Exists() bool`: reports whether the path currently exists
- `Chown(uid, gid int, ctx Context) error`: applies `os.Chown` to the directory path
- `Join(parts ...string) string`: joins descendant path segments onto the directory path
- `List() ([]os.DirEntry, error)`: returns the directory's direct children using `os.ReadDir`
- `ChangeTo() error`: changes the process working directory to this path
- `Dir(name string) Dir`: returns a child directory handle
- `File(name string) File`: returns a child file handle
- `CopyToPath(path string, opts CopyOptions) error`: copies the directory tree onto an exact destination path
- `CopyToDir(dst Dir, opts CopyOptions) error`: same exact-path directory copy using `dst.Path()`
- `CopyIntoDir(parent Dir, opts CopyOptions) error`: copies the directory tree under `parent` using the source basename
- `Validate(opts ValidateOptions) error`: validates that the path is either missing or a directory and that validation policy accepts the path shape
- `Empty(ctx Context) error`: removes all children while preserving the directory itself
- `DeleteIfExists(ctx Context) error`: removes the directory tree when it exists
- `Ensure(ctx Context) error`: creates the directory tree using `ctx.DirMode`

Notable behavior:

- `DeleteIfExists` uses recursive removal
- `Empty` rejects symlink roots and removes symlink children as entries without following them
- mutating `Dir` operations reject symlink leaves; `Link` is the type that manages symlink entries intentionally
- mutating `Dir` operations also reject symlink parents by default; set `Context.PathSafetyPolicy` to `PathSafetyFollowSymlinks` to opt in to path-following behavior
- `Exists` only checks current filesystem state; it does not validate that the path is a directory
- `List` returns entries sorted by filename, matching `os.ReadDir`
- `ParentDir` preserves compose-base metadata when the receiver belongs to a composed tree
- `RelToPath` and `RelPathTo` differ only by direction: `RelToPath` makes the receiver relative to a base, while `RelPathTo` makes a target relative to the receiver
- the declared-path helpers return `ok == false` when the handle was not attached through `Compose`
- for a root field declared as `layout:"."`, `DeclaredPath()` returns `.`
- the composed-path helpers return `ok == false` when the handle was not attached through `Compose`
- for the compose base directory itself, `ComposedRelativePath()` returns `.`

### `File`

```go
type File struct{}
```

Description:

- stateless handle to a regular file path

Methods:

- `Path() string`: returns the bound path
- `Base() string`: returns the final path element
- `Ext() string`: returns the final extension including the leading dot
- `Stem() string`: returns the final path element without its final extension
- `ParentPath() string`: returns `filepath.Dir(Path())`
- `ParentDir() Dir`: returns the parent directory handle
- `RelTo(base Pather) (string, error)`: returns the path relative to another node with a `Path()`
- `JoinRelTo(base Pather, parts ...string) (string, error)`: joins path parts onto the relative path from another node
- `RelToPath(base string) (string, error)`: returns the receiver path relative to a raw base path
- `JoinRelToPath(base string, parts ...string) (string, error)`: joins path parts onto the receiver path relative to a raw base path
- `RelPathTo(target string) (string, error)`: returns a target path relative to the receiver path
- `JoinRelPathTo(target string, parts ...string) (string, error)`: joins path parts onto a target path relative to the receiver path
- `DeclaredPath() (string, bool)`: returns the node's own declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the handle belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the composed-relative path
- `ComposePath(path string)`: binds the handle to a concrete path and resets composition metadata
- `Exists() bool`: reports whether the path currently exists
- `Chown(uid, gid int, ctx Context) error`: applies `os.Chown` to the file path
- `IsExecutable() bool`: reports whether the current target is an executable regular file
- `Truncate(size int64, ctx Context) error`: resizes the file using `os.Truncate`
- `AppendReader(src io.Reader, ctx Context) error`: creates parent directories if needed and appends bytes read from a reader
- `AppendBytes(data []byte, ctx Context) error`: creates parent directories if needed and appends raw bytes
- `AppendString(content string, ctx Context) error`: creates parent directories if needed and appends string content
- `AppendFile(src File, ctx Context) error`: creates parent directories if needed and appends another file's payload
- `AppendFiles(ctx Context, srcs ...File) error`: appends multiple file payloads in order
- `WriteBytes(data []byte, ctx Context) error`: creates parent directories and writes raw bytes
- `ReadBytes() ([]byte, error)`: reads the file contents
- `ReadBytesIfExists() ([]byte, bool, error)`: reads the file if present and returns `ok == false` for missing files
- `CopyToPath(path string, opts CopyOptions) error`: copies the file payload to an exact destination path
- `CopyToFile(dst File, opts CopyOptions) error`: same exact-path file copy using `dst.Path()`
- `CopyIntoDir(dir Dir, opts CopyOptions) error`: copies the file under `dir` using the source basename
- `Validate(opts ValidateOptions) error`: validates that the path is either missing or a regular file and that validation policy accepts the path shape
- `DeleteIfExists(ctx Context) error`: removes the file when it exists
- `Ensure(ctx Context) error`: creates the parent directories and creates the file if missing

Notable behavior:

- `Ensure` uses `os.O_CREATE` and does not truncate existing file contents
- `AppendReader`, `AppendFile`, and `AppendFiles` stream through `io.Copy`; they do not read the whole source into memory first
- `AppendFiles` appends sources in argument order and may leave already-appended content in place if a later source fails
- append helpers create parent directories and the destination file when missing
- concurrent append calls rely on OS append mode for destination offset management, but no whole-call atomicity is guaranteed
- `WriteBytes` always rewrites the file contents
- mutating `File` operations reject symlink leaves; `Link` is the type that manages symlink entries intentionally
- mutating `File` operations reject symlink parents by default; set `Context.PathSafetyPolicy` to `PathSafetyFollowSymlinks` to opt in to path-following behavior
- `IsExecutable` returns false for missing paths and non-regular filesystem entries
- `CopyTo*` uses streamed I/O through `io.Copy`; it does not read the whole file into memory first
- `Exists` only checks that some filesystem entry exists at the path
- the declared-path helpers return `ok == false` when the handle was not attached through `Compose`
- the composed-path helpers return `ok == false` when the handle was not attached through `Compose`
- `ParentDir` preserves compose-base metadata when the receiver belongs to a composed tree
- `RelToPath` and `RelPathTo` differ only by direction: `RelToPath` makes the receiver relative to a base, while `RelPathTo` makes a target relative to the receiver
- dotfiles such as `.env` report an empty extension and keep the full basename as the stem

### `CopyOptions`

```go
type CopyOptions struct{}
```

Description:

- policy object for `File` and `Dir` copy helpers

Fields:

- `Overwrite CopyOverwritePolicy`: controls whether an existing destination is rejected or replaced
- `Symlinks CopySymlinkPolicy`: controls whether symlinks are preserved, followed, or rejected
- `PathSafetyPolicy PathSafetyPolicy`: controls whether destination path resolution rejects symlink parents
- `PreserveMode bool`: when true, copies use source permission bits for new files and directories
- `FileMode os.FileMode`: fallback file mode when `PreserveMode` is false
- `DirMode os.FileMode`: fallback directory mode when `PreserveMode` is false

Notable behavior:

- `DefaultCopyOptions` preserves source symlinks, preserves source modes, rejects symlink parents in destination paths, and fails when the destination already exists
- the zero `CopyOptions{}` value is treated as `DefaultCopyOptions`
- when `PreserveMode` is false and a mode field is zero, copy falls back to `DefaultContext.FileMode` or `DefaultContext.DirMode`

### `CopyOverwritePolicy`

```go
type CopyOverwritePolicy uint8
```

Constants:

- `CopyOverwriteFail`: return an error when the destination path already exists
- `CopyOverwriteReplace`: remove the existing destination path before copying

### `CopySymlinkPolicy`

```go
type CopySymlinkPolicy uint8
```

Constants:

- `CopySymlinkPreserve`: recreate symlinks as symlinks using the raw source target string
- `CopySymlinkFollow`: copy the symlink target payload instead of the symlink entry
- `CopySymlinkReject`: fail when a symlink is encountered

### `Exec`

```go
type Exec struct{}
```

Description:

- executable file handle with process-launch helpers

Methods:

- `Path() string`
- `Base() string`
- `Ext() string`
- `Stem() string`
- `ParentPath() string`
- `ParentDir() Dir`
- `RelTo(base Pather) (string, error)`
- `JoinRelTo(base Pather, parts ...string) (string, error)`
- `RelToPath(base string) (string, error)`
- `JoinRelToPath(base string, parts ...string) (string, error)`
- `RelPathTo(target string) (string, error)`
- `JoinRelPathTo(target string, parts ...string) (string, error)`
- `DeclaredPath() (string, bool)`
- `JoinDeclaredPath(parts ...string) (string, bool)`
- `ComposedBaseDir() (Dir, bool)`
- `ComposedRelativePath() (string, bool)`
- `JoinComposedPath(parts ...string) (string, bool)`
- `ComposePath(path string)`
- `Exists() bool`
- `Chown(uid, gid int, ctx Context) error`
- `Truncate(size int64, ctx Context) error`
- `AppendReader(src io.Reader, ctx Context) error`
- `AppendBytes(data []byte, ctx Context) error`
- `AppendString(content string, ctx Context) error`
- `AppendFile(src File, ctx Context) error`
- `AppendFiles(ctx Context, srcs ...File) error`
- `ReadBytes() ([]byte, error)`
- `ReadBytesIfExists() ([]byte, bool, error)`
- `WriteBytes(data []byte, ctx Context) error`
- `DeleteIfExists(ctx Context) error`
- `CopyToPath(path string, opts CopyOptions) error`
- `CopyToFile(dst File, opts CopyOptions) error`
- `CopyIntoDir(dir Dir, opts CopyOptions) error`
- `Validate(opts ValidateOptions) error`: validates executable file shape and validation-path policy
- `Ensure(ctx Context) error`: creates the file and ensures executable permissions
- `EnsureExecutable(ctx Context) error`: same executable materialization behavior as `Ensure`
- `IsExecutable() bool`: reports whether the current target is an executable regular file
- `Command(ctx context.Context, opts RunOptions) *exec.Cmd`: builds an `exec.Cmd`
- `Run(ctx context.Context, opts RunOptions) error`: runs the executable
- `Output(ctx context.Context, opts RunOptions) ([]byte, error)`: runs and captures stdout
- `CombinedOutput(ctx context.Context, opts RunOptions) ([]byte, error)`: runs and captures combined stdout and stderr
- `EnsureDeep(ctx Context) (ResultCode, error)`: ensures the executable node during deep traversal

Notable behavior:

- `Ensure` and `EnsureExecutable` apply `Context.ExecMode`, or `FileMode` with execute bits added when `ExecMode` is zero
- mutating `Exec` operations reject symlink leaves and reject symlink parents by default; set `Context.PathSafetyPolicy` to `PathSafetyFollowSymlinks` to opt in to path-following behavior
- `Command` returns an `*exec.Cmd` even when configuration is invalid; the error is stored on `cmd.Err`
- `Run` fails when `ctx` is nil
- `Output` and `CombinedOutput` reject explicit `Stdout` or `Stderr` writers in `RunOptions`
- `Interpreter` runs the managed file as an argument to the interpreter command instead of executing it directly

### `Link`

```go
type Link struct{}
```

Description:

- symlink handle with cached target state and lifecycle helpers

Methods:

- `Path() string`: returns the bound link path
- `Base() string`: returns the final path element
- `Ext() string`: returns the final extension including the leading dot
- `Stem() string`: returns the final path element without its final extension
- `RelTo(base Pather) (string, error)`: returns the path relative to another node with a `Path()`
- `JoinRelTo(base Pather, parts ...string) (string, error)`: joins path parts onto the relative path from another node
- `RelToPath(base string) (string, error)`: returns the path relative to a raw base path
- `JoinRelToPath(base string, parts ...string) (string, error)`: joins path parts onto the relative path from a raw base path
- `RelPathTo(target string) (string, error)`: returns a target path relative to the link path
- `JoinRelPathTo(target string, parts ...string) (string, error)`: joins path parts onto a target path relative to the link path
- `DeclaredPath() (string, bool)`: returns the node's own declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the handle belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the composed-relative path
- `ComposePath(path string)`: binds the link to a concrete path and resets cached target and state
- `Exists() bool`: reports whether a symlink exists at the path
- `Target() (string, bool)`: returns the cached raw symlink target string
- `MustTarget() string`: returns the cached target string or panics when it is absent
- `SetTarget(target string)`: stores a raw symlink target string in memory and marks it dirty
- `SetDefaultTarget(target string) bool`: sets the target only when it is currently absent
- `HasTarget() bool`: reports whether a target string is currently cached
- `HasContent() bool`: same cached-target check used by the generic load/sync contracts
- `ClearTarget()`: drops the cached target string
- `ResolvedTargetPath() (string, bool)`: resolves the cached target against the link's parent directory when it is relative
- `TargetExists() (bool, error)`: reports whether the resolved target currently exists
- `IsDangling() (bool, error)`: reports whether the cached target is currently missing
- `Delete(ctx Context) error`: removes the symlink when it exists
- `Validate(opts ValidateOptions) error`: validates that the path is either missing or a symlink and that validation policy accepts the path shape
- `Load() (bool, error)`: reads the symlink target from disk
- `Unload()`: drops the cached target string without touching disk
- `Discover() (DiskState, error)`: observes the symlink's presence on disk without loading a new target
- `Scan() (DiskState, error)`: observes the symlink's presence on disk without changing cached memory content
- `Sync(ctx Context) (ResultCode, error)`: creates or rewrites the symlink from the cached target string when policy allows
- `DiskState() DiskState`: returns the cached disk state
- `MemoryState() MemoryState`: returns the cached memory state
- `HasKnownDiskState() bool`: reports whether disk state has been observed
- `WasObservedOnDisk() bool`: reports whether the last disk observation found a symlink
- `HasBeenLoaded() bool`: reports whether target state has been loaded, synced, or dirtied in memory
- `IsDirty() bool`: reports whether the cached target has been modified in memory since load or sync

Notable behavior:

- `Exists` uses `os.Lstat`, so dangling symlinks still count as existing
- `Load` succeeds for dangling symlinks because `os.Readlink` returns the raw target string
- `Scan`, `Discover`, and `Load` fail when the path exists but is not a symlink
- `Sync` manages only the symlink entry at `Path()` and its parent directory
- `Link` does not participate in `EnsureDeep`; links are materialized through `Sync`/`SyncDeep` after a target is set

### `FileLink`

```go
type FileLink struct{}
```

Description:

- symlink wrapper that exposes the resolved target as a `File` handle

Methods:

- all promoted `Link` methods
- `Validate(opts ValidateOptions) error`: validates the link entry plus resolved target file type when a target is cached
- `TargetFile() (File, bool)`: resolves the cached target to a `File` handle
- `MustTargetFile() File`: panicking version of `TargetFile()`

### `DirLink`

```go
type DirLink struct{}
```

Description:

- symlink wrapper that exposes the resolved target as a `Dir` handle

Methods:

- all promoted `Link` methods
- `Validate(opts ValidateOptions) error`: validates the link entry plus resolved target directory type when a target is cached
- `TargetDir() (Dir, bool)`: resolves the cached target to a `Dir` handle
- `MustTargetDir() Dir`: panicking version of `TargetDir()`

### `RunOptions`

```go
type RunOptions struct {
	Dir         string
	Args        []string
	Env         []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interpreter []string
}
```

Fields:

- `Dir`: working directory for the spawned process
- `Args`: arguments passed to the executable or interpreter invocation
- `Env`: additional environment variables appended to the current process environment
- `Stdin`: reader connected to process stdin
- `Stdout`: writer connected to process stdout
- `Stderr`: writer connected to process stderr
- `Interpreter`: command prefix used to invoke the file through another program

Notable behavior:

- when `Interpreter` is set, the spawned argv is `Interpreter + file path + Args`
- an empty first interpreter element is invalid
- `Env` is appended to `os.Environ()`, not used as a complete replacement

### `Pather`

```go
type Pather interface {
	Path() string
}
```

Description:

- minimal path-bearing contract used by relative path helpers

Notable behavior:

- implemented naturally by `Dir`, `File`, `Exec`, `Slot[T]`, `FileSlot[T]`, typed files, and text-template wrappers that expose `Path()`
- used by `RelTo(...)` and `JoinRelTo(...)` without requiring method overloading

### `Defaulter`

```go
type Defaulter interface {
	Default() error
}
```

Description:

- opt-in contract for nodes that can seed missing in-memory state with defaults

Notable behavior:

- `Default()` applies defaults in memory only
- `conduit.DefaultDeep` calls `Default()` on matching nodes
- concrete implementations decide which values are considered unset and whether to apply defaults

### `Renderable`

```go
type Renderable interface {
	Render() (string, error)
	SetRendered(string)
}
```

Description:

- opt-in contract for text-template wrappers that can derive raw text into memory

Notable behavior:

- `Render()` computes text but does not write to disk by itself
- `SetRendered(string)` stores rendered text into the target's in-memory file state
- `conduit.RenderDeep` calls `Render()` and passes the result into `SetRendered(string)`
- concrete implementations decide how to handle missing or incomplete render context

### `Templatable`

```go
type Templatable interface {
	Template() string
	RenderTemplate(string) (string, error)
	SetRendered(string)
}
```

Description:

- opt-in contract for the built-in `text/template` render path

Notable behavior:

- `Template()` returns the source template text
- `RenderTemplate(string)` executes template text against the current cached render context
- `SetRendered(string)` stores rendered text into the target's in-memory file state
- `conduit.RenderDeep` uses this path only when the node does not implement `Renderable`

### `Slot[T]`

```go
type Slot[T any] struct{}
```

### `SlotEntry[T]`

```go
type SlotEntry[T any] struct {
	Name string
	Item T
}
```

Description:

- keyed container for repeated child layouts rooted under one directory

Methods:

- `Path() string`: returns the slot root path
- `DeclaredPath() (string, bool)`: returns the slot field's declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the slot's declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the slot belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the slot root path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the slot's composed-relative path
- `ComposePath(path string)`: binds the slot root and clears the cache
- `Exists() bool`: reports whether the slot root exists on disk
- `Root() Dir`: returns the slot root as a `Dir`
- `Len() int`: returns the number of cached items
- `Has(name string) bool`: reports whether a named child directory exists on disk
- `Get(name string) (T, bool)`: returns a cached item only
- `Put(name string, item T)`: inserts or replaces a cached item
- `Remove(name string)`: removes a cached item
- `Delete(name string, ctx Context) error`: removes the child tree from disk if present and evicts the cached item
- `Clear()`: clears the cache
- `Entries() []SlotEntry[T]`: returns a sorted snapshot of cached entries
- `All() iter.Seq2[string, T]`: iterates cached entries in sorted key order
- `Keys() []string`: returns sorted cached keys
- `At(name string) (T, error)`: returns a cached item or composes and caches one lazily
- `MustAt(name string) T`: panicking form of `At`
- `Add(name string, ctx Context) (T, error)`: creates the child root on disk, composes the item, ensures its structure, and caches it
- `Require(name string) (T, error)`: returns an item only when the child root already exists on disk
- `Ensure(ctx Context) error`: ensures the slot root directory
- `EnsureDeep(ctx Context) (ResultCode, error)`: ensures the slot root and all cached items
- `DiscoverDeep(ctx Context) (ResultCode, error)`: discovers child directories on disk and scans them without loading typed content
- `LoadDeep(ctx Context) (ResultCode, error)`: discovers child directories on disk and loads them
- `ScanDeep(ctx Context) (ResultCode, error)`: scans only cached items
- `SyncDeep(ctx Context) (ResultCode, error)`: ensures cached items, then syncs typed content within those items
- `RenderDeep() error`: renders only currently cached items
- `DefaultDeep() error`: applies defaults only to currently cached items
- `ValidateDeep(opts ValidateOptions) (ResultCode, error)`: validates only currently cached items

Notable behavior:

- `At` composes items relative to `slotRoot/<name>` and caches them
- `Add` ensures the child root and calls `EnsureDeep` on the new child
- `Delete` removes both the on-disk child tree and the cached entry
- `Len`, `Entries`, `All`, and `Keys` are cache-based; they do not list the filesystem directly
- `Entries` and `All` return cached items as-is, preserving pointer or value semantics chosen by `T`
- the declared-path helpers delegate to the slot root and expose the slot field's own declared fragment
- the composed-path helpers delegate to the slot root and return `ok == false` until the slot has been attached through `Compose`
- `DiscoverDeep` discovers directory-backed entries from disk without loading typed files
- `LoadDeep` discovers directory-backed entries from disk
- `ScanDeep` and `SyncDeep` do not discover uncached entries
- `Slot.SyncDeep` ensures cached children before syncing them, and that preparation ensure pass respects `Context.EnsurePolicy`

### `FileSlot[T]`

```go
type FileSlot[T any] struct{}
```

### `FileSlotEntry[T]`

```go
type FileSlotEntry[T any] struct {
	Name string
	Item T
}
```

Description:

- keyed container for repeated direct-child files rooted under one directory

Methods:

- `Path() string`: returns the slot root path
- `DeclaredPath() (string, bool)`: returns the slot field's declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the slot's declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the slot belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the slot root path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the slot's composed-relative path
- `ComposePath(path string)`: binds the slot root and clears the cache
- `Exists() bool`: reports whether the slot root exists on disk
- `Root() Dir`: returns the slot root as a `Dir`
- `Len() int`: returns the number of cached items
- `Has(name string) bool`: reports whether a named child file exists on disk
- `Get(name string) (T, bool)`: returns a cached item only
- `Put(name string, item T)`: inserts or replaces a cached item
- `Remove(name string)`: removes a cached item
- `Delete(name string, ctx Context) error`: removes the child file from disk if present and evicts the cached item
- `Clear()`: clears the cache
- `Entries() []FileSlotEntry[T]`: returns a sorted snapshot of cached entries
- `All() iter.Seq2[string, T]`: iterates cached entries in sorted key order
- `Keys() []string`: returns sorted cached keys
- `At(name string) (T, error)`: returns a cached item or composes and caches one lazily
- `MustAt(name string) T`: panicking form of `At`
- `Add(name string, ctx Context) (T, error)`: ensures the slot root, composes the file-backed item, ensures it, and caches it
- `Require(name string) (T, error)`: returns an item only when the child file already exists on disk
- `Ensure(ctx Context) error`: ensures the slot root directory
- `EnsureDeep(ctx Context) (ResultCode, error)`: ensures the slot root and all cached items
- `DiscoverDeep(ctx Context) (ResultCode, error)`: discovers child files on disk and scans them without loading typed content
- `LoadDeep(ctx Context) (ResultCode, error)`: discovers child files on disk and loads them
- `ScanDeep(ctx Context) (ResultCode, error)`: scans only cached items
- `SyncDeep(ctx Context) (ResultCode, error)`: ensures cached items, then syncs typed content within those items
- `RenderDeep() error`: renders only currently cached items
- `DefaultDeep() error`: applies defaults only to currently cached items
- `ValidateDeep(opts ValidateOptions) (ResultCode, error)`: validates only currently cached items

Notable behavior:

- `At` composes items relative to `slotRoot/<name>` and caches them
- `Add` ensures the slot root and calls `EnsureDeep` on the item
- `Delete` removes both the on-disk child file and the cached entry
- item names must identify a single direct child; empty, absolute, dot, dot-dot, and separator-containing names are rejected
- `Has` and `Require` treat only regular files as valid child entries; symlinks are rejected
- `Len`, `Entries`, `All`, and `Keys` are cache-based; they do not list the filesystem directly
- `Entries` and `All` return cached items as-is, preserving pointer or value semantics chosen by `T`
- the declared-path helpers delegate to the slot root and expose the slot field's own declared fragment
- the composed-path helpers delegate to the slot root and return `ok == false` until the slot has been attached through `Compose`
- `DiscoverDeep` and `LoadDeep` discover direct child regular files from disk and ignore subdirectories and symlinks
- `ScanDeep` and `SyncDeep` do not discover uncached entries
- `FileSlot.SyncDeep` ensures cached children before syncing them, and that preparation ensure pass respects `Context.EnsurePolicy`

### `LinkSlot[T]`

```go
type LinkSlot[T layout.LinkSlotItem] struct{}
```

### `LinkSlotEntry[T]`

```go
type LinkSlotEntry[T layout.LinkSlotItem] struct {
	Name string
	Item T
}
```

Description:

- keyed container for repeated direct-child symlink entries rooted under one directory

Methods:

- `Path() string`: returns the slot root path
- `DeclaredPath() (string, bool)`: returns the slot field's declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the slot's declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the slot belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the slot root path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the slot's composed-relative path
- `ComposePath(path string)`: binds the slot root and clears the cache
- `Exists() bool`: reports whether the slot root exists on disk
- `Root() Dir`: returns the slot root as a `Dir`
- `Len() int`: returns the number of cached items
- `Has(name string) bool`: reports whether a named child symlink exists on disk
- `Get(name string) (T, bool)`: returns a cached item only
- `Put(name string, item T)`: inserts or replaces a cached item
- `Remove(name string)`: removes a cached item
- `Delete(name string, ctx Context) error`: removes the child symlink from disk if present and evicts the cached item
- `Clear()`: clears the cache
- `Entries() []LinkSlotEntry[T]`: returns a sorted snapshot of cached entries
- `All() iter.Seq2[string, T]`: iterates cached entries in sorted key order
- `Keys() []string`: returns sorted cached keys
- `At(name string) (T, error)`: returns a cached item or composes and caches one lazily
- `MustAt(name string) T`: panicking form of `At`
- `Add(name string, ctx Context) (T, error)`: ensures the slot root, composes the link item, and caches it
- `Require(name string) (T, error)`: returns an item only when the child symlink already exists on disk
- `Ensure(ctx Context) error`: ensures the slot root directory
- `EnsureDeep(ctx Context) (ResultCode, error)`: ensures the slot root and visits cached items without materializing the links themselves
- `DiscoverDeep(ctx Context) (ResultCode, error)`: discovers child symlinks on disk and scans them without loading target content
- `LoadDeep(ctx Context) (ResultCode, error)`: discovers child symlinks on disk and loads them
- `ScanDeep(ctx Context) (ResultCode, error)`: scans only cached items
- `SyncDeep(ctx Context) (ResultCode, error)`: ensures cached items, then syncs link entries within those items
- `ValidateDeep(opts ValidateOptions) (ResultCode, error)`: validates only currently cached items

Notable behavior:

- `T` is restricted to the built-in link family: `Link`, `FileLink`, or `DirLink`
- `At` composes items relative to `slotRoot/<name>` and caches them
- `Add` ensures the slot root but does not materialize the symlink entry itself; links are created by `Sync`/`SyncDeep`
- `Delete` removes only symlink entries and returns an error when a non-symlink entry exists at the child path
- item names must identify a single direct child; empty, absolute, dot, dot-dot, and separator-containing names are rejected
- `Len`, `Entries`, `All`, and `Keys` are cache-based; they do not list the filesystem directly
- `Entries` and `All` return cached items as-is, preserving value semantics chosen by `T`
- the declared-path helpers delegate to the slot root and expose the slot field's own declared fragment
- the composed-path helpers delegate to the slot root and return `ok == false` until the slot has been attached through `Compose`
- `DiscoverDeep` and `LoadDeep` discover symlink entries from disk and ignore regular files and directories
- `ScanDeep` and `SyncDeep` do not discover uncached entries
- `LinkSlot.SyncDeep` ensures cached children before syncing them, and that preparation ensure pass respects `Context.EnsurePolicy`

### `TextTemplate[C]`

```go
type TextTemplate[C any] struct{}
```

Description:

- stateful raw-text file with cached render context

Exposed API:

- all string-content operations analogous to `Format[string]`
- all inherited `File` methods from embedded raw-text file state

Context methods:

- `SetContext(ctx C)`: replaces cached render context
- `SetDefaultContext(ctx C) bool`: stores default render context only when no context is currently cached
- `GetContext() (C, bool)`: returns cached render context if present
- `MustContext() C`: returns cached render context or panics when unset
- `HasContext() bool`: reports whether render context is currently cached
- `ClearContext()`: clears cached render context
- `RenderTemplate(tpl string) (string, error)`: executes template text against the current render context using the built-in `text/template` renderer
- `SetRendered(value string)`: stores rendered text in the file's cached content

Notable behavior:

- mirrors the same disk and memory state model as `Format[string]`
- resets cached render context when `Compose` rebinds the file path
- `SetDefaultContext` returns `false` and leaves context unchanged when context is already cached
- provides the built-in templated render path used by `Templatable`
- leaves custom render validation and semantics to the user-defined `Render()` implementation

## Additional Reference

### `Context`

```go
type Context struct {
	DirMode          os.FileMode
	FileMode         os.FileMode
	ExecMode         os.FileMode
	EnsurePolicy     EnsurePolicy
	SyncPolicy       SyncPolicy
	PathSafetyPolicy PathSafetyPolicy
	Reporter         Reporter
}
```

Fields:

- `DirMode`: mode used when creating directories
- `FileMode`: mode used when creating regular files
- `ExecMode`: mode used when creating or ensuring `Exec` files
- `EnsurePolicy`: selects which node kinds `Ensure` and `EnsureDeep` may materialize
- `SyncPolicy`: selects which typed memory states `Sync` and `SyncDeep` may write, with optional disk-state filters
- `PathSafetyPolicy`: controls whether mutating typed filesystem operations reject symlink parents during path resolution
- `Reporter`: optional sink for per-path deep-operation results

### `DefaultContext`

Type:

```go
var DefaultContext Context
```

Default value:

```go
Context{
	DirMode:          0o755,
	FileMode:         0o644,
	ExecMode:         0o755,
	EnsurePolicy:     EnsureAll,
	SyncPolicy:       SyncRewrite,
	PathSafetyPolicy: PathSafetyRejectSymlinkParents,
}
```

### `DefaultCopyOptions`

Type:

```go
var DefaultCopyOptions CopyOptions
```

Default value:

```go
CopyOptions{
	Overwrite:        CopyOverwriteFail,
	Symlinks:         CopySymlinkPreserve,
	PathSafetyPolicy: PathSafetyRejectSymlinkParents,
	PreserveMode:     true,
}
```

### `PathSafetyPolicy`

```go
type PathSafetyPolicy uint8
```

Constants:

- `PathSafetyRejectSymlinkParents`: reject existing symlink parents during mutating path resolution
- `PathSafetyFollowSymlinks`: preserve path-following behavior for mutating operations

### `EnsurePolicy`

```go
type EnsurePolicy uint8
```

Constants:

- `EnsureDirs`: include raw directories
- `EnsureFiles`: include raw files
- `EnsureExecs`: include executable files
- `EnsureSyncables`: include syncable stateful wrappers such as `Format`
- `EnsureAll`: historical ensure behavior
- `EnsureScaffold`: raw `Dir`, `File`, and `Exec` scaffolding only
- `EnsureNone`: explicit no-op ensure policy

Methods:

- `Allow(bits EnsurePolicy) EnsurePolicy`: enables public ensure-policy bits
- `Deny(bits EnsurePolicy) EnsurePolicy`: disables public ensure-policy bits
- `Has(bits EnsurePolicy) bool`: reports whether all requested public bits are enabled

### `SyncPolicy`

```go
type SyncPolicy uint8
```

Constants:

- `SyncOnLoaded`: include `MemoryLoaded`
- `SyncOnSynced`: include `MemorySynced`
- `SyncOnDirty`: include `MemoryDirty`
- `SyncOnDiskUnknown`: include `DiskUnknown`
- `SyncOnDiskMissing`: include `DiskMissing`
- `SyncOnDiskPresent`: include `DiskPresent`
- `SyncRewrite`: include loaded, synced, and dirty states
- `SyncIfDirty`: include only dirty state
- `SyncIfUnsynced`: include loaded and dirty states
- `SyncIfMissing`: include any writable memory state, but only when the last known disk state is missing

### `DiskState`

```go
type DiskState uint8
```

Constants:

- `DiskUnknown`: disk state has not been observed, or known state was discarded during composition
- `DiskMissing`: the entry was checked and was absent on disk
- `DiskPresent`: the entry was checked and was present on disk

### `MemoryState`

```go
type MemoryState uint8
```

Constants:

- `MemoryUnknown`: no meaningful in-memory content is currently loaded
- `MemoryLoaded`: in-memory content reflects what was loaded from disk
- `MemorySynced`: in-memory content was written to disk by Conduit
- `MemoryDirty`: in-memory content was set or changed after load or sync

### `ValidateOptions`

```go
type ValidateOptions struct {
	Reporter Reporter
	PathSafetyPolicy PathSafetyPolicy
}
```

Fields:

- `Reporter`: optional sink for per-path validation results
- `PathSafetyPolicy`: controls whether built-in typed filesystem nodes reject symlink parents during validation

### `Report`

```go
type Report struct { ... }
```

Description:

- in-memory collector for operation entries recorded during deep traversal

Methods:

- `Record(Entry)`: appends one entry to the report
- `Entries() []Entry`: returns a snapshot copy of recorded entries
- `Len() int`: returns the number of recorded entries
- `HasErrors() bool`: reports whether any recorded entry carries an error
- `Filter(func(Entry) bool) []Entry`: returns a filtered snapshot
- `Sort(func(Entry, Entry) bool)`: reorders the recorded entries in place
- `SortByPath()`: sorts entries by path, then operation, then result
- `RenderTree() string`: renders recorded entries as a path-oriented tree

### `Entry`

```go
type Entry struct {
	Op     Operation
	Path   string
	Result ResultCode
	Err    error
}
```

Methods:

- `IsError() bool`: reports whether the entry carries an error
- `IsSkipped() bool`: reports whether the result represents a visited-but-not-applied outcome
- `IsSuccess() bool`: reports whether the entry completed without error
- `ResultName() string`: returns a stable lowercase result name relative to `Op`

### `Operation`

```go
type Operation uint8
```

Constants:

- `OpEnsure`
- `OpLoad`
- `OpDiscover`
- `OpScan`
- `OpSync`
- `OpValidate`

Methods:

- `String() string`: returns the lowercase operation name used in reports

### `ResultCode`

```go
type ResultCode uint8
```

Constants:

- ensure results: `EnsureEnsured`, `EnsureSkippedPolicy`, `EnsureFailed`
- load results: `LoadLoaded`, `LoadMissing`, `LoadTraversed`, `LoadNotApplicable`, `LoadFailed`
- discover results: `DiscoverPresent`, `DiscoverMissing`, `DiscoverTraversed`, `DiscoverNotApplicable`, `DiscoverFailed`
- scan results: `ScanPresent`, `ScanMissing`, `ScanTraversed`, `ScanNotApplicable`, `ScanFailed`
- sync results: `SyncWritten`, `SyncTraversed`, `SyncNotApplicable`, `SyncSkippedNoContent`, `SyncSkippedPolicy`, `SyncFailed`
- validate results: `ValidateOK`, `ValidateTraversed`, `ValidateNotApplicable`, `ValidateFailed`

### `Codec[T]`

```go
type Codec[T any] interface {
	Marshal(T) ([]byte, error)
	Unmarshal([]byte) (T, error)
}
```

Description:

- codec contract used by `Format[T, C]`

### `Format[T, C]`

```go
type Format[T any, C Codec[T]] struct{}
```

Description:

- codec-backed typed file with explicit disk and memory state

Exposed API:

- all promoted `File` methods from the embedded raw file handle
- `ComposePath(path string)`: binds the format to a path and resets cached content and state
- `Get() (T, bool)`: returns the cached typed value, if any
- `MustGet() T`: returns the cached value or panics when absent
- `Set(value T)`: replaces cached content and marks memory state dirty
- `SetDefault(value T) bool`: stores a default value only when no content is cached
- `Clear()`: clears cached content and resets memory state to unknown
- `Delete(ctx Context) error`: removes the file from disk, clears cached content, and marks disk state missing
- `Write(value T, ctx Context) error`: marshals and writes a supplied value directly without changing cached state
- `Read() (T, error)`: reads and unmarshals from disk without changing cached state
- `ReadIfExists() (T, bool, error)`: reads and unmarshals when the file exists
- `LoadOrInit(defaultValue T) error`: loads existing content or stores a default value in memory when missing
- `EnsureDeep(ctx Context) (ResultCode, error)`: ensures the backing file when `ctx.EnsurePolicy` includes syncable nodes
- `Save(ctx Context) error`: writes the cached value and marks memory state synced
- `Load() (bool, error)`: loads content into memory and reports whether the file existed
- `HasContent() bool`: reports whether a value is currently cached
- `Unload()`: clears cached content and resets memory state to unknown
- `Discover() (DiskState, error)`: refreshes disk-state metadata without replacing cached content
- `Sync(ctx Context) (ResultCode, error)`: writes cached content when present and allowed by `ctx.SyncPolicy`
- `DiskState() DiskState`: returns current disk-state metadata
- `MemoryState() MemoryState`: returns current memory-state metadata
- `HasKnownDiskState() bool`: reports whether disk state is not unknown
- `WasObservedOnDisk() bool`: reports whether the last known disk state is present
- `HasBeenLoaded() bool`: reports whether memory state has reached loaded, synced, or dirty
- `IsDirty() bool`: reports whether memory state is dirty
- `Scan() (DiskState, error)`: refreshes disk-state metadata without replacing cached content

### `Interfaces`

```go
type Node interface {
	Path() string
	Exists() bool
}
```

```go
type Composable interface {
	ComposePath(string)
}
```

```go
type Reporter interface {
	Record(Entry)
}
```

```go
type Loadable interface {
	Load() (bool, error)
	HasContent() bool
	Unload()
}
```

```go
type Discoverable interface {
	Discover() (DiskState, error)
}
```

```go
type Syncer interface {
	Sync(ctx Context) (ResultCode, error)
}
```

```go
type Scannable interface {
	Scan() (DiskState, error)
}
```

```go
type Validator interface {
	Validate(opts ValidateOptions) error
}
```

```go
type DeepEnsurer interface {
	EnsureDeep(ctx Context) (ResultCode, error)
}
```

```go
type DeepLoader interface {
	LoadDeep(ctx Context) (ResultCode, error)
}
```

```go
type DeepDiscoverer interface {
	DiscoverDeep(ctx Context) (ResultCode, error)
}
```

```go
type DeepSyncer interface {
	SyncDeep(ctx Context) (ResultCode, error)
}
```

```go
type DeepScanner interface {
	ScanDeep(ctx Context) (ResultCode, error)
}
```

```go
type DeepRenderer interface {
	RenderDeep() error
}
```

```go
type DeepDefaulter interface {
	DefaultDeep() error
}
```

```go
type DeepValidator interface {
	ValidateDeep(opts ValidateOptions) (ResultCode, error)
}
```

## Functions

### `NewDir`

```go
func NewDir(path string) Dir
```

Description:

- returns a standalone `Dir` handle for `path`

### `NewFile`

```go
func NewFile(path string) File
```

Description:

- returns a standalone `File` handle for `path`

### `NewExec`

```go
func NewExec(path string) Exec
```

Description:

- returns a standalone `Exec` handle for `path`

### `NewLink`

```go
func NewLink(path string) Link
```

Description:

- returns a standalone `Link` handle for `path`

### `NewSlot[T]`

```go
func NewSlot[T any](root Dir) Slot[T]
```

Description:

- returns a slot rooted at `root` with an empty cache

### `NewFileSlot[T]`

```go
func NewFileSlot[T any](root Dir) FileSlot[T]
```

Description:

- returns a file slot rooted at `root` with an empty cache

### `NewLinkSlot[T]`

```go
func NewLinkSlot[T LinkSlotItem](root Dir) LinkSlot[T]
```

Description:

- returns a link slot rooted at `root` with an empty cache

### `ComposeAs[T]`

```go
func ComposeAs[T any](root Dir) (T, error)
```

Description:

- composes a new value of `T` rooted at `root`

Notable behavior:

- `T` must be a struct or pointer to a struct that follows Conduit composition rules
- especially useful when creating slot-backed children lazily from an already composed `Dir`

### `ValidateDeep`

```go
func ValidateDeep(target any, opts ValidateOptions) (ResultCode, error)
```

Description:

- recursively validates an already composed or cached layout without mutating disk or memory state
