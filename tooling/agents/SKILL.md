---
name: conduit
description: Use this skill when working with github.com/qlustra/conduit in Go projects. It teaches the library's contract-based filesystem model, how to declare layouts with `layout` tags, when to use `Compose`, `EnsureDeep`, `DefaultDeep`, `DiscoverDeep`, `LoadDeep`, `RenderDeep`, `SyncDeep`, and `ScanDeep`, and how to work safely with `layout.Dir`, `layout.File`, `layout.Exec`, `layout.Slot[T]`, typed files such as `formats.YAMLFile[T]`, `formats.JSONFile[T]`, and `formats.TOMLFile[T]`, and derived text via `layout.TextTemplate[C]`.
---

# Conduit

Conduit models a filesystem as Go structs with explicit state movement between disk and memory.

Treat these as separate phases:

1. `Compose(root, &layout)` binds paths to a struct.
2. `EnsureDeep` creates declared structure.
3. `DefaultDeep` seeds missing in-memory defaults for already composed or cached items.
4. `DiscoverDeep` discovers slot entries from disk without loading typed file content.
5. `LoadDeep` reads typed file content and discovers slot entries from disk.
6. `RenderDeep` derives text content into memory for renderable template files.
7. `SyncDeep` writes loaded or dirty typed or rendered content back to disk.
8. `ScanDeep` refreshes disk-presence metadata without loading content.

Conduit does not reconcile disk and memory for you. There is no background sync and no merge policy. Discovery only happens when you explicitly ask for `DiscoverDeep` or `LoadDeep`.

## Core rules

- Always `Compose` before using any node or deep operation.
- `Compose` binds paths only. It does not touch the filesystem.
- `EnsureDeep` creates structure but does not load data.
- `DefaultDeep` applies defaults in memory only. It does not read or write disk state.
- `DiscoverDeep` discovers `Slot[T]` items that already exist on disk without loading typed files.
- `LoadDeep` reads typed files and discovers `Slot[T]` items that already exist on disk.
- `RenderDeep` derives text into memory only. It does not discover slots or write files.
- `SyncDeep` only writes typed files that currently hold content in memory.
- `ScanDeep` updates "present vs missing" knowledge only; it does not load bytes or replace memory.
- `Slot[T]` discovery is asymmetric:
  `DiscoverDeep` discovers entries from disk and preserves unloaded typed-file memory.
  `LoadDeep` discovers entries from disk.
  `DefaultDeep`, `RenderDeep`, `ScanDeep`, and `SyncDeep` only recurse into already cached entries.

## Layout declaration

Layouts are plain exported Go structs with `layout:"..."` tags.

Use:

- `layout:"."` for the current root
- relative paths for children
- `Slot[*T]` for repeated child layouts under one directory

Example:

```go
type AppConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type App struct {
	Root   layout.Dir                  `layout:"."`
	Config formats.YAMLFile[AppConfig] `layout:"config.yaml"`
	Logs   layout.Dir                  `layout:"logs"`
	Run    layout.Exec                 `layout:"bin/run"`
}

type Workspace struct {
	Root layout.Dir         `layout:"."`
	Apps layout.Slot[*App]  `layout:"apps"`
}

var ws Workspace
err := conduit.Compose("/workspace", &ws)
```

Composition rules worth remembering:

- `target` must be a non-nil pointer to a struct.
- only exported fields with `layout` tags are composed.
- tagged pointer-to-struct fields are allocated automatically.
- nested structs and anonymous embedded fields recurse naturally.

## Public node types

### `Dir`

Use `Dir` for directory handles. It is stateless apart from its bound path.

Common methods:

- `Path()`
- `Exists()`
- `Join(...)`
- `Dir(name)`
- `File(name)`
- `Ensure(ctx)`
- `DeleteIfExists()`

### `File`

Use `File` when you want raw bytes, not typed content tracking.

Common methods:

- `Path()`
- `Exists()`
- `ReadBytes()`
- `ReadBytesIfExists()`
- `WriteBytes(data, dirMode, fileMode)`
- `Ensure(ctx)`
- `DeleteIfExists()`

### `Exec`

`Exec` is a managed executable file. It behaves like `File`, but can also run the file.

Common methods:

- `Ensure(ctx)` / `EnsureExecutable(ctx)`
- `IsExecutable()`
- `Command(ctx, opts)`
- `Run(ctx, opts)`
- `Output(ctx, opts)`
- `CombinedOutput(ctx, opts)`

Use `RunOptions.Interpreter` when the file should be invoked through `sh`, `python3`, etc.

## Typed files

The stateful types are:

- `formats.JSONFile[T]`
- `formats.YAMLFile[T]`
- `formats.TOMLFile[T]`
- `layout.TextTemplate[C]`

The typed files expose the same `Format[T]` behavior:

- `Load() (bool, error)`
- `LoadOrInit(defaultValue)`
- `Get() (T, bool)` / `MustGet() T`
- `Set(value)`
- `SetDefault(value)`
- `Save(ctx)`
- `Sync(ctx)`
- `Discover()`
- `Scan()`
- `Clear()`
- `Unload()`
- `Delete()`
- `HasContent()`
- `HasBeenLoaded()`
- `IsDirty()`

The important mental model is two independent axes:

- disk state: unknown, missing, present
- memory state: unknown, loaded, synced, dirty

High-value behavioral rules:

- `Set` changes memory only and marks it dirty.
- `Load` is authoritative for memory.
  If the file exists, memory is replaced from disk.
  If the file is missing, cached content is cleared.
- `LoadOrInit(default)` is not a write.
  If the file is missing, the default lives only in memory until `Save`, `Sync`, or `SyncDeep`.
