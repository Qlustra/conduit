# Layout API

This page documents the exported layout-oriented types.

## Types

### `Dir`

```go
type Dir struct{}
```

Description:

- stateless handle to a directory path

Methods:

- `Path() string`: returns the bound path
- `Exists() bool`: reports whether the path currently exists
- `Join(parts ...string) string`: joins descendant path segments onto the directory path
- `Dir(name string) Dir`: returns a child directory handle
- `File(name string) File`: returns a child file handle
- `DeleteIfExists() error`: removes the directory tree when it exists
- `Ensure(ctx Context) error`: creates the directory tree using `ctx.DirMode`

Notable behavior:

- `DeleteIfExists` uses recursive removal
- `Exists` only checks current filesystem state; it does not validate that the path is a directory

### `File`

```go
type File struct{}
```

Description:

- stateless handle to a regular file path

Methods:

- `Path() string`: returns the bound path
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

### `Exec`

```go
type Exec struct{}
```

Description:

- executable file handle with process-launch helpers

Methods:

- `Path() string`
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

### `Slot[T]`

```go
type Slot[T any] struct{}
```

Description:

- keyed container for repeated child layouts rooted under one directory

Methods:

- `Path() string`: returns the slot root path
- `Exists() bool`: reports whether the slot root exists on disk
- `Root() Dir`: returns the slot root as a `Dir`
- `Has(name string) bool`: reports whether a named child directory exists on disk
- `Get(name string) (T, bool)`: returns a cached item only
- `Put(name string, item T)`: inserts or replaces a cached item
- `Remove(name string)`: removes a cached item
- `Clear()`: clears the cache
- `Keys() []string`: returns sorted cached keys
- `At(name string) (T, error)`: returns a cached item or composes and caches one lazily
- `MustAt(name string) T`: panicking form of `At`
- `Add(name string, ctx Context) (T, error)`: creates the child root on disk, composes the item, ensures its structure, and caches it
- `Require(name string) (T, error)`: returns an item only when the child root already exists on disk
- `Ensure(ctx Context) error`: ensures the slot root directory
- `EnsureDeep(ctx Context) error`: ensures the slot root and all cached items
- `DiscoverDeep(ctx Context) error`: discovers child directories on disk and scans them without loading typed content
- `LoadDeep(ctx Context) error`: discovers child directories on disk and loads them
- `ScanDeep(ctx Context) error`: scans only cached items
- `SyncDeep(ctx Context) error`: ensures cached items, then syncs typed content within those items

Notable behavior:

- `At` composes items relative to `slotRoot/<name>` and caches them
- `Add` ensures the child root and calls `EnsureDeep` on the new child
- `Keys` is cache-based; it does not list the filesystem directly
- `DiscoverDeep` discovers directory-backed entries from disk without loading typed files
- `LoadDeep` discovers directory-backed entries from disk
- `ScanDeep` and `SyncDeep` do not discover uncached entries
- `Slot.SyncDeep` ensures cached children before syncing them
