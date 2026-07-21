# Operations usage

Conduit separates layout declaration from filesystem operations. You compose a layout once, then explicitly choose how state moves between disk and memory.

## Compose

`Compose(root, target)` binds a layout to a real filesystem root:

```go
var ws Workspace
err := conduit.Compose("/srv/workspace", &ws)
```

Composition does not touch the filesystem. It only assigns paths to the declared nodes.

## Ensure

`EnsureDeep(target, ctx)` materializes the declared structure on disk:

```go
_, err := conduit.EnsureDeep(&ws, conduit.DefaultContext)
```

What it does:

- creates directories declared by `Dir`
- creates files declared by `File`
- creates executable files declared by `Exec`
- creates backing files for syncable stateful nodes when `ctx.EnsurePolicy` includes them
- ensures already cached slot items

What it does not do:

- load stateful content into memory
- discover new slot entries from disk
- delete anything

For slot types such as `Slot[T]`, `FileSlot[T]`, and `LinkSlot[T]`, only cached items are ensured. Use `slot.Add(name, ctx)` when you want to create a new dynamic child explicitly.

`ctx.EnsurePolicy` lets you narrow the pass. For example:

- `conduit.EnsureAll` preserves the historical behavior
- `conduit.EnsureScaffold` materializes raw `Dir`, `File`, and `Exec` nodes but skips syncable stateful wrappers
- `conduit.EnsureDirs | conduit.EnsureSyncables` creates directories and syncable backing files, but skips raw standalone files

## Default

`DefaultDeep(target)` applies in-memory defaults without consulting disk:

```go
err := conduit.DefaultDeep(&ws)
```

What it does:

- calls `Default() error` on nodes that implement it
- seeds only already composed or cached children
- preserves existing in-memory values when wrappers use `SetDefault(...)`, `SetDefaultContext(...)`, or `SetDefaultTarget(...)`

What it does not do:

- read from disk
- discover new slot entries from disk
- render templates
- write anything back to disk

This makes `DefaultDeep` the in-memory seeding phase that usually sits before `RenderDeep`, `ValidateDeep`, or `SyncDeep`.

## Load

`LoadDeep(target, ctx)` reads filesystem content into the in-memory model:

```go
_, err := conduit.LoadDeep(&ws, conduit.DefaultContext)
```

What it does:

- loads stateful nodes such as `layout.Link`, `layout.TextTemplate[C]`, and format-backed files
- discovers slot, file-slot, and link-slot entries from disk according to slot kind
- composes and loads discovered slot items recursively

What it does not do:

- create missing files
- write anything back to disk
- remove cached slot items that no longer exist

For stateful nodes, a missing load marks disk state missing and clears the cached in-memory value or link target.

## Discover

`DiscoverDeep(target, ctx)` discovers the declared layout from disk without loading stateful content:

```go
_, err := conduit.DiscoverDeep(&ws, conduit.DefaultContext)
```

What it does:

- discovers slot, file-slot, and link-slot entries from disk according to slot kind
- composes discovered slot items recursively
- updates disk state for stateful nodes such as links, text templates, and format-backed files
- preserves the current in-memory content, target, and memory state

What it does not do:

- load file content into memory
- create missing files
- write anything back to disk

This makes `DiscoverDeep` the middle ground between `LoadDeep` and `ScanDeep`: it discovers structure like `LoadDeep`, but it only observes stateful nodes like `ScanDeep`.

## Render

`RenderDeep(target)` renders derived text templates into cached file content:

```go
err := conduit.RenderDeep(&ws)
```

What it does:

- calls `Render() (string, error)` on nodes that implement `layout.Renderable`
- otherwise, calls `Template()` and `RenderTemplate(...)` on nodes that implement `layout.Templatable`
- stores rendered text in memory via `SetRendered(string)`
- visits only already composed or cached children

What it does not do:

- discover new slot entries from disk
- create missing files
- write anything back to disk

This makes `RenderDeep` the derive-in-memory phase that usually sits after `DefaultDeep` and before `ValidateDeep` or `SyncDeep`.

