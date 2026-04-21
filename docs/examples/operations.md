# Operations examples

These examples show the intent behind each operation and how they fit together in realistic flows.

## Bootstrap a new service

Goal: create the filesystem contract first, then write initial config.

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_ = conduit.EnsureDeep(&ws, conduit.DefaultContext)

svc, _ := ws.Services.Add("api", conduit.DefaultContext)
svc.Config.Set(ServiceConfig{
	Name: "api",
	Port: 8080,
})

_ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Why the sequence matters:

- `Compose` binds paths
- `EnsureDeep` creates the static roots
- `Add` creates one explicit dynamic child
- `SyncDeep` persists the in-memory config

## Load, edit, sync

Goal: treat disk as authoritative, change one field, then write it back.

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_ = conduit.LoadDeep(&ws, conduit.DefaultContext)

svc := ws.Services.MustAt("api")
cfg := svc.Config.MustGet()
cfg.Port = 9090
svc.Config.Set(cfg)

_ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

This is the main "edit existing content" path.

## Inspect presence without loading

Goal: check whether known files exist without replacing in-memory content.

```go
svc := ws.Services.MustAt("api")
svc.Config.Set(ServiceConfig{Name: "preview", Port: 3000})

_ = conduit.ScanDeep(svc, conduit.DefaultContext)
```

After `ScanDeep`:

- the config's disk metadata is refreshed
- the in-memory value is still `preview:3000`
- no file content was loaded

This is useful for validation or status reporting.

## Discover dynamic entries from disk

Goal: find existing service directories and load their typed files.

```go
var ws Workspace

_ = conduit.Compose("/srv/workspace", &ws)
_ = conduit.LoadDeep(&ws, conduit.DefaultContext)

for _, name := range ws.Services.Keys() {
	svc := ws.Services.MustAt(name)
	cfg := svc.Config.MustGet()
	_ = cfg
}
```

Use `LoadDeep` for discovery. `ScanDeep` does not enumerate new slot entries.

## Initialize defaults lazily

Goal: keep a default in memory only when a file is missing.

```go
svc := ws.Services.MustAt("worker")

_ = svc.Config.LoadOrInit(ServiceConfig{
	Name: "worker",
	Port: 7000,
})

_ = svc.Config.Sync(conduit.DefaultContext)
```

This avoids writing a file unless you choose to persist the default.
