# Operations API

This page documents the exported `github.com/qlustra/conduit` manager, operations, policies, validation, and reporting types. Carrier-local APIs live in the [Layout API](layout.md).

## Structure and composition

```go
func Compile[T any]() (*Structure[T], error)
func (s *Structure[T]) Compose(root string) (*Layout[T], error)
func (l *Layout[T]) Tree() *T
```

`Compile[T]` recognizes the layout declaration using Sail. `Structure.Compose` creates a fresh tree, allocates declared pointer paths, resolves paths, and configures collection member composition. A compiled `Structure` can compose multiple independent layouts at different filesystem roots.

Declarations use `layout:"role"` and `path:"relative/path"`. Roles are `dir`, `file`, `exec`, `content`, `group`, `directories`, `documents`, and `links`. Only explicitly tagged fields participate; tag nested groups and embeddings deliberately. Paths must remain within the composition root. Carrier roles require complete native APIs; content and collection families retain their instantiated generic types. Collection roles also validate their zero-safe, state-independent `ChildKind()` against the role during `Compile`; member charts receive complete-carrier and role checks when prepared during composition.

Composition affects memory only. `Tree` returns the live tree managed by the layout, or nil before successful composition. Edit content and collection caches through this accessor.

## Layout operations

Operations use the cached Sail vessel to visit declared participants and typed live collection members. Local carrier hooks own their branches. Traversal stops at the first failure.

Ensure, Load, Discover, Scan, Sync, and Validate return a root `ResultCode` plus an error. The code summarizes the visited root rather than aggregating descendant outcomes; attach a reporter for per-path details. Default, Render, and Unload return an error.

### `Layout.Ensure`

```go
func (l *Layout[T]) Ensure(ctx Context) (ResultCode, error)
```

- prepares declared filesystem structure according to `ctx.EnsurePolicy`
- prepares each collection root and all cached directory member roots before entering that collection's member trees, including empty trees
- may leave other member roots created when a descendant later fails
- typed `Format` backing files use `EnsureSyncables`, independently of `EnsureFiles`
- leaves link entries to Sync, which requires a cached target
- visits cached members without discovering new names or persisting content
- reports per-path results through `ctx.Reporter`

### `Layout.Load`

```go
func (l *Layout[T]) Load(ctx Context) (ResultCode, error)
```

- enumerates physical collection members and composes them into the cache
- directory slots admit directories, file slots regular files, and link slots symlinks, including dangling links
- loads stateful content through local Load hooks; a missing file clears its cached content
- retains stale cached members but does not visit them during this pass
- performs no writes and reports through `ctx.Reporter`

### `Layout.Discover`

```go
func (l *Layout[T]) Discover(ctx Context) (ResultCode, error)
```

- enumerates and composes physical collection members with the same kind rules as Load
- refreshes disk-state metadata while preserving cached content and memory state
- uses local Discover hooks, falling back to Scan hooks
- retains stale cached members without visiting them
- performs no writes and reports through `ctx.Reporter`

### `Layout.Scan`

```go
func (l *Layout[T]) Scan(ctx Context) (ResultCode, error)
```

- refreshes disk-state metadata for declared nodes and cached members
- preserves cached content and memory state
- does not enumerate new collection names or modify the filesystem
- reports through `ctx.Reporter`

### `Layout.Sync`

```go
func (l *Layout[T]) Sync(ctx Context) (ResultCode, error)
```

- writes eligible cached content and link targets through local Sync hooks
- honors the carrier's sync, write, and path-safety policies
- reports through `ctx.Reporter`
- leaves discovery and scaffolding to separate Discover/Load and Ensure passes
- individual carrier writes may create their own parent directories

Use pointer collection members for carriers such as `Format` and `Link`. Sail rejects nonaddressable value members whose pointer-only hooks cannot be bound; pointers retain cached member identity and metadata updates.

### `Layout.Default`

```go
func (l *Layout[T]) Default() error
```

Calls `Default() error` on declared participants and cached members. Defaults stay in memory. The implementation decides which values count as unset; built-in setters such as `SetDefault`, `SetDefaultContext`, and `SetDefaultTarget` preserve existing values.