## Sync

`SyncDeep(target, ctx)` writes sync-eligible in-memory state back to disk:

```go
_, err := conduit.SyncDeep(&ws, conduit.DefaultContext)
```

What it does:

- writes sync-eligible in-memory state for format-backed files, text templates, and links
- syncs already cached slot items recursively
- runs the slot, file-slot, and link-slot preparation ensure phase under the same `ctx.EnsurePolicy`
- allows callers to choose rewrite behavior per sync pass

What it does not do:

- materialize standalone raw `Dir` or `File` fields
- invent uncached slot entries
- delete files or directories that are missing from memory
- merge disk content with memory content

For stateful nodes, `Sync` returns a skipped result when no syncable value is cached or when the current memory state is excluded by `ctx.SyncPolicy`.

## Scan

`ScanDeep(target, ctx)` refreshes disk-presence metadata for already composed items:

```go
_, err := conduit.ScanDeep(&ws, conduit.DefaultContext)
```

What it does:

- updates disk state for stateful nodes such as links, text templates, and format-backed files
- preserves the current in-memory content, target, and memory state
- scans cached slot items recursively

What it does not do:

- load file content
- discover new slot entries from disk
- modify files on disk

This makes `ScanDeep` useful for "is it there?" checks, not discovery.

## Validate

`ValidateDeep(target, opts)` validates the already composed or cached layout without mutating it:

```go
_, err := conduit.ValidateDeep(&ws, conduit.ValidateOptions{})
```

What it does:

- calls `Validate(opts) error` on nodes that implement it
- calls `ValidateDeep(opts)` on nodes that own their own validation traversal
- validates only already composed or cached children
- optionally records per-path validation results through `opts.Reporter`
- applies `opts.PathSafetyPolicy` when validating built-in filesystem node types

What it does not do:

- create missing files or directories
- discover new slot entries from disk
- load stateful content into memory
- render templates
- write anything back to disk

This makes `ValidateDeep` the semantic-check phase that can sit between load or render and sync.

Every deep operation returns `(ResultCode, error)`.

- `ResultCode` summarizes what happened at the visited root.
- `error` preserves the existing fail-fast traversal behavior.
- when `Context.Reporter` is set, per-path details are still collected into the report.

Inspect a root-level result when you need it:

```go
result, err := conduit.SyncDeep(&ws, conduit.DefaultContext)
if err != nil {
	return err
}
if result == conduit.SyncSkippedPolicy {
	// the root was visited, but nothing was written
}
```

## Context

Every filesystem operation accepts a `Context`:

```go
ctx := conduit.Context{
	DirMode:          0o755,
	FileMode:         0o644,
	ExecMode:         0o755,
	EnsurePolicy:     conduit.EnsureAll,
	SyncPolicy:       conduit.SyncRewrite,
	PathSafetyPolicy: conduit.PathSafetyRejectSymlinkParents,
	WritePolicy:      conduit.WriteDirect,
	Reporter:         &conduit.Report{},
}
```

- `DirMode` controls created directories.
- `FileMode` controls regular files.
- `ExecMode` controls `Exec` files.
- `EnsurePolicy` controls which node kinds `Ensure` and `EnsureDeep` may materialize.
- `SyncPolicy` controls which stateful memory states `Sync` and `SyncDeep` may write, with optional extra disk-state filters.
- `PathSafetyPolicy` controls whether mutating operations reject symlink parents during path resolution.
- `WritePolicy` controls direct rewrites vs atomic replacement in `File.WriteBytes`.
- `TempFilePlacement` and `TempDir` control where atomic writes stage their temporary file.
- `Reporter` optionally collects per-path operation results during deep traversal.

Available ensure policy bits:

- `conduit.EnsureDirs`
- `conduit.EnsureFiles`
- `conduit.EnsureExecs`
- `conduit.EnsureSyncables`

Available ensure policy presets:

