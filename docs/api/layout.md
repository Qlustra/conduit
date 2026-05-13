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
- `RelTo(base Pather) (string, error)`: returns the path relative to another node with a `Path()`
- `JoinRelTo(base Pather, parts ...string) (string, error)`: joins path parts onto the relative path from another node
- `RelToPath(base string) (string, error)`: returns the path relative to a raw base path
- `JoinRelToPath(base string, parts ...string) (string, error)`: joins path parts onto the relative path from a raw base path
- `DeclaredPath() (string, bool)`: returns the node's own declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the handle belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the composed-relative path
- `Exists() bool`: reports whether the path currently exists
- `Join(parts ...string) string`: joins descendant path segments onto the directory path
- `Dir(name string) Dir`: returns a child directory handle
- `File(name string) File`: returns a child file handle
- `DeleteIfExists() error`: removes the directory tree when it exists
- `Ensure(ctx Context) error`: creates the directory tree using `ctx.DirMode`

Notable behavior:

- `DeleteIfExists` uses recursive removal
- `Exists` only checks current filesystem state; it does not validate that the path is a directory
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
- `RelTo(base Pather) (string, error)`: returns the path relative to another node with a `Path()`
- `JoinRelTo(base Pather, parts ...string) (string, error)`: joins path parts onto the relative path from another node
- `RelToPath(base string) (string, error)`: returns the path relative to a raw base path
- `JoinRelToPath(base string, parts ...string) (string, error)`: joins path parts onto the relative path from a raw base path
- `DeclaredPath() (string, bool)`: returns the node's own declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the handle belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the composed-relative path
- `Exists() bool`: reports whether the path currently exists
- `WriteBytes(data []byte, dirMode os.FileMode, fileMode os.FileMode) error`: creates parent directories and writes raw bytes
- `ReadBytes() ([]byte, error)`: reads the file contents
- `ReadBytesIfExists() ([]byte, bool, error)`: reads the file if present and returns `ok == false` for missing files
- `DeleteIfExists() error`: removes the file when it exists
- `Ensure(ctx Context) error`: creates the parent directories and creates the file if missing

Notable behavior:

- `Ensure` uses `os.O_CREATE` and does not truncate existing file contents
- `WriteBytes` always rewrites the file contents
- `Exists` only checks that some filesystem entry exists at the path
- the declared-path helpers return `ok == false` when the handle was not attached through `Compose`
- the composed-path helpers return `ok == false` when the handle was not attached through `Compose`
- dotfiles such as `.env` report an empty extension and keep the full basename as the stem

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
- `RelTo(base Pather) (string, error)`
- `JoinRelTo(base Pather, parts ...string) (string, error)`
- `RelToPath(base string) (string, error)`
- `JoinRelToPath(base string, parts ...string) (string, error)`
- `DeclaredPath() (string, bool)`
- `JoinDeclaredPath(parts ...string) (string, bool)`
- `ComposedBaseDir() (Dir, bool)`
- `ComposedRelativePath() (string, bool)`
- `JoinComposedPath(parts ...string) (string, bool)`
- `Exists() bool`
- `ReadBytes() ([]byte, error)`
- `ReadBytesIfExists() ([]byte, bool, error)`
- `WriteBytes(data []byte, dirMode os.FileMode, fileMode os.FileMode) error`
- `DeleteIfExists() error`
- `Ensure(ctx Context) error`: creates the file and ensures executable permissions
- `EnsureExecutable(ctx Context) error`: same executable materialization behavior as `Ensure`
- `IsExecutable() bool`: reports whether the current target is an executable regular file
- `Command(ctx context.Context, opts RunOptions) *exec.Cmd`: builds an `exec.Cmd`
- `Run(ctx context.Context, opts RunOptions) error`: runs the executable
- `Output(ctx context.Context, opts RunOptions) ([]byte, error)`: runs and captures stdout
- `CombinedOutput(ctx context.Context, opts RunOptions) ([]byte, error)`: runs and captures combined stdout and stderr

Notable behavior:

- `Ensure` and `EnsureExecutable` apply `Context.ExecMode`, or `FileMode` with execute bits added when `ExecMode` is zero
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
- `DeclaredPath() (string, bool)`: returns the node's own declared layout fragment
- `JoinDeclaredPath(parts ...string) (string, bool)`: joins path parts onto the declared layout fragment
- `ComposedBaseDir() (Dir, bool)`: returns the compose base directory when the handle belongs to a composed tree
- `ComposedRelativePath() (string, bool)`: returns the path relative to the compose base directory
- `JoinComposedPath(parts ...string) (string, bool)`: joins path parts onto the composed-relative path
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
- `Delete() error`: removes the symlink when it exists
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
- `Exists() bool`: reports whether the slot root exists on disk
- `Root() Dir`: returns the slot root as a `Dir`
- `Len() int`: returns the number of cached items
- `Has(name string) bool`: reports whether a named child directory exists on disk
- `Get(name string) (T, bool)`: returns a cached item only
- `Put(name string, item T)`: inserts or replaces a cached item
- `Remove(name string)`: removes a cached item
- `Delete(name string) error`: removes the child tree from disk if present and evicts the cached item
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
- `Slot.SyncDeep` ensures cached children before syncing them

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
- `Exists() bool`: reports whether the slot root exists on disk
- `Root() Dir`: returns the slot root as a `Dir`
- `Len() int`: returns the number of cached items
- `Has(name string) bool`: reports whether a named child file exists on disk
- `Get(name string) (T, bool)`: returns a cached item only
- `Put(name string, item T)`: inserts or replaces a cached item
- `Remove(name string)`: removes a cached item
- `Delete(name string) error`: removes the child file from disk if present and evicts the cached item
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

Notable behavior:

- `At` composes items relative to `slotRoot/<name>` and caches them
- `Add` ensures the slot root and calls `EnsureDeep` on the item
- `Delete` removes both the on-disk child file and the cached entry
- `Len`, `Entries`, `All`, and `Keys` are cache-based; they do not list the filesystem directly
- `Entries` and `All` return cached items as-is, preserving pointer or value semantics chosen by `T`
- the declared-path helpers delegate to the slot root and expose the slot field's own declared fragment
- the composed-path helpers delegate to the slot root and return `ok == false` until the slot has been attached through `Compose`
- `DiscoverDeep` and `LoadDeep` discover file-backed entries from disk and ignore subdirectories
- `ScanDeep` and `SyncDeep` do not discover uncached entries

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
