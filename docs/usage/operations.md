# Operations usage

Compile the declaration, compose a tree, then choose explicit operations on its `Layout` manager. The manager owns whole-tree traversal; carriers own local filesystem and cached-content behavior.

## Layout manager

```go
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Service struct {
	Root   layout.Dir                  `layout:"dir" path:"."`
	Config formats.YAMLFile[ServiceConfig] `layout:"content" path:"config.yaml"`
	Runner layout.Exec                 `layout:"exec" path:"bin/run"`
}

type Workspace struct {
	Root     layout.Dir             `layout:"dir" path:"."`
	Services layout.Slot[*Service] `layout:"directories" path:"services"`
}

structure, err := conduit.Compile[Workspace]()
if err != nil {
	return err
}
managed, err := structure.Compose("/srv/workspace")
if err != nil {
	return err
}
ws := managed.Tree()
```

Import `conduit`, `layout`, and `formats` from `github.com/qlustra/conduit`. Each fragment below assumes this setup.

Compilation recognizes role and path declarations. Composition creates a fresh tree, allocates declared pointer paths, and configures slots without touching the filesystem. The compiled Structure can be reused to compose independent trees at other roots. Untagged fields and embeddings do not participate; nested groups use `layout:"group"`.

## Ensure

```go
_, err := managed.Ensure(conduit.DefaultContext)
```

Ensure creates declared directories, raw files, executable files, and typed backing files according to `EnsurePolicy`. It prepares a collection root and every cached directory member root before visiting that collection's descendant fields, even for empty trees. A later failure can leave other member roots created.

Typed files use `EnsureSyncables`. Links are persisted by Sync after a target is cached. Ensure visits existing cached members; it does not discover names, load content, or persist cached values.

Use `EnsureScaffold` for raw Dir, File, and Exec scaffolding, or combine policy bits such as `EnsureDirs | EnsureSyncables`.

## Load and Discover

```go
_, err := managed.Load(conduit.DefaultContext)
// Or discover names and presence while preserving live content:
_, err = managed.Discover(conduit.DefaultContext)
```

Both operations read collection directories and compose physical members into the cache. Directory slots admit directories, file slots regular files, and link slots symlinks, including dangling links. Members are composed and processed lazily in filesystem order, stopping on the first failure.

Load reads stateful content. A missing file clears cached content and marks disk state missing. Discover observes presence while preserving content, targets, and memory state.

Both accumulate members. Stale cached names remain in the cache but are not visited during these filesystem-driven passes. Neither writes or evicts cached members.

## Scan

```go
_, err := managed.Scan(conduit.DefaultContext)
```

Scan observes disk presence for declared nodes and cached members. It preserves live content and memory state, does not discover new collection names, and performs no writes.

## Default and Render

```go
if err := managed.Default(); err != nil {
	return err
}
if err := managed.Render(); err != nil {
	return err
}
```

Default calls local `Default() error` hooks. Implementations can use `SetDefault`, `SetDefaultContext`, or `SetDefaultTarget` to preserve values already present.

Render calls `Renderable.Render`, or the Templatable `Template`/`RenderTemplate` path, and stores the result through `SetRendered`. Both passes visit composed and cached members and keep their effects in memory. Sync writes the resulting content separately.

## Sync

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty
_, err := managed.Sync(ctx)
```

Sync persists eligible cached typed-file content, rendered text, and link targets. Its policy gates writes by memory state and optionally the last observed disk state. A carrier may create parent directories for its own write.

Ensure and Sync are separate passes. Call Ensure when you need empty directories or other declared scaffolding. Sync neither discovers members nor deletes disk entries omitted from memory.

Use pointer collection members for stateful carriers. Pointers let Sail bind mutable hooks to the cached instance. Values yielded by a collection are ordinary Go copies; pointer-only hooks on nonaddressable values cannot be bound. A value Format can still use its value-receiver Ensure hook.

## Validate

```go
_, err := managed.Validate(conduit.ValidateOptions{})
```

Validate checks collection directories before entering members and calls local validation hooks. It validates declared nodes and cached members, applying path-safety policy to built-in carriers. It does not mutate content, cache membership, or the filesystem.

## Unload

```go
err := managed.Unload()
```

Unload clears content through local carrier hooks. It retains cached collection membership and the last disk observation, so a later Scan, Load, or Discover can work with the same live tree.

## Results and reporting

Ensure, Load, Discover, Scan, Sync, and Validate return `(ResultCode, error)`. The result describes the visited root; a traversal result does not mean every descendant had the same outcome. Default, Render, and Unload return an error.

Operations stop at the first failure. Attach a `Report` to capture per-path results and enclosing collection failures:

```go
var report conduit.Report
ctx := conduit.DefaultContext
ctx.Reporter = &report

_, err := managed.Sync(ctx)
fmt.Println(report.RenderTree())
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
- `EnsurePolicy` controls which node kinds `Ensure` may materialize.
- `SyncPolicy` controls which stateful memory states `Sync` may write, with optional extra disk-state filters.
- `PathSafetyPolicy` controls whether mutating operations reject symlink parents during path resolution.
- `WritePolicy` controls direct rewrites vs atomic replacement in `File.WriteBytes`.
- `TempFilePlacement` and `TempDir` control where atomic writes stage their temporary file.
- `Reporter` optionally collects per-path operation results during tree traversal.

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

Collect a report during a tree operation:

```go
var report conduit.Report

ctx := conduit.DefaultContext
ctx.Reporter = &report

_, _ = managed.Load(ctx)

if report.HasErrors() {
	// inspect report.Entries()
}
```

Validation uses a small dedicated options type:

```go
opts := conduit.ValidateOptions{
	Reporter: &report,
}

_, _ = managed.Validate(opts)
```

## Typical collection workflows

Compose a cached member, prepare content, then persist a whole-tree pass:

```go
svc, err := ws.Services.ComposeMember("billing")
if err != nil {
	return err
}
svc.Config.Set(ServiceConfig{Name: "billing", Port: 8080})
_, err = managed.Sync(conduit.DefaultContext)
```

`ComposeMember` is memory-only. When the workflow also requires declared scaffolding, call `managed.Ensure(ctx)` before Sync. Both manager passes operate on the whole tree.

Load discovered content, edit one member, then persist:

```go
if _, err := managed.Load(conduit.DefaultContext); err != nil {
	return err
}
svc, ok := ws.Services.Get("billing")
if !ok {
	return fmt.Errorf("billing service was not discovered")
}
cfg := svc.Config.MustGet()
cfg.Port = 9000
svc.Config.Set(cfg)

ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty
_, err := managed.Sync(ctx)
```

Write only when a file was observed missing:

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty | conduit.SyncOnDiskMissing
_, err := managed.Sync(ctx)
```

For generated content, use `Default → Render → Validate → Sync`. For disk-authoritative content, use `Load → edit → Validate → Sync`. Choose Ensure separately when the workflow requires scaffolding.