- `conduit.EnsureAll`
- `conduit.EnsureScaffold`
- `conduit.EnsureNone`

Available sync policies:

- `conduit.SyncRewrite`: write loaded, dirty, and already-synced stateful content
- `conduit.SyncIfDirty`: write only dirty stateful content
- `conduit.SyncIfUnsynced`: write loaded and dirty stateful content, but skip already-synced content
- `conduit.SyncIfMissing`: write only when the file was last observed missing

Available sync filter bits:

- memory-state filters: `conduit.SyncOnLoaded`, `conduit.SyncOnSynced`, `conduit.SyncOnDirty`
- disk-state filters: `conduit.SyncOnDiskUnknown`, `conduit.SyncOnDiskMissing`, `conduit.SyncOnDiskPresent`

Available path safety policies:

- `conduit.PathSafetyRejectSymlinkParents`
- `conduit.PathSafetyFollowSymlinks`

Available write policies:

- `conduit.WriteDirect`
- `conduit.WriteAtomicReplace`

Available atomic-write temp placements:

- `conduit.TempFileSystem`
- `conduit.TempFileDir`
- `conduit.TempFileAdjacent`

Behavior notes:

- if no memory-state bits are set, sync defaults to `conduit.SyncRewrite`
- if no disk-state bits are set, sync does not restrict by disk state
- combine memory and disk bits with `|` when you need both gates

`conduit.DefaultContext` is:

```go
conduit.Context{
	DirMode:          0o755,
	FileMode:         0o644,
	ExecMode:         0o755,
	EnsurePolicy:     conduit.EnsureAll,
	SyncPolicy:       conduit.SyncRewrite,
	PathSafetyPolicy: conduit.PathSafetyRejectSymlinkParents,
}
```

`WritePolicy` defaults to `conduit.WriteDirect`, and `TempFilePlacement` defaults to `conduit.TempFileSystem` when you enable atomic replacement.

Collect a report during a deep operation:

```go
var report conduit.Report

ctx := conduit.DefaultContext
ctx.Reporter = &report

_, _ = conduit.LoadDeep(&ws, ctx)

if report.HasErrors() {
	// inspect report.Entries()
}
```

Validation uses a small dedicated options type:

```go
opts := conduit.ValidateOptions{
	Reporter: &report,
}

_, _ = conduit.ValidateDeep(&ws, opts)
```

## Typical workflows

Bootstrap a new workspace:

```go
var ws Workspace
_ = conduit.Compose("/srv/workspace", &ws)
_, _ = conduit.EnsureDeep(&ws, conduit.DefaultContext)

svc, _ := ws.Services.Add("billing", conduit.DefaultContext)
svc.Config.Set(ServiceConfig{Name: "billing", Port: 8080})
_, _ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Sync only dirty stateful content during a pass:

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty

_, _ = conduit.SyncDeep(&ws, ctx)
```

Write only when the file was observed missing:

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty | conduit.SyncOnDiskMissing

_, _ = conduit.SyncDeep(&ws, ctx)
```

Discover an existing workspace without loading stateful content:

```go
var ws Workspace
_ = conduit.Compose("/srv/workspace", &ws)
_, _ = conduit.DiscoverDeep(&ws, conduit.DefaultContext)
```

Validate before syncing:

```go
_ = conduit.RenderDeep(&ws)
_, _ = conduit.ValidateDeep(&ws, conduit.ValidateOptions{})
_, _ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Load discovered content into memory:

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_, _ = conduit.LoadDeep(&ws, conduit.DefaultContext)

svc := ws.Services.MustAt("billing")
cfg := svc.Config.MustGet()
cfg.Port = 9000
svc.Config.Set(cfg)

_, _ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Check disk presence without loading content:

```go
svc := ws.Services.MustAt("billing")
_, _ = conduit.ScanDeep(svc, conduit.DefaultContext)
```

The core rule is simple: Conduit never decides direction for you. You choose whether the next step is ensure, default, discover, load, render, validate, sync, or scan, and you choose how aggressive sync should be for stateful content.
