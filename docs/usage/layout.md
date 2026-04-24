# Layout usage

Conduit layouts are plain Go structs with `layout` tags. `Compose` walks the struct, resolves each tagged field relative to a root path, and binds the corresponding filesystem handle or typed file.

## Defining a layout

Use `layout:"."` for the root and relative paths for children:

```go
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Service struct {
	Root   conduit.Dir                     `layout:"."`
	Config conduit.YAMLFile[ServiceConfig] `layout:"config.yaml"`
	Logs   conduit.Dir                     `layout:"logs"`
}

type Workspace struct {
	Root     conduit.Dir            `layout:"."`
	Services conduit.Slot[*Service] `layout:"services"`
}

var ws Workspace
err := conduit.Compose("/srv/workspace", &ws)
```

`Compose` requires a non-nil pointer to a struct. Tagged pointer fields are allocated automatically if needed.

## Node types

### `Dir`

`Dir` is a stateless handle to a directory path.

Useful methods:

- `Path()` returns the absolute or cleaned path bound during composition.
- `Exists()` reports whether the directory currently exists on disk.
- `Join(...)` builds a descendant path.
- `Dir(name)` and `File(name)` derive child handles.
- `Ensure(ctx)` creates the directory tree.
- `DeleteIfExists()` removes the directory recursively when it exists.

### `File`

`File` is a stateless handle to a regular file path.

Useful methods:

- `Path()` and `Exists()`
- `ReadBytes()` and `ReadBytesIfExists()`
- `WriteBytes(data, dirMode, fileMode)`
- `Ensure(ctx)` to create the file and its parent directories
- `DeleteIfExists()`

Use `File` when you want raw bytes and do not need codec-backed state tracking.

### `Exec`

`Exec` is a `File` with executable semantics.

Useful methods:

- `Ensure(ctx)` and `EnsureExecutable(ctx)` create the file with executable permissions.
- `IsExecutable()` reports whether the current target is an executable regular file.
- `Command(ctx, opts)`, `Run(ctx, opts)`, `Output(ctx, opts)`, and `CombinedOutput(ctx, opts)` run the managed file.

`RunOptions` supports:

- `Args` for argv
- `Dir` for the working directory
- `Env` for extra environment variables
- `Stdin`, `Stdout`, and `Stderr` for stream wiring
- `Interpreter` for running the file through something like `[]string{"sh"}` or `[]string{"python3"}`

If `Context.ExecMode` is unset, Conduit falls back to `FileMode` and adds execute bits automatically.

### Typed files

`JSONFile[T]`, `YAMLFile[T]`, and `TOMLFile[T]` are codec-backed files that embed `Format[T]`.

They behave like regular layout nodes, but also keep typed content in memory:

```go
type App struct {
	Config conduit.JSONFile[map[string]any] `layout:"config.json"`
}
```

See [Formats usage](formats.md) for the full content API.

### `Slot[T]`

`Slot[T]` models repeated child structures under one directory:

```go
type Service struct {
	Config conduit.YAMLFile[ServiceConfig] `layout:"config.yaml"`
}

type Workspace struct {
	Services conduit.Slot[*Service] `layout:"services"`
}
```

Each key becomes a child root under the slot path:

```text
services/
  api/
    config.yaml
  worker/
    config.yaml
```

Useful methods:

- `At(name)` composes and caches an item lazily.
- `MustAt(name)` is the panicking version of `At`.
- `Add(name, ctx)` creates the child root on disk, composes the item, and ensures its declared structure.
- `Get(name)` reads the cache without composing.
- `Put(name, item)`, `Remove(name)`, `Clear()`, and `Keys()` manage the cache.
- `Require(name)` fails unless the child directory already exists on disk.
- `Root()` returns the slot root as a `Dir`.

## Composition rules

`Compose` only binds:

- exported fields
- fields tagged with `layout:"..."`

The tag is always resolved relative to the containing struct's root. `layout:"."` means "bind this field to the current root".

The deep operations (`EnsureDeep`, `DiscoverDeep`, `LoadDeep`, `SyncDeep`, `ScanDeep`) also recurse through anonymous embedded fields.

Nested structs work naturally:

```go
type Tooling struct {
	Scripts struct {
		Build conduit.Exec `layout:"build"`
		Test  conduit.Exec `layout:"test"`
	} `layout:"bin"`
}
```

After `Compose("/workspace", &tooling)`, the executables resolve to:

- `/workspace/bin/build`
- `/workspace/bin/test`

## Common layout patterns

Static structure:

```go
type Repo struct {
	Root   conduit.Dir                `layout:"."`
	Config conduit.TOMLFile[Settings] `layout:"settings.toml"`
	Hooks  conduit.Dir                `layout:"hooks"`
}
```

Static structure plus dynamic children:

```go
type Project struct {
	Config conduit.YAMLFile[ProjectConfig] `layout:"project.yaml"`
}

type Monorepo struct {
	Root     conduit.Dir            `layout:"."`
	Projects conduit.Slot[*Project] `layout:"projects"`
}
```

Managed tools next to data:

```go
type Environment struct {
	Root   conduit.Dir                 `layout:"."`
	Env    conduit.YAMLFile[EnvConfig] `layout:"env.yaml"`
	Deploy conduit.Exec                `layout:"bin/deploy"`
}
```
