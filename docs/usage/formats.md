# Formats usage

Typed files are the stateful part of a Conduit layout. They combine a filesystem path with a codec and an in-memory value.

The public format types are:

- `conduit.JSONFile[T]`
- `conduit.YAMLFile[T]`
- `conduit.TOMLFile[T]`
- `conduit.TextTemplate[C]`

The typed formats expose the same content API through `Format[T]`. `TextTemplate[C]` mirrors that state model for raw text and adds cached render context.

## Declaring typed files

```go
type ServiceConfig struct {
	Name string `yaml:"name" json:"name" toml:"name"`
	Port int    `yaml:"port" json:"port" toml:"port"`
}

type Service struct {
	YAML conduit.YAMLFile[ServiceConfig] `layout:"service.yaml"`
	JSON conduit.JSONFile[ServiceConfig] `layout:"service.json"`
	TOML conduit.TOMLFile[ServiceConfig] `layout:"service.toml"`
}
```

## Loading and reading content

`Load()` reads from disk into memory and reports whether the file existed:

```go
loaded, err := service.YAML.Load()
if err != nil {
	return err
}
if loaded {
	cfg := service.YAML.MustGet()
	_ = cfg
}
```

You can also read the cached value without panicking:

```go
cfg, ok := service.YAML.Get()
```

Useful read-side methods:

- `Load() (bool, error)`
- `LoadOrInit(defaultValue T) error`
- `Get() (T, bool)`
- `MustGet() T`
- `HasContent() bool`
- `HasBeenLoaded() bool`

`LoadOrInit` is useful for "load if present, otherwise start with this default":

```go
err := service.YAML.LoadOrInit(ServiceConfig{
	Name: "billing",
	Port: 8080,
})
```

If the file is missing, the default is stored in memory and marked dirty. It is not written until you call `Save`, `Sync`, or `SyncDeep`.

When you want to apply defaults without touching disk, wrappers can use `SetDefault(...)` inside `Default() error` and then participate in `DefaultDeep(...)`.

## Mutating content

Use `Set` to replace the in-memory value:

```go
cfg := service.YAML.MustGet()
cfg.Port = 9000
service.YAML.Set(cfg)
```

Other write-side methods:

- `Set(value T)`
- `Clear()`
- `Unload()`
- `Delete() error`

`Clear()` and `Unload()` both remove cached content from memory. `Delete()` removes the file from disk if it exists, then clears the in-memory value.

## Persisting content

There are three ways to write typed files:

- `Write(value, ctx)` marshals and writes a supplied value directly
- `Save(ctx)` writes the cached value
- `Sync(ctx)` writes the cached value when `ctx.SyncPolicy` allows the current memory state, or does nothing otherwise

Example:

```go
service.YAML.Set(ServiceConfig{
	Name: "billing",
	Port: 9000,
})

err := service.YAML.Save(conduit.DefaultContext)
```

Or as part of a larger layout:

```go
err := conduit.SyncDeep(&workspace, conduit.DefaultContext)
```

To avoid rewriting already-synced or freshly-loaded files during a larger pass:

```go
ctx := conduit.DefaultContext
ctx.SyncPolicy = conduit.SyncIfDirty

err := conduit.SyncDeep(&workspace, ctx)
```

## State model

Formats track two independent axes internally:

- disk state: unknown, missing, or present
- memory state: unknown, loaded, synced, or dirty

The important behavioral rules are:

- `Set` marks the value dirty.
- `Load` marks the value loaded when the file exists.
- `Save` and `Sync` mark the value synced after a successful write.
- `Discover` updates disk knowledge without overwriting in-memory content.
- `Scan` updates disk knowledge without overwriting in-memory content.
- loading a missing file clears in-memory content and marks disk as missing.

This separation is what lets Conduit avoid implicit reconciliation.

For the full state model and transitions, see [States usage](states.md).

## Scanning

`Scan()` checks whether the file exists on disk and updates only the disk side of the state model:

```go
_, err := service.YAML.Scan()
```

That makes it safe to ask "is this file present?" without loading or replacing the current cached value.

`Discover()` has the same typed-file behavior as `Scan()`. The distinction shows up during deep traversal: `DiscoverDeep` discovers slot items from disk, while `ScanDeep` only visits already cached items.

## Codec behavior

Each typed file has a fixed codec:

- `JSONFile[T]` writes indented JSON with a trailing newline
- `YAMLFile[T]` uses `gopkg.in/yaml.v3`
- `TOMLFile[T]` uses `github.com/pelletier/go-toml/v2`

Choose the format based on how the file will be consumed:

- JSON for machine-oriented artifacts
- YAML for hand-edited operational config
- TOML for settings-style files

## Text templates

Use `TextTemplate[C]` for fully derived raw-text artifacts such as README files, scripts, or generated notes.

Example:

```go
type ReadmeContext struct {
	Name  string
	Items []string
}

type ReadmeFile struct {
	conduit.TextTemplate[ReadmeContext]
}

func (f *ReadmeFile) Template() string {
	return "# {{ .Name }}\n"
}
```

`TextTemplate[C]` exposes the same file-state operations as `Format[string]` and adds:

- `SetContext(ctx C)`
- `SetDefaultContext(ctx C) bool`
- `GetContext() (C, bool)`
- `MustContext() C`
- `HasContext() bool`
- `ClearContext()`
- `RenderTemplate(tpl string) (string, error)`
- `SetRendered(value string)`

You can use it in two ways:

- implement `Template() string` and let `RenderDeep` use the built-in `text/template` path
- implement `Render() (string, error)` for custom rendering logic; this takes precedence over `Template()`

Typical flow:

```go
file.SetContext(ReadmeContext{Name: "billing"})

if err := conduit.RenderDeep(&workspace); err != nil {
	return err
}

return conduit.SyncDeep(&workspace, conduit.DefaultContext)
```

This keeps rendering explicit:

- `SetContext` prepares render inputs in memory
- `RenderDeep` derives raw text into cached file content
- `SyncDeep` persists the rendered text to disk

Defaults fit into the same phase model:

- implement `Default() error` on wrappers that should seed missing typed content or missing render context
- use `SetDefault(...)` on typed files and `SetDefaultContext(...)` on text templates to avoid overwriting existing memory
- call `DefaultDeep(&workspace)` before `RenderDeep(&workspace)` when defaults should feed rendering
