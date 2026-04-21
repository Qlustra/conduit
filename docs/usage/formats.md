# Formats usage

Typed files are the stateful part of a Conduit layout. They combine a filesystem path with a codec and an in-memory value.

The public format types are:

- `conduit.JSONFile[T]`
- `conduit.YAMLFile[T]`
- `conduit.TOMLFile[T]`

All three expose the same content API through `Format[T]`.

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
- `Save(ctx)` writes the currently loaded value
- `Sync(ctx)` writes the currently loaded value, or does nothing if no value is loaded

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

## State model

Formats track two independent axes internally:

- disk state: unknown, missing, or present
- memory state: unknown, loaded, synced, or dirty

The important behavioral rules are:

- `Set` marks the value dirty.
- `Load` marks the value loaded when the file exists.
- `Save` and `Sync` mark the value synced after a successful write.
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

## Codec behavior

Each typed file has a fixed codec:

- `JSONFile[T]` writes indented JSON with a trailing newline
- `YAMLFile[T]` uses `gopkg.in/yaml.v3`
- `TOMLFile[T]` uses `github.com/pelletier/go-toml/v2`

Choose the format based on how the file will be consumed:

- JSON for machine-oriented artifacts
- YAML for hand-edited operational config
- TOML for settings-style files
