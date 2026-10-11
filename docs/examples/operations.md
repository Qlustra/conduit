# Operations examples

These examples show the intent behind each operation and how they fit together in realistic flows.

Collection examples below assume `managed` is an already composed
`Layout[Workspace]` using the setup in [Operations usage](../usage/operations.md#layout-manager), and `ws := managed.Tree()`.

## Compose a new service

Goal: obtain a cached member, then prepare its in-memory config.

```go
svc, _ := ws.Services.ComposeMember("api")
svc.Config.Set(ServiceConfig{
	Name: "api",
	Port: 8080,
})
```

`ComposeMember` touches memory only. `managed.Ensure(ctx)` prepares declared and cached structure in a whole-tree pass; `managed.Sync(ctx)` persists eligible content separately.

## Sync only dirty content

Goal: avoid rewriting typed files that are already loaded or already synced.

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty

_, _ = managed.Sync(ctx)
```

Why the policy matters:

- dirty typed files are written
- loaded-but-unchanged typed files are skipped
- already-synced typed files are skipped

## Load, edit, sync

Goal: treat disk as authoritative, change one field, then write it back.

```go
_, _ = managed.Load(conduit.DefaultContext)

svc := ws.Services.MustComposeMember("api")
cfg := svc.Config.MustGet()
cfg.Port = 9090
svc.Config.Set(cfg)

_, _ = managed.Sync(conduit.DefaultContext)
```

This is the main "edit existing content" path.

## Inspect presence without loading

Goal: check whether known files exist without replacing in-memory content.

```go
svc := ws.Services.MustComposeMember("api")
svc.Config.Set(ServiceConfig{Name: "preview", Port: 3000})

_, _ = managed.Scan(conduit.DefaultContext)
```

After the whole-tree `Scan` pass:

- the config's disk metadata is refreshed
- the in-memory value is still `preview:3000`
- no file content was loaded

This is useful for validation or status reporting.

## Discover dynamic entries from disk

Goal: find existing service directories without loading their typed files.

```go
_, _ = managed.Discover(conduit.DefaultContext)

for _, name := range ws.Services.Keys() {
	svc := ws.Services.MustComposeMember(name)
	_, _ = svc.Config.Get()
}
```

Use `Layout.Discover` when you want slot discovery but do not want stateful content in memory.

## Discover and then load content

Goal: enumerate existing services first, then load only when you choose to.

```go
_, _ = managed.Discover(conduit.DefaultContext)
_, _ = managed.Load(conduit.DefaultContext)

for _, name := range ws.Services.Keys() {
	svc := ws.Services.MustComposeMember(name)
	cfg := svc.Config.MustGet()
	_ = cfg
}
```

`Layout.Scan` does not enumerate new slot entries. `Layout.Discover` does, while preserving unloaded cached state and retaining stale cached members.

## Initialize defaults lazily

Goal: keep a default in memory only when a file is missing.

```go
svc := ws.Services.MustComposeMember("worker")

_ = svc.Config.LoadOrInit(ServiceConfig{
	Name: "worker",
	Port: 7000,
})

ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfMissing

_, _ = svc.Config.Sync(ctx)
```

This writes the default only after the file has been observed missing, and it skips later sync passes once the file exists.

## Require force only when a file has meaningful content

Goal: treat whitespace-only or comment-only files as safe to replace, while requiring explicit force for real content.

```go
hasMeaningfulContent := func(line string) (bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return false, nil
	}
	return true, nil
}

readme := layout.NewFile("/srv/workspace/README.md")

matched, err := readme.MatchLinesIfExists(layout.DefaultContext, hasMeaningfulContent)
if err != nil {
	return err
}

if matched && !force {
	return fmt.Errorf("refusing to replace %s without --force", readme.Path())
}

if err := readme.WriteBytes([]byte("# generated\n"), layout.DefaultContext); err != nil {
	return err
}
```

Why the sequence matters:

- `MatchLinesIfExists` keeps your notion of meaningful content in one small callback
- `MatchLinesIfExists` folds the common missing-file case into `false`
- `MatchLinesIfExists` removes the `bufio.Scanner` boilerplate for the common line-oriented case
- the matcher can stop at the first line that proves `--force` is needed
- the write path stays separate and explicit

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
	Readme ReadmeFile `layout:"content" path:"README.md"`
}

svc := ws.Services.MustComposeMember("api")
svc.Readme.SetContext(ReadmeContext{
	Name: "api",
	Port: 8080,
})

_ = managed.Render()
_, _ = managed.Sync(conduit.DefaultContext)
```

Why the sequence matters:

- `SetContext` prepares render inputs in memory
- `Layout.Render` turns template context into cached file content
- `Layout.Sync` persists the rendered text using the same typed-file sync rules as other managed files

## Collect a traversal report

Goal: inspect what a tree operation actually visited and whether it skipped, wrote, validated, or failed per path.

```go
var report conduit.Report

ctx := conduit.DefaultContext
ctx.Reporter = &report

_, _ = managed.Sync(ctx)

entries := report.Entries()
tree := report.RenderTree()

_ = entries
_ = tree
```

Why this is useful:

- `ResultCode` on the root only tells you the top-level outcome
- the report lets you inspect per-path results after the manager’s Ensure, Load, Discover, Scan, Sync, or Validate passes
- `RenderTree()` is useful when you want one human-readable summary for logs or tests

Validation uses the same `Report` type through `ValidateOptions`:

```go
var report conduit.Report

opts := conduit.ValidateOptions{
	Reporter: &report,
}

_, _ = managed.Validate(opts)
```
