# Operations examples

These examples show the intent behind each operation and how they fit together in realistic flows.

## Bootstrap a new service

Goal: create the filesystem contract first, then write initial config.

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_, _ = conduit.EnsureDeep(&ws, conduit.DefaultContext)

svc, _ := ws.Services.Add("api", conduit.DefaultContext)
svc.Config.Set(ServiceConfig{
	Name: "api",
	Port: 8080,
})

_, _ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Why the sequence matters:

- `Compose` binds paths
- `EnsureDeep` creates the static roots
- `Add` creates one explicit dynamic child
- `SyncDeep` persists the in-memory config

## Sync only dirty content

Goal: avoid rewriting typed files that are already loaded or already synced.

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty

_, _ = conduit.SyncDeep(&ws, ctx)
```

Why the policy matters:

- dirty typed files are written
- loaded-but-unchanged typed files are skipped
- already-synced typed files are skipped

## Load, edit, sync

Goal: treat disk as authoritative, change one field, then write it back.

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_, _ = conduit.LoadDeep(&ws, conduit.DefaultContext)

svc := ws.Services.MustAt("api")
cfg := svc.Config.MustGet()
cfg.Port = 9090
svc.Config.Set(cfg)

_, _ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

This is the main "edit existing content" path.

## Inspect presence without loading

Goal: check whether known files exist without replacing in-memory content.

```go
svc := ws.Services.MustAt("api")
svc.Config.Set(ServiceConfig{Name: "preview", Port: 3000})

_, _ = conduit.ScanDeep(svc, conduit.DefaultContext)
```

After `ScanDeep`:

- the config's disk metadata is refreshed
- the in-memory value is still `preview:3000`
- no file content was loaded

This is useful for validation or status reporting.

## Discover dynamic entries from disk

Goal: find existing service directories without loading their typed files.

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_, _ = conduit.DiscoverDeep(&ws, conduit.DefaultContext)

for _, name := range ws.Services.Keys() {
	svc := ws.Services.MustAt(name)
	_, _ = svc.Config.Get()
}
```

Use `DiscoverDeep` when you want slot discovery but do not want typed file content in memory.

## Discover and then load content

Goal: enumerate existing services first, then load only when you choose to.

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_, _ = conduit.DiscoverDeep(&ws, conduit.DefaultContext)
_, _ = conduit.LoadDeep(&ws, conduit.DefaultContext)

for _, name := range ws.Services.Keys() {
	svc := ws.Services.MustAt(name)
	cfg := svc.Config.MustGet()
	_ = cfg
}
```

`ScanDeep` does not enumerate new slot entries. `DiscoverDeep` does, while still preserving unloaded typed-file memory.

## Initialize defaults lazily

Goal: keep a default in memory only when a file is missing.

```go
svc := ws.Services.MustAt("worker")

_ = svc.Config.LoadOrInit(ServiceConfig{
	Name: "worker",
	Port: 7000,
})

ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfMissing

_, _ = svc.Config.Sync(ctx)
```

This writes the default only after the file has been observed missing, and it skips later sync passes once the file exists.

## Render generated files explicitly

Goal: keep derived text artifacts in a separate render phase instead of mixing them into load or sync.

```go
type ReadmeContext struct {
	Name string
	Port int
}

type ReadmeFile struct {
	layout.TextTemplate[ReadmeContext]
}

func (f *ReadmeFile) Template() string {
	return "# {{ .Name }}\n\nPort: {{ .Port }}\n"
}

type Service struct {
	Readme ReadmeFile `layout:"README.md"`
}

svc := ws.Services.MustAt("api")
svc.Readme.SetContext(ReadmeContext{
	Name: "api",
	Port: 8080,
})

_ = conduit.RenderDeep(&ws)
_, _ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Why the sequence matters:

- `SetContext` prepares render inputs in memory
- `RenderDeep` turns template context into cached file content
- `SyncDeep` persists the rendered text using the same typed-file sync rules as other managed files

## Collect a traversal report

Goal: inspect what a deep operation actually visited and whether it skipped, wrote, or failed per path.

```go
var report conduit.Report

ctx := conduit.DefaultContext
ctx.Reporter = &report

_, _ = conduit.SyncDeep(&ws, ctx)

entries := report.Entries()
tree := report.RenderTree()

_ = entries
_ = tree
```

Why this is useful:

- `ResultCode` on the root only tells you the top-level outcome
- the report lets you inspect per-path results after `EnsureDeep`, `LoadDeep`, `DiscoverDeep`, `ScanDeep`, or `SyncDeep`
- `RenderTree()` is useful when you want one human-readable summary for logs or tests
