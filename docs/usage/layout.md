# Layout usage

Conduit layouts are plain Go structs with `layout` tags. `Compose` walks the struct, resolves each tagged field relative to a root path, and binds the corresponding filesystem handle or typed file.

In code, that usually means:

- import `github.com/qlustra/conduit` for operations such as `Compose`
- import `github.com/qlustra/conduit/layout` for `Dir`, `File`, `Exec`, `Slot[T]`, and `TextTemplate[C]`
- import `github.com/qlustra/conduit/formats` for `JSONFile[T]`, `YAMLFile[T]`, and `TOMLFile[T]`

## Defining a layout

Use `layout:"."` for the root and relative paths for children:

```go
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Service struct {
	Root   layout.Dir                      `layout:"."`
	Config formats.YAMLFile[ServiceConfig] `layout:"config.yaml"`
	Logs   layout.Dir                      `layout:"logs"`
}

type Workspace struct {
	Root     layout.Dir             `layout:"."`
	Services layout.Slot[*Service]  `layout:"services"`
}

var ws Workspace
err := conduit.Compose("/srv/workspace", &ws)
```

`Compose` requires a non-nil pointer to a struct. Tagged pointer fields are allocated automatically if needed.

## Node types

### `Dir`

`Dir` is a stateless handle to a directory path.

Useful methods:

- `Path()` returns the cleaned path bound during composition.
- `Base()` returns the final path element.
- `Stem()` returns the final path element without its final extension.
- `RelTo(...)` and `JoinRelTo(...)` build paths relative to another path-bearing node.
- `RelToPath(...)` and `JoinRelToPath(...)` do the same against a raw string base.
- `DeclaredPath()` and `JoinDeclaredPath(...)` expose the node's own declared layout fragment.
- `ComposedBaseDir()`, `ComposedRelativePath()`, and `JoinComposedPath(...)` expose compose-base-relative paths when the handle belongs to a composed tree.
- `Exists()` reports whether the directory currently exists on disk.
- `Join(...)` builds a descendant path.
- `Dir(name)` and `File(name)` derive child handles.
- `Ensure(ctx)` creates the directory tree.
- `DeleteIfExists()` removes the directory recursively when it exists.

### `File`

`File` is a stateless handle to a regular file path.

Useful methods:

- `Path()` and `Exists()`
- `Base()`, `Ext()`, and `Stem()` for path fragments
- `RelTo(...)`, `JoinRelTo(...)`, `RelToPath(...)`, and `JoinRelToPath(...)` for ordinary relative path math
- `DeclaredPath()` and `JoinDeclaredPath(...)` for local declared layout fragments
- `ComposedBaseDir()`, `ComposedRelativePath()`, and `JoinComposedPath(...)` for compose-base-relative path fragments
- `ReadBytes()` and `ReadBytesIfExists()`
- `WriteBytes(data, dirMode, fileMode)`
- `Ensure(ctx)` to create the file and its parent directories
- `DeleteIfExists()`

Use `File` when you want raw bytes and do not need codec-backed state tracking.

### `Exec`

`Exec` is a `File` with executable semantics.

Useful methods:

- `Ensure(ctx)` and `EnsureExecutable(ctx)` create the file with executable permissions.
- `Base()`, `Ext()`, and `Stem()` are inherited from `File`.
- `RelTo(...)`, `JoinRelTo(...)`, `RelToPath(...)`, and `JoinRelToPath(...)` are inherited from `File`.
- `DeclaredPath()` and `JoinDeclaredPath(...)` are inherited from `File`.
- `ComposedBaseDir()`, `ComposedRelativePath()`, and `JoinComposedPath(...)` are inherited from `File`.
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

`formats.JSONFile[T]`, `formats.YAMLFile[T]`, and `formats.TOMLFile[T]` are codec-backed files that embed `layout.Format[T, C]`.

They behave like regular layout nodes, but also keep typed content in memory:

```go
type App struct {
	Config formats.JSONFile[map[string]any] `layout:"config.json"`
}
```

See [Formats usage](formats.md) for the full content API.

### `Slot[T]`

`Slot[T]` models repeated child structures under one directory:

```go
type Service struct {
	Config formats.YAMLFile[ServiceConfig] `layout:"config.yaml"`
}

type Workspace struct {
	Services layout.Slot[*Service] `layout:"services"`
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
- `Delete(name)` removes the child tree from disk when it exists and evicts the cached item.
- `Get(name)` reads the cache without composing.
- `DeclaredPath()` and `JoinDeclaredPath(...)` expose the slot field's own declared layout fragment.
- `ComposedBaseDir()`, `ComposedRelativePath()`, and `JoinComposedPath(...)` expose the slot root relative to the tree's compose base.
- `Entries()` returns a sorted snapshot of cached `{Name, Item}` pairs.
- `All()` iterates cached items in sorted key order with `for name, item := range slot.All()`.
- `Put(name, item)`, `Remove(name)`, `Clear()`, `Len()`, and `Keys()` manage the cache. `Remove(name)` is cache-only.
- `Require(name)` fails unless the child directory already exists on disk.
- `Root()` returns the slot root as a `Dir`.

`Entries()` and `All()` are cache-based only. They do not discover from disk or lazily compose missing items.

The composed-path helpers return `ok == false` until a node has been attached through `Compose`. When they are available, the compose base is the same root that anchored the whole composed tree, not the nearest nested struct or slot item.

The declared-path helpers are different: they return the node's own declared layout fragment only. They do not reconstruct ancestor fragments. For example, a field declared as `layout:"build"` reports `build`, not `bin/build`, even when it lives inside a nested struct rooted at `layout:"bin"`.

The ordinary relative helpers are different again: they ignore composition metadata and declared layout metadata entirely. They just perform `filepath.Rel` and optional joining against another path-bearing node or a raw base path.

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
		Build layout.Exec `layout:"build"`
		Test  layout.Exec `layout:"test"`
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
	Root   layout.Dir                 `layout:"."`
	Config formats.TOMLFile[Settings] `layout:"settings.toml"`
	Hooks  layout.Dir                 `layout:"hooks"`
}
```

Static structure plus dynamic children:

```go
type Project struct {
	Config formats.YAMLFile[ProjectConfig] `layout:"project.yaml"`
}

type Monorepo struct {
	Root     layout.Dir             `layout:"."`
	Projects layout.Slot[*Project]  `layout:"projects"`
}
```

Managed tools next to data:

```go
type Environment struct {
	Root   layout.Dir                  `layout:"."`
	Env    formats.YAMLFile[EnvConfig] `layout:"env.yaml"`
	Deploy layout.Exec                 `layout:"bin/deploy"`
}
```
