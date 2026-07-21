---
name: conduit
description: Use this skill when working with github.com/qlustra/conduit in Go projects. It teaches the library's contract-based filesystem model, the deep operation phases including `ValidateDeep`, how to work with directories, files, execs, links, slots, and typed files, and the path-safety and write-policy rules that matter when changing Conduit-backed layouts.
---

# Conduit

Conduit models a filesystem as Go structs with explicit state movement between disk and memory.

Treat these as separate phases:

1. `Compose(root, &layout)` binds paths to a struct.
2. `EnsureDeep` creates declared structure.
3. `DefaultDeep` seeds missing in-memory defaults for already composed or cached items.
4. `DiscoverDeep` discovers slot entries from disk and refreshes stateful disk knowledge without loading content.
5. `LoadDeep` reads stateful content and discovers slot entries from disk.
6. `RenderDeep` derives text content into memory for renderable template files.
7. `ValidateDeep` checks the already composed or cached layout without mutating it.
8. `SyncDeep` writes loaded or dirty syncable state back to disk.
9. `ScanDeep` refreshes disk-presence metadata without loading content.

Conduit does not reconcile disk and memory for you. There is no background sync and no merge policy. Discovery only happens when you explicitly ask for `DiscoverDeep` or `LoadDeep`.

## Core rules

- Always `Compose` before using any node or deep operation.
- `Compose` binds paths only. It does not touch the filesystem.
- `EnsureDeep` creates structure but does not load data.
- `DefaultDeep` applies defaults in memory only. It does not read or write disk state.
- `DiscoverDeep` discovers `Slot[T]`, `FileSlot[T]`, and `LinkSlot[T]` items that already exist on disk without loading content.
- `LoadDeep` reads syncable state such as typed files, text templates, and links, and discovers dynamic slot items that already exist on disk.
- `RenderDeep` derives text into memory only. It does not discover slots or write files.
- `ValidateDeep` checks the current composed or cached shape without creating, loading, or writing anything.
- `SyncDeep` only writes syncable nodes that currently hold a cached value or link target in memory.
- `ScanDeep` updates "present vs missing" knowledge only; it does not load bytes, target strings, or replace memory.
- `Slot[T]` discovery is asymmetric:
  `DiscoverDeep` discovers entries from disk and preserves unloaded cached state.
  `LoadDeep` discovers entries from disk.
  `DefaultDeep`, `RenderDeep`, `ScanDeep`, `ValidateDeep`, and `SyncDeep` only recurse into already cached entries.

## Layout declaration

Layouts are plain exported Go structs with `layout:"..."` tags.

Use:

- `layout:"."` for the current root
- relative paths for children
- `Slot[*T]` for repeated child directories under one directory
- `FileSlot[T]` for repeated direct-child files
- `LinkSlot[T]` for repeated direct-child symlink entries

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
- `DeleteIfExists(ctx)`

### `File`

Use `File` when you want raw bytes, not typed content tracking.

Common methods:

- `Path()`
- `Exists()`
- `ReadBytes()`
- `ReadBytesIfExists()`
- `WriteBytes(data, ctx)`
- `OpenRead(ctx, op)` / `OpenWrite(ctx, op)` and the other `Open*` variants
- `Append*`, `Concat*`, `Transform*`, `Hash`, and `HashHex`
- `Ensure(ctx)`
- `DeleteIfExists(ctx)`

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

### `Link`

`Link` is a stateful symlink node. It tracks a cached target string in memory using the same disk/memory-state model as typed files.

Common methods:

- `Target()` / `MustTarget()`
- `SetTarget(target)` / `SetDefaultTarget(target)`
- `ResolvedTargetPath()` / `TargetExists()` / `IsDangling()`
- `Load()` / `Discover()` / `Scan()`
- `Sync(ctx)` / `Delete(ctx)` / `Unload()`

Important rules:

- `Link` manages only the symlink entry at `Path()`.
- `Load()` succeeds for dangling symlinks because it reads the raw `os.Readlink` target.
- `Sync(ctx)` creates or rewrites the symlink entry itself. It does not create or validate the target payload.
- `EnsureDeep` does not materialize declared links.

## Path safety and writes