### `Layout.Render`

```go
func (l *Layout[T]) Render() error
```

Calls `Renderable.Render` and stores the result with `SetRendered`. If the node does not implement Renderable, it uses the Templatable `Template`/`RenderTemplate` path. Rendering visits composed and cached nodes and stores text in memory; Sync persists it separately.

### `Layout.Unload`

```go
func (l *Layout[T]) Unload() error
```

Calls local `Unload()` hooks on declared participants and cached members. It clears cached content while retaining collection membership and the last known disk state. It does not change the filesystem.

### `Layout.Validate`

```go
func (l *Layout[T]) Validate(opts ValidateOptions) (ResultCode, error)
```

Checks collection roots and calls local `Validate(opts) error` hooks on composed nodes and cached members. Built-in carriers apply `opts.PathSafetyPolicy`. Validation does not discover members or change content, cache membership, or the filesystem. Reports use `opts.Reporter`.

All operations require a successfully composed Layout. Result-returning operations return their Failed code plus an error otherwise; memory-only operations return an error.

## Variables

### `DefaultContext`

Type:

```go
var DefaultContext conduit.Context
```

Default value:

```go
conduit.Context{
	DirMode:          0o755,
	FileMode:         0o644,
	ExecMode:         0o755,
	EnsurePolicy:     conduit.EnsureAll,
	SyncPolicy:       conduit.SyncRewrite,
	PathSafetyPolicy: conduit.PathSafetyRejectSymlinkParents,
	WritePolicy:      conduit.WriteDirect,
}
```

Notable behavior:

- used as the default permission set in most examples
- `ExecMode` is applied by `Exec` operations
- `SyncPolicy` defaults to `SyncRewrite`
- `PathSafetyPolicy` defaults to rejecting symlink parents during mutating path resolution
- `WritePolicy` defaults to direct rewrites
- atomic writes stage temporary files in the system temp directory by default

## Types

### `Context`

```go
type Context struct {
	DirMode          os.FileMode
	FileMode         os.FileMode
	ExecMode         os.FileMode
	EnsurePolicy     conduit.EnsurePolicy
	SyncPolicy       conduit.SyncPolicy
	PathSafetyPolicy conduit.PathSafetyPolicy
	WritePolicy       conduit.WritePolicy
	TempFilePlacement conduit.TempFilePlacement
	TempDir           conduit.Dir
	Reporter         conduit.Reporter
}
```

Fields:

- `DirMode`: mode used when creating directories
- `FileMode`: mode used when creating regular files
- `ExecMode`: mode used when creating or ensuring `Exec` files
- `EnsurePolicy`: selects which node kinds `Ensure` may materialize
- `SyncPolicy`: selects which stateful memory states `Sync` may write, with optional disk-state filters
- `PathSafetyPolicy`: controls whether mutating typed filesystem operations reject symlink parents during path resolution
- `WritePolicy`: selects direct rewrites or per-file atomic replacement for `File.WriteBytes`
- `TempFilePlacement`: selects where atomic writes create their temporary file
- `TempDir`: custom staging directory used when `TempFilePlacement` is `TempFileDir`
- `Reporter`: optional sink for per-path tree-operation results

Notable behavior:

- when `ExecMode` is zero, `Exec` falls back to `FileMode` and adds execute bits automatically
- when `EnsurePolicy` is zero, ensure operations fall back to `EnsureAll`
- when `SyncPolicy` has no memory-state bits, sync operations fall back to `SyncRewrite`
- when `SyncPolicy` has no disk-state bits, sync operations do not restrict by disk state
- when `PathSafetyPolicy` is zero, mutating typed filesystem operations reject symlink parents
- when `WritePolicy` is zero, `File.WriteBytes` rewrites directly
- when `TempFilePlacement` is zero, atomic writes use the system temp directory
- when `Reporter` is nil, tree operations do not collect traversal reports

### `EnsurePolicy`

```go
type EnsurePolicy uint8
```

Description:

- bitmask policy that filters which node kinds `Ensure` may materialize

Constants:

- `EnsureDirs`: include raw directories
- `EnsureFiles`: include raw files
- `EnsureExecs`: include executable files
- `EnsureSyncables`: include syncable stateful wrappers such as typed files
- `EnsureAll`: include all supported ensure kinds
- `EnsureScaffold`: raw `Dir`, `File`, and `Exec` scaffolding only
- `EnsureNone`: explicit no-op ensure policy

Methods:

- `Allow(bits EnsurePolicy) EnsurePolicy`: enables public ensure-policy bits
- `Deny(bits EnsurePolicy) EnsurePolicy`: disables public ensure-policy bits
- `Has(bits EnsurePolicy) bool`: reports whether all requested public bits are enabled

### `PathSafetyPolicy`

```go
type PathSafetyPolicy uint8
```

Description:

- policy that controls how mutating typed filesystem operations treat symlinks encountered while resolving destination paths

Constants:

- `PathSafetyRejectSymlinkParents`: reject existing symlink parents during mutating path resolution
- `PathSafetyFollowSymlinks`: preserve path-following behavior for mutating operations

### `WritePolicy`

```go
type WritePolicy uint8
```

Description:

- controls how `File.WriteBytes`, typed file `Save`, and sync operations replace destination file content

Constants:

- `WriteDirect`: rewrite the destination path directly
- `WriteAtomicReplace`: write to a temporary file, then atomically replace the destination when the filesystem can guarantee an atomic rename

### `TempFilePlacement`

```go
type TempFilePlacement uint8
```

Description:

- controls where atomic writes stage their temporary file

Constants:

- `TempFileSystem`: create temporary files in the system temp directory
- `TempFileDir`: create temporary files in `Context.TempDir`
- `TempFileAdjacent`: create temporary files next to the destination file

`TempFileSystem` avoids watcher noise near destination paths. If the staged temp
file cannot be atomically renamed over the destination, the operation fails with
corrective actions instead of falling back to adjacent temp files.

### `OpenPolicy`

```go
type OpenPolicy uint8
```

Description:

- root-package alias for `layout.OpenPolicy`, used by `layout.File` open helpers

Constants:

- `OpenExisting`: add no creation flags
- `OpenOrCreate`: add `os.O_CREATE`
- `OpenOrCreateExclusive`: add `os.O_CREATE | os.O_EXCL`

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

- in-memory collector for operation entries recorded during tree traversal

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

- one reported path-level outcome for a tree operation

Methods:

- `IsError() bool`: reports whether the entry carries an error
- `IsSkipped() bool`: reports whether the result represents a visited-but-not-applied outcome
- `IsSuccess() bool`: reports whether the entry completed without error
- `ResultName() string`: returns a stable lowercase result name relative to `Op`

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

Methods:

- `String() string`: returns the lowercase operation name used in reports

### `ResultCode`

```go
type ResultCode uint8
```

Description:

- operation-specific outcome code returned by tree operations and recorded in reports
- interpret values relative to the operation that produced them

Constants:

- ensure results: `EnsureEnsured`, `EnsureSkippedPolicy`, `EnsureFailed`
- load results: `LoadLoaded`, `LoadMissing`, `LoadTraversed`, `LoadNotApplicable`, `LoadFailed`
- discover results: `DiscoverPresent`, `DiscoverMissing`, `DiscoverTraversed`, `DiscoverNotApplicable`, `DiscoverFailed`
- scan results: `ScanPresent`, `ScanMissing`, `ScanTraversed`, `ScanNotApplicable`, `ScanFailed`
- sync results: `SyncWritten`, `SyncTraversed`, `SyncNotApplicable`, `SyncSkippedNoContent`, `SyncSkippedPolicy`, `SyncFailed`
- validate results: `ValidateOK`, `ValidateTraversed`, `ValidateNotApplicable`, `ValidateFailed`

### `SyncPolicy`

```go
type SyncPolicy uint8
```

Description:

- bitmask policy that filters which stateful memory states are writable during `Sync`, with optional disk-state gates

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
