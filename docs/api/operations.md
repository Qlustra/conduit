# Operations API

This page documents the exported `github.com/qlustra/conduit` package-level operations and configuration types.

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
	Reporter   conduit.Reporter
}
```

Fields:

- `DirMode`: mode used when creating directories
- `FileMode`: mode used when creating regular files
- `ExecMode`: mode used when creating or ensuring `Exec` files
- `SyncPolicy`: selects which typed memory states `Sync` and `SyncDeep` may write
- `Reporter`: optional sink for per-path deep-operation results

Notable behavior:

- when `ExecMode` is zero, `Exec` falls back to `FileMode` and adds execute bits automatically
- when `SyncPolicy` is zero, sync operations fall back to `SyncRewrite`
- when `Reporter` is nil, deep operations do not collect traversal reports

### `Reporter`

```go
type Reporter interface {
	Record(conduit.Entry)
}
```

Description:

- optional sink carried on `Context` for deep-operation reporting
- built-in `conduit.Report` implements this interface

### `Report`

```go
type Report struct { ... }
```

Description:

- in-memory collector for operation entries recorded during deep traversal

Notable methods:

- `Record(Entry)`
- `Entries() []Entry`
- `Len() int`
- `HasErrors() bool`
- `Filter(func(Entry) bool) []Entry`
- `Sort(func(Entry, Entry) bool)`
- `SortByPath()`
- `RenderTree() string`

### `Entry`

```go
type Entry struct {
	Op     conduit.Operation
	Path   string
	Result conduit.ResultCode
	Err    error
}
```

Description:

- one reported path-level outcome for a deep operation

Notable behavior:

- `Result` is interpreted relative to `Op`
- `Err` is populated on failures

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

- creates `layout.Dir`, `layout.File`, and `layout.Exec` nodes as required by the layout
- only ensures cached `layout.Slot[T]` items
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

- loads `layout.Format`-backed files such as `formats.JSONFile[T]`, `formats.YAMLFile[T]`, and `formats.TOMLFile[T]`
- discovers `layout.Slot[T]` entries by reading child directories from disk
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

- discovers `layout.Slot[T]` entries by reading child directories from disk
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
- only syncs cached `layout.Slot[T]` items
- `layout.Slot[T]` ensures cached children before syncing them
- does not materialize standalone raw `layout.Dir` or `layout.File` nodes
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
- only scans cached `layout.Slot[T]` items
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

- calls `Render() (string, error)` on nodes that implement `layout.Renderable`
- otherwise, calls `Template()` and `RenderTemplate(...)` on nodes that implement `layout.Templatable`
- stores rendered text in memory via `SetRendered(string)`
- only visits cached `layout.Slot[T]` items
- does not discover new slot entries from disk
- does not write to disk; pair it with `SyncDeep` to persist rendered content
- `layout.Renderable` takes precedence over `layout.Templatable` when a type implements both
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

- calls `Default() error` on nodes that implement `layout.Defaulter`
- only visits cached `layout.Slot[T]` items
- does not discover new slot entries from disk
- does not read from disk
- does not write to disk; pair it with `RenderDeep` or `SyncDeep` as needed
- returns an error if `target` is nil