- `Context.PathSafetyPolicy` defaults to `PathSafetyRejectSymlinkParents`. Set `PathSafetyFollowSymlinks` only when following symlink parents is intentional.
- `OpenPolicy` controls whether `File.Open*` helpers require an existing file or add `os.O_CREATE`.
- `Context.WritePolicy` controls `File.WriteBytes`: `WriteDirect` rewrites in place, `WriteAtomicReplace` stages through a temp file and replaces atomically when the filesystem allows it.
- When using atomic replacement, `Context.TempFilePlacement` and `Context.TempDir` control where the temp file is staged.

## Typed files

The codec-backed typed file wrappers are:

- `formats.JSONFile[T]`
- `formats.YAMLFile[T]`
- `formats.TOMLFile[T]`

They expose the same `Format[T]` behavior:

- `Load() (bool, error)`
- `LoadOrInit(defaultValue)`
- `Read()` / `ReadIfExists()`
- `Get() (T, bool)` / `MustGet() T`
- `Set(value)` / `SetDefault(value)`
- `Write(value, ctx)`
- `Save(ctx)`
- `Sync(ctx)`
- `Discover()`
- `Scan()`
- `Clear()`
- `Unload()`
- `Delete(ctx)`
- `HasContent()`
- `DiskState()` / `MemoryState()`
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
- `Delete(ctx)` removes the file on disk and clears memory.

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
- `SetDefaultContext(ctx)` / `HasContext()` / `ClearContext()`
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
- `DiscoverDeep(ctx)` discovers child directories from disk and scans them without loading stateful content
- `LoadDeep(ctx)` discovers child directories from disk and loads them
- `DefaultDeep()`, `RenderDeep()`, `ScanDeep(ctx)`, `ValidateDeep(opts)`, and `SyncDeep(ctx)` recurse only into cached items

Use `Add` for explicit creation. Use `DiscoverDeep` when you want discovery without loading stateful content. Use `LoadDeep` when disk is authoritative and you want both discovery and content loading.

## `FileSlot[T]` and `LinkSlot[T]`

- `FileSlot[T]` models repeated direct-child files under one directory.
- `LinkSlot[T]` models repeated direct-child symlink entries under one directory.
- Their cache and traversal behavior mirrors `Slot[T]`, but discovery is type-specific:
  `FileSlot[T]` discovers regular files only.
  `LinkSlot[T]` discovers symlink entries only.
- `RenderDeep` and `DefaultDeep` recurse into cached `FileSlot[T]` items as well.
- `LinkSlot[T]` supports only the built-in link family: `layout.Link`, `layout.FileLink`, and `layout.DirLink`.

## Validation and reporting

- `ValidateDeep(target, opts)` is the non-mutating validation phase.
- `Context.Reporter` collects path-level results during deep operations such as ensure, load, discover, scan, and sync.
- `ValidateOptions.Reporter` does the same for validation.
- Use `conduit.Report` when you want an in-memory report implementation you can inspect after a pass.

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
- expecting `ValidateDeep` to discover or load children for you
- expecting `SyncDeep` to create uncached slot items automatically
- expecting `SyncDeep` to materialize a link that has no cached target
- assuming `Keys()` reflects the filesystem without a prior `DiscoverDeep` or `LoadDeep`
- using `MustGet()` before `Load`, `LoadOrInit`, or `Set`
- using `Save` when no content is loaded and expecting a no-op; use `Sync` for that

## Agent guidance

When modifying code that uses Conduit:

- identify whether disk or memory is authoritative for the current step
- keep the operation sequence explicit rather than collapsing it into helper magic
- prefer `EnsureDeep` for "declare structure", `DefaultDeep` for "seed missing memory", `DiscoverDeep` for "enumerate existing structure", `LoadDeep` for "read existing state", `RenderDeep` for "derive text into memory", `ValidateDeep` for "check current state", and `SyncDeep` for "persist current memory"
- use `Slot[T]` for keyed child directories, `FileSlot[T]` for keyed child files, and `LinkSlot[T]` for keyed child symlink entries
- use plain `File` for bytes, typed files for codec-backed state, `Link` for managed symlink entries, and `TextTemplate[C]` for derived text that should participate in the deep phase model
