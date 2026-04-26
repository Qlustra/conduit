# Operations API

This page documents the exported package-level operations and configuration types.

## Variables

### `DefaultContext`

Type:

```go
var DefaultContext conduit.Context
```

Default value:

```go
conduit.Context{
	DirMode:    0o755,
	FileMode:   0o644,
	ExecMode:   0o755,
	SyncPolicy: conduit.SyncRewrite,
}
```

Notable behavior:

- used as the default permission set in most examples
- `ExecMode` is applied by `Exec` operations
- `SyncPolicy` defaults to `SyncRewrite`

## Types

### `Context`

```go
type Context struct {
	DirMode    os.FileMode
	FileMode   os.FileMode
	ExecMode   os.FileMode
	SyncPolicy conduit.SyncPolicy
}
```

Fields:

- `DirMode`: mode used when creating directories
- `FileMode`: mode used when creating regular files
- `ExecMode`: mode used when creating or ensuring `Exec` files
- `SyncPolicy`: selects which typed memory states `Sync` and `SyncDeep` may write

Notable behavior:

- when `ExecMode` is zero, `Exec` falls back to `FileMode` and adds execute bits automatically
- when `SyncPolicy` is zero, sync operations fall back to `SyncRewrite`

### `SyncPolicy`

```go
type SyncPolicy uint8
```

Description:

- bitmask policy that filters which typed memory states are writable during `Sync` and `SyncDeep`

Constants:

- `SyncOnLoaded`: include `MemoryLoaded`
- `SyncOnSynced`: include `MemorySynced`
- `SyncOnDirty`: include `MemoryDirty`
- `SyncRewrite`: include loaded, synced, and dirty states
- `SyncIfDirty`: include only dirty state
- `SyncIfUnsynced`: include loaded and dirty states

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
- `DefaultDeep` calls `Default()` on matching nodes
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
- `RenderDeep` calls `Render()` and passes the result into `SetRendered(string)`
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
- `SetRendered(string)` stores the rendered text into the target's in-memory file state
- `RenderDeep` uses this path only when the node does not implement `Renderable`

## Functions

### `Compose`

```go
func Compose(root string, target any) error
```

Description:

- binds a layout struct to a filesystem root by assigning resolved paths to tagged fields

Arguments:

- `root`: base path for the layout
- `target`: non-nil pointer to a struct

Notable behavior:

- does not touch the filesystem
- returns an error if `target` is nil, not a pointer, or does not point to a struct
- allocates tagged pointer-to-struct fields as needed
- only fields tagged with `layout:"..."` are composed

### `EnsureDeep`

```go
func EnsureDeep(target any, ctx Context) error
```

Description:

- recursively materializes declared structure on disk

Arguments:

- `target`: composed struct or node tree
- `ctx`: directory and file permission settings

Notable behavior:

- creates directories, regular files, and executable files as required by the layout
- only ensures cached `Slot[T]` items
- does not load typed file content
- does not discover new slot items from disk
- returns an error if `target` is nil

### `LoadDeep`

```go
func LoadDeep(target any, ctx Context) error
```

Description:

- recursively loads typed content from disk into memory

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep loaders

Notable behavior:

- loads `Format`-backed files such as `JSONFile[T]`, `YAMLFile[T]`, and `TOMLFile[T]`
- discovers `Slot[T]` entries by reading child directories from disk
- does not create missing files
- leaves uncached missing slot entries undiscovered until they exist on disk
- returns an error if `target` is nil

### `DiscoverDeep`

```go
func DiscoverDeep(target any, ctx Context) error
```

Description:

- recursively discovers slot-backed structure from disk without loading typed file content

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep discoverers

Notable behavior:

- discovers `Slot[T]` entries by reading child directories from disk
- composes discovered children recursively
- updates typed-file disk state without loading bytes into memory
- preserves existing in-memory values and memory state
- does not create missing files
- returns an error if `target` is nil

### `SyncDeep`

```go
func SyncDeep(target any, ctx Context) error
```

Description:

- recursively writes sync-eligible typed in-memory content back to disk

Arguments:

- `target`: composed struct or node tree
- `ctx`: directory and file permission settings

Notable behavior:

- only writes typed files that currently have content loaded in memory
- applies `ctx.SyncPolicy` to typed memory state before writing
- only syncs cached `Slot[T]` items
- `Slot[T]` ensures cached children before syncing them
- does not materialize standalone raw `Dir` or `File` nodes
- does not delete files or directories
- returns an error if `target` is nil

### `ScanDeep`

```go
func ScanDeep(target any, ctx Context) error
```

Description:

- recursively refreshes disk-presence metadata for composed items

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep scanners

Notable behavior:

- updates disk state without loading file content
- preserves current in-memory values and memory state
- only scans cached `Slot[T]` items
- does not discover new slot entries from disk
- returns an error if `target` is nil

### `RenderDeep`

```go
func RenderDeep(target any) error
```

Description:

- recursively renders text-template wrappers into in-memory file content

Arguments:

- `target`: composed struct or node tree

Notable behavior:

- calls `Render() (string, error)` on nodes that implement `Renderable`
- otherwise, calls `Template()` and `RenderTemplate(...)` on nodes that implement `Templatable`
- stores rendered text in memory via `SetRendered(string)`
- only visits cached `Slot[T]` items
- does not discover new slot entries from disk
- does not write to disk; pair it with `SyncDeep` to persist rendered content
- `Renderable` takes precedence over `Templatable` when a type implements both
- returns an error if `target` is nil

### `DefaultDeep`

```go
func DefaultDeep(target any) error
```

Description:

- recursively applies defaults to already composed or cached items in memory

Arguments:

- `target`: composed struct or node tree

Notable behavior:

- calls `Default() error` on nodes that implement `Defaulter`
- only visits cached `Slot[T]` items
- does not discover new slot entries from disk
- does not read from disk
- does not write to disk; pair it with `RenderDeep` or `SyncDeep` as needed
- returns an error if `target` is nil
