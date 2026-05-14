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
	DirMode:      0o755,
	FileMode:     0o644,
	ExecMode:     0o755,
	EnsurePolicy: conduit.EnsureAll,
	SyncPolicy:   conduit.SyncRewrite,
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
	DirMode      os.FileMode
	FileMode     os.FileMode
	ExecMode     os.FileMode
	EnsurePolicy conduit.EnsurePolicy
	SyncPolicy   conduit.SyncPolicy
	Reporter     conduit.Reporter
}
```

Fields:

- `DirMode`: mode used when creating directories
- `FileMode`: mode used when creating regular files
- `ExecMode`: mode used when creating or ensuring `Exec` files
- `EnsurePolicy`: selects which node kinds `Ensure` and `EnsureDeep` may materialize
- `SyncPolicy`: selects which typed memory states `Sync` and `SyncDeep` may write, with optional disk-state filters
- `Reporter`: optional sink for per-path deep-operation results

Notable behavior:

- when `ExecMode` is zero, `Exec` falls back to `FileMode` and adds execute bits automatically
- when `EnsurePolicy` is zero, ensure operations fall back to `EnsureAll`
- when `SyncPolicy` has no memory-state bits, sync operations fall back to `SyncRewrite`
- when `SyncPolicy` has no disk-state bits, sync operations do not restrict by disk state
- when `Reporter` is nil, deep operations do not collect traversal reports

### `EnsurePolicy`

```go
type EnsurePolicy uint8
```

Description:

- bitmask policy that filters which node kinds `Ensure` and `EnsureDeep` may materialize

Constants:

- `EnsureDirs`: include raw directories
- `EnsureFiles`: include raw files
- `EnsureExecs`: include executable files
- `EnsureSyncables`: include syncable stateful wrappers such as typed files
- `EnsureAll`: historical ensure behavior
- `EnsureScaffold`: raw `Dir`, `File`, and `Exec` scaffolding only
- `EnsureNone`: explicit no-op ensure policy

### `ValidateOptions`

```go
type ValidateOptions struct {
	Reporter conduit.Reporter
	PathSafetyPolicy conduit.PathSafetyPolicy
}
```

Fields:

- `Reporter`: optional sink for per-path validation results
- `PathSafetyPolicy`: controls whether built-in typed filesystem nodes reject symlink parents during validation

Notable behavior:

- the zero value is ready to use
- validation reporting is separate from `Context` because validation does not need file modes or sync policy
- the zero-value path policy is `PathSafetyRejectSymlinkParents`

### `Reporter`

```go
type Reporter interface {
	Record(conduit.Entry)
}
```

Description:

- optional sink used by `Context.Reporter` and `ValidateOptions.Reporter` for
  path-level reporting
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

### `ResultCode`

```go
type ResultCode uint8
```

Description:

- operation-specific outcome code returned by deep operations and recorded in reports
- interpret values relative to the operation that produced them

Notable validate results:

- `EnsureSkippedPolicy`
- `ValidateOK`
- `ValidateTraversed`
- `ValidateNotApplicable`
- `ValidateFailed`

### `SyncPolicy`

```go
type SyncPolicy uint8
```

Description:

- bitmask policy that filters which typed memory states are writable during `Sync` and `SyncDeep`, with optional disk-state gates

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
- layout tags must stay within the original compose root; absolute tags and escaping relative tags return an error

### `EnsureDeep`

```go
func EnsureDeep(target any, ctx Context) (conduit.ResultCode, error)
```

Description:

- recursively materializes declared structure on disk

Arguments:

- `target`: composed struct or node tree
- `ctx`: directory and file permission settings

Returns:

- `ResultCode`: semantic ensure outcome for the visited root
- `error`: first failure encountered during traversal

Notable behavior:

- creates `layout.Dir`, `layout.File`, and `layout.Exec` nodes as required by the layout
- ensures syncable stateful wrappers only when `ctx.EnsurePolicy` includes them
- records `EnsureSkippedPolicy` for visited nodes skipped by `ctx.EnsurePolicy`
- only ensures cached slot items such as `layout.Slot[T]`, `layout.FileSlot[T]`, and `layout.LinkSlot[T]`
- does not load typed file content
- does not discover new slot items from disk
- returns an error if `target` is nil

### `LoadDeep`

```go
func LoadDeep(target any, ctx Context) (conduit.ResultCode, error)
```

Description:

- recursively loads typed content from disk into memory

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep loaders

Returns:

- `ResultCode`: semantic load outcome for the visited root
- `error`: first failure encountered during traversal

Notable behavior:

- loads `layout.Format`-backed files such as `formats.JSONFile[T]`, `formats.YAMLFile[T]`, and `formats.TOMLFile[T]`
- discovers slot-backed entries from disk according to slot kind
- does not create missing files
- leaves uncached missing slot entries undiscovered until they exist on disk
- returns an error if `target` is nil

### `DiscoverDeep`

```go
func DiscoverDeep(target any, ctx Context) (conduit.ResultCode, error)
```

Description:

- recursively discovers slot-backed structure from disk without loading typed file content

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep discoverers

Returns:

- `ResultCode`: semantic discover outcome for the visited root
- `error`: first failure encountered during traversal

Notable behavior:

- discovers slot-backed entries from disk according to slot kind
- composes discovered children recursively
- updates typed-file disk state without loading bytes into memory
- preserves existing in-memory values and memory state
- does not create missing files
- returns an error if `target` is nil

### `SyncDeep`

```go
func SyncDeep(target any, ctx Context) (conduit.ResultCode, error)
```

Description:

- recursively writes sync-eligible typed in-memory content back to disk

Arguments:

- `target`: composed struct or node tree
- `ctx`: directory and file permission settings

Returns:

- `ResultCode`: semantic sync outcome for the visited root
- `error`: first failure encountered during traversal

Notable behavior:

- only writes typed files that currently have content loaded in memory
- applies `ctx.SyncPolicy` to typed memory state and optional disk-state filters before writing
- only syncs cached slot items
- `layout.Slot[T]`, `layout.FileSlot[T]`, and `layout.LinkSlot[T]` ensure cached children before syncing them, and that ensure pass respects `ctx.EnsurePolicy`
- does not materialize standalone raw `layout.Dir` or `layout.File` nodes
- does not delete files or directories
- returns an error if `target` is nil

### `ScanDeep`

```go
func ScanDeep(target any, ctx Context) (conduit.ResultCode, error)
```

Description:

- recursively refreshes disk-presence metadata for composed items

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep scanners

Returns:

- `ResultCode`: semantic scan outcome for the visited root
- `error`: first failure encountered during traversal

Notable behavior:

- updates disk state without loading file content
- preserves current in-memory values and memory state
- only scans cached slot items
- does not discover new slot entries from disk
- returns an error if `target` is nil

### `ValidateDeep`

```go
func ValidateDeep(target any, opts ValidateOptions) (conduit.ResultCode, error)
```

Description:

- recursively validates an already composed or cached layout without mutating disk or memory state

Arguments:

- `target`: composed struct or node tree
- `opts`: validation reporting options

Returns:

- `ResultCode`: semantic validation outcome for the visited root
- `error`: first failure encountered during traversal

Notable behavior:

- calls `Validate(opts) error` on nodes that implement `layout.Validator`
- calls `ValidateDeep(opts)` on nodes that implement `layout.DeepValidator`
- only visits already composed or cached children
- does not discover new slot entries from disk
- does not read from disk into typed memory
- does not render templates or write to disk
- built-in `File`, `Dir`, `Exec`, and link nodes apply `opts.PathSafetyPolicy` during validation
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
