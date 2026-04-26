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
err := conduit.EnsureDeep(&ws, conduit.DefaultContext)
```

What it does:

- creates directories declared by `Dir`
- creates files declared by `File`
- creates executable files declared by `Exec`
- ensures already cached `Slot` items

What it does not do:

- load typed content into memory
- discover new slot entries from disk
- delete anything

For `Slot[T]`, only cached items are ensured. Use `slot.Add(name, ctx)` when you want to create a new dynamic child explicitly.

## Load

`LoadDeep(target, ctx)` reads filesystem content into the in-memory model:

```go
err := conduit.LoadDeep(&ws, conduit.DefaultContext)
```

What it does:

- loads typed files such as `JSONFile[T]`, `YAMLFile[T]`, and `TOMLFile[T]`
- discovers slot entries by listing child directories on disk
- composes and loads discovered slot items recursively

What it does not do:

- create missing files
- write anything back to disk
- remove cached slot items that no longer exist

For a typed file, loading a missing file clears in-memory content and marks the file as missing.

## Discover

`DiscoverDeep(target, ctx)` discovers the declared layout from disk without loading typed file content:

```go
err := conduit.DiscoverDeep(&ws, conduit.DefaultContext)
```

What it does:

- discovers slot entries by listing child directories on disk
- composes discovered slot items recursively
- updates typed-file disk state through the declared layout
- preserves the current in-memory content and memory state

What it does not do:

- load file content into memory
- create missing files
- write anything back to disk

This makes `DiscoverDeep` the middle ground between `LoadDeep` and `ScanDeep`: it discovers structure like `LoadDeep`, but it only observes typed files like `ScanDeep`.

## Sync

`SyncDeep(target, ctx)` writes sync-eligible in-memory typed content back to disk:

```go
err := conduit.SyncDeep(&ws, conduit.DefaultContext)
```

What it does:

- writes typed files that currently hold content and match `ctx.SyncPolicy`
- syncs already cached slot items recursively
- allows callers to choose rewrite behavior per sync pass

What it does not do:

- materialize standalone raw `Dir` or `File` fields
- invent uncached slot entries
- delete files or directories that are missing from memory
- merge disk content with memory content

For typed files, `Sync` is a no-op when no content is loaded or when the current memory state is excluded by `ctx.SyncPolicy`.

## Scan

`ScanDeep(target, ctx)` refreshes disk-presence metadata for already composed items:

```go
err := conduit.ScanDeep(&ws, conduit.DefaultContext)
```

What it does:

- updates the disk state for typed files
- preserves the current in-memory content and memory state
- scans cached slot items recursively

What it does not do:

- load file content
- discover new slot entries from disk
- modify files on disk

This makes `ScanDeep` useful for "is it there?" checks, not discovery.

## Context

Every filesystem operation accepts a `Context`:

```go
ctx := conduit.Context{
	DirMode:    0o755,
	FileMode:   0o644,
	ExecMode:   0o755,
	SyncPolicy: conduit.SyncRewrite,
}
```

- `DirMode` controls created directories.
- `FileMode` controls regular files.
- `ExecMode` controls `Exec` files.
- `SyncPolicy` controls which typed memory states `Sync` and `SyncDeep` may write.

Available sync policies:

- `conduit.SyncRewrite`: write loaded, dirty, and already-synced typed content
- `conduit.SyncIfDirty`: write only dirty typed content
- `conduit.SyncIfUnsynced`: write loaded and dirty typed content, but skip already-synced content

`conduit.DefaultContext` is:

```go
conduit.Context{
	DirMode:    0o755,
	FileMode:   0o644,
	ExecMode:   0o755,
	SyncPolicy: conduit.SyncRewrite,
}
```

## Typical workflows

Bootstrap a new workspace:

```go
var ws Workspace
_ = conduit.Compose("/srv/workspace", &ws)
_ = conduit.EnsureDeep(&ws, conduit.DefaultContext)

svc, _ := ws.Services.Add("billing", conduit.DefaultContext)
svc.Config.Set(ServiceConfig{Name: "billing", Port: 8080})
_ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Sync only dirty typed content during a pass:

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty

_ = conduit.SyncDeep(&ws, ctx)
```

Load an existing workspace, edit it, then persist:

```go
var ws Workspace
_ = conduit.Compose("/srv/workspace", &ws)
_ = conduit.DiscoverDeep(&ws, conduit.DefaultContext)
```

Load discovered content into memory:

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_ = conduit.LoadDeep(&ws, conduit.DefaultContext)

svc := ws.Services.MustAt("billing")
cfg := svc.Config.MustGet()
cfg.Port = 9000
svc.Config.Set(cfg)

_ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Check disk presence without loading content:

```go
svc := ws.Services.MustAt("billing")
_ = conduit.ScanDeep(svc, conduit.DefaultContext)
```

The core rule is simple: Conduit never decides direction for you. You choose whether the next step is ensure, discover, load, sync, or scan, and you choose how aggressive sync should be for typed content.