- `Save` fails if no content is loaded.
- `Sync` is a no-op if no content is loaded.
- `Scan` preserves memory and only refreshes disk knowledge.
- `Discover` has the same typed-file effect as `Scan`; the distinction shows up during deep traversal.
- `Delete` removes the file on disk and clears memory.

Choose format by consumer:

- `JSONFile[T]` for machine-oriented artifacts
- `YAMLFile[T]` for hand-edited operational config
- `TOMLFile[T]` for settings-style files

Use `SetDefault(value)` inside `Default() error` implementations when you want to seed missing typed content without overwriting existing memory.

## `TextTemplate[C]`

`TextTemplate[C]` is the stateful raw-text counterpart used for fully derived text artifacts.

Useful methods:

- all string-content methods analogous to `Format[string]`
- `SetContext(ctx)` / `GetContext()` / `MustContext()`
- `SetDefaultContext(ctx)`
- `RenderTemplate(tpl)`
- `SetRendered(value)`

Built-in render contracts:

- `Templatable`: implement `Template() string` and let `RenderDeep` use the built-in `text/template` path
- `Renderable`: implement `Render() (string, error)` and `SetRendered(string)` for custom rendering
- if a type implements both, `Renderable` takes precedence over `Templatable`

Use `TextTemplate[C]` when the file is a derived artifact. Keep rendering memory-only until `SyncDeep`.

## `Slot[T]`

`Slot[T]` models repeated child layouts under one directory.

Example:

```go
type Workspace struct {
	Apps layout.Slot[*App] `layout:"apps"`
}
```

Each key becomes a child root like `apps/<name>`.

Important methods:

- `At(name)` lazily composes and caches an item
- `MustAt(name)` panics on error
- `Add(name, ctx)` creates the child root on disk, composes it, ensures its declared structure, and caches it
- `Require(name)` only succeeds if the child directory already exists
- `Get(name)` returns cached items only
- `Keys()` returns sorted cached keys only
- `DiscoverDeep(ctx)` discovers child directories from disk and scans them without loading typed files
- `LoadDeep(ctx)` discovers child directories from disk and loads them
- `DefaultDeep()`, `RenderDeep()`, `ScanDeep(ctx)`, and `SyncDeep(ctx)` recurse only into cached items

Use `Add` for explicit creation. Use `DiscoverDeep` when you want discovery without loading typed content. Use `LoadDeep` when disk is authoritative and you want both discovery and content loading.

## Canonical workflows

### Bootstrap a new tree

```go
var ws Workspace
if err := conduit.Compose("/workspace", &ws); err != nil {
	return err
}
if err := conduit.EnsureDeep(&ws, conduit.DefaultContext); err != nil {
	return err
}

app, err := ws.Apps.Add("billing", conduit.DefaultContext)
if err != nil {
	return err
}

if err := app.Config.LoadOrInit(AppConfig{Name: "billing", Port: 8080}); err != nil {
	return err
}

return conduit.SyncDeep(&ws, conduit.DefaultContext)
```

### Default, render, persist

```go
type ReadmeContext struct {
	Name string
}

type ReadmeFile struct {
	layout.TextTemplate[ReadmeContext]
}

func (f *ReadmeFile) Default() error {
	f.SetDefaultContext(ReadmeContext{Name: "billing"})
	return nil
}

func (f *ReadmeFile) Template() string {
	return "# {{ .Name }}\n"
}

if err := conduit.DefaultDeep(&ws); err != nil {
	return err
}
if err := conduit.RenderDeep(&ws); err != nil {
	return err
}
return conduit.SyncDeep(&ws, conduit.DefaultContext)
```

### Load, edit, persist

```go
var ws Workspace
if err := conduit.Compose("/workspace", &ws); err != nil {
	return err
}
if err := conduit.LoadDeep(&ws, conduit.DefaultContext); err != nil {
	return err
}

app := ws.Apps.MustAt("billing")
cfg := app.Config.MustGet()
cfg.Port = 9000
app.Config.Set(cfg)

return conduit.SyncDeep(&ws, conduit.DefaultContext)
```

### Observe presence without replacing memory

```go
_, err := app.Config.Scan()
```

Use `Scan` / `ScanDeep` when you need existence information without loading or overwriting current in-memory state for already cached items. Use `DiscoverDeep` when you also need slot discovery.

## Common mistakes to avoid

- expecting `Compose` to create files or directories
- expecting `EnsureDeep` to discover `Slot[T]` entries from disk
- expecting `LoadOrInit` to write defaults immediately
- expecting `DefaultDeep` to read from disk or discover slots
- expecting `RenderDeep` to write files immediately
- expecting `SyncDeep` to create uncached slot items automatically
- assuming `Keys()` reflects the filesystem without a prior `DiscoverDeep` or `LoadDeep`
- using `MustGet()` before `Load`, `LoadOrInit`, or `Set`
- using `Save` when no content is loaded and expecting a no-op; use `Sync` for that

## Agent guidance

When modifying code that uses Conduit:

- identify whether disk or memory is authoritative for the current step
- keep the operation sequence explicit rather than collapsing it into helper magic
- prefer `EnsureDeep` for "declare structure", `DefaultDeep` for "seed missing memory", `DiscoverDeep` for "enumerate existing structure", `LoadDeep` for "read existing state", `RenderDeep` for "derive text into memory", and `SyncDeep` for "persist current memory"
- use `Slot[T]` only when children are keyed directories with the same layout shape
- use plain `File` for bytes, typed files for codec-backed state, and `TextTemplate[C]` for derived text that should participate in the deep phase model
