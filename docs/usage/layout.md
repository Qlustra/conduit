# Layout usage

Conduit layouts are plain Go structs with `layout` tags. `Compose` walks the struct, resolves each tagged field relative to a root path, and binds the corresponding filesystem handle or typed file.

In code, that usually means:

- import `github.com/qlustra/conduit` for operations such as `Compose`
- import `github.com/qlustra/conduit/layout` for `Dir`, `File`, `Link`, `FileLink`, `DirLink`, `Exec`, `Slot[T]`, `FileSlot[T]`, `LinkSlot[T]`, and `TextTemplate[C]`
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
- `ParentPath()` returns the parent path string.
- `ParentDir()` returns the parent as a `Dir` handle.
- `RelTo(...)` and `JoinRelTo(...)` build the receiver path relative to another path-bearing node.
- `RelToPath(...)` and `JoinRelToPath(...)` do the same against a raw string base.
- `RelPathTo(...)` and `JoinRelPathTo(...)` reverse that direction and build a target path relative to the receiver.
- `DeclaredPath()` and `JoinDeclaredPath(...)` expose the node's own declared layout fragment.
- `ComposedBaseDir()`, `ComposedRelativePath()`, and `JoinComposedPath(...)` expose compose-base-relative paths when the handle belongs to a composed tree.
- `Exists()` reports whether the directory currently exists on disk.
- `Chown(uid, gid)` applies `os.Chown` to the directory path.
- `Join(...)` builds a descendant path.
- `List()` returns the directory's direct children.
- `ChangeTo()` changes the process working directory to that path.
- `Dir(name)` and `File(name)` derive child handles.
- `CopyToPath(path, opts)`, `CopyToDir(dir, opts)`, and `CopyIntoDir(parent, opts)` copy directory trees with explicit overwrite, symlink, and mode policy.
- `Ensure(ctx)` creates the directory tree.
- `Empty()` removes all children while preserving the directory itself. It rejects symlink roots and removes symlink children as entries.
- `DeleteIfExists()` removes the directory recursively when it exists.

### `File`

`File` is a stateless handle to a regular file path.

Useful methods:

- `Path()` and `Exists()`
- `Base()`, `Ext()`, and `Stem()` for path fragments
- `ParentPath()` and `ParentDir()` for filesystem parent lookup
- `RelTo(...)`, `JoinRelTo(...)`, `RelToPath(...)`, and `JoinRelToPath(...)` when the receiver should be made relative to a base
- `RelPathTo(...)` and `JoinRelPathTo(...)` when some target path should be made relative to the receiver
- `DeclaredPath()` and `JoinDeclaredPath(...)` for local declared layout fragments
- `ComposedBaseDir()`, `ComposedRelativePath()`, and `JoinComposedPath(...)` for compose-base-relative path fragments
- `ReadBytes()` and `ReadBytesIfExists()`
- `Chown(uid, gid)` for ownership changes
- `IsExecutable()` to check execute bits on regular files
- `Truncate(size)` to resize the file in place
- `AppendReader(src, dirMode, fileMode)` to stream any reader into the file in append mode
- `AppendBytes(data, dirMode, fileMode)`, `AppendString(content, dirMode, fileMode)`, `AppendFile(src, dirMode, fileMode)`, and `AppendFiles(dirMode, fileMode, srcs...)` for append and concat workflows
- `WriteBytes(data, dirMode, fileMode)`
- `CopyToPath(path, opts)`, `CopyToFile(dst, opts)`, and `CopyIntoDir(dir, opts)` for streamed file copies
- `Ensure(ctx)` to create the file and its parent directories
- `DeleteIfExists()`

Use `File` when you want raw bytes and do not need codec-backed state tracking.

Copy helpers use `layout.CopyOptions`. `layout.DefaultCopyOptions` preserves source modes, preserves symlinks as symlinks, and fails when the destination already exists. Switch `Overwrite` to `layout.CopyOverwriteReplace` when you want replacement semantics, and switch `Symlinks` to `layout.CopySymlinkFollow` or `layout.CopySymlinkReject` when preserving symlinks is not what you want.

### `Exec`

`Exec` is a `File` with executable semantics.

Useful methods:

- `Ensure(ctx)` and `EnsureExecutable(ctx)` create the file with executable permissions.
- `Base()`, `Ext()`, and `Stem()` are inherited from `File`.
- `RelTo(...)`, `JoinRelTo(...)`, `RelToPath(...)`, `JoinRelToPath(...)`, `RelPathTo(...)`, and `JoinRelPathTo(...)` are inherited from `File`.
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

### `Link`, `FileLink`, and `DirLink`

`Link` models a symlink node at its own path. `FileLink` and `DirLink` embed `Link` and add typed access to the resolved target path.

Useful methods on `Link`:

- `Path()` returns the link path itself.
- `Base()`, `Ext()`, and `Stem()` expose path fragments for the link path.
- `RelTo(...)`, `JoinRelTo(...)`, `RelToPath(...)`, `JoinRelToPath(...)`, `RelPathTo(...)`, and `JoinRelPathTo(...)` work like the other node types.
- `DeclaredPath()` and `JoinDeclaredPath(...)` expose the node's own declared layout fragment.
- `ComposedBaseDir()`, `ComposedRelativePath()`, and `JoinComposedPath(...)` expose compose-base-relative path fragments.
- `Exists()` reports whether a symlink exists at the path. It does not require the target to resolve.
- `Target()`, `MustTarget()`, `SetTarget(...)`, `SetDefaultTarget(...)`, `HasTarget()`, and `ClearTarget()` manage the in-memory target string.
- `ResolvedTargetPath()` resolves relative targets from the link's parent directory.
- `TargetExists()` and `IsDangling()` inspect the current in-memory target.
- `Load()`, `Discover()`, `Scan()`, `Sync(ctx)`, `Delete()`, `Unload()`, `DiskState()`, and `MemoryState()` follow the same state model used by typed files.

Useful methods on the typed wrappers:

- `FileLink.TargetFile()` / `MustTargetFile()`
- `DirLink.TargetDir()` / `MustTargetDir()`

Notable behavior:

- `Link` does not implement ensure semantics. `EnsureDeep` leaves declared links alone.
- `Load()` succeeds for dangling symlinks and loads the raw target string from `os.Readlink`.
- `Scan()` and `Load()` fail when the path exists but is not a symlink.
- `Sync(ctx)` manages only the symlink entry at the declared path. It never creates or validates the target payload.
- `FileLink` and `DirLink` describe how you intend to use the target handle; they do not change the fact that the node at `Path()` is a symlink.

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

Slot item names must identify one direct child only. Empty names, absolute paths, `.` / `..`, and names containing path separators are rejected.

The composed-path helpers return `ok == false` until a node has been attached through `Compose`. When they are available, the compose base is the same root that anchored the whole composed tree, not the nearest nested struct or slot item.

The declared-path helpers are different: they return the node's own declared layout fragment only. They do not reconstruct ancestor fragments. For example, a field declared as `layout:"build"` reports `build`, not `bin/build`, even when it lives inside a nested struct rooted at `layout:"bin"`.

The ordinary relative helpers are different again: they ignore composition metadata and declared layout metadata entirely. They just perform `filepath.Rel` and optional joining against another path-bearing node or a raw base path.

### `FileSlot[T]`

`FileSlot[T]` models repeated direct-child files under one directory:

```go
type Workspace struct {
	Configs layout.FileSlot[formats.YAMLFile[ServiceConfig]] `layout:"configs"`
}
```

Each key becomes a file path directly under the slot path:

```text
configs/
  api.yaml
  worker.yaml
```

Useful methods mirror `Slot[T]`, but with file semantics:

- `At(name)` composes and caches an item lazily at `slotRoot/<name>`.
- `Add(name, ctx)` ensures the slot root, composes the file-backed item, and ensures it.
- `Delete(name)` removes the child file from disk when it exists and evicts the cached item.
- `Require(name)` fails unless the child file already exists on disk.
- `LoadDeep(ctx)` and `DiscoverDeep(ctx)` enumerate direct child files and ignore subdirectories.

As with `Slot[T]`, `Entries()`, `All()`, `Len()`, and `Keys()` are cache-based only.

File slot item names follow the same direct-child restriction as `Slot[T]`.

### `LinkSlot[T]`

`LinkSlot[T]` models repeated direct-child symlink entries under one directory:

```go
type Workspace struct {
	Context layout.LinkSlot[layout.Link] `layout:"context"`
}
```

Each key becomes one symlink path directly under the slot path:

```text
context/
  README
  assets
```

Useful methods mirror `FileSlot[T]`, but with symlink-entry semantics:

- `At(name)` composes and caches a link item lazily at `slotRoot/<name>`.
- `Add(name, ctx)` ensures the slot root, composes the link item, and caches it. The link itself is still materialized by `Sync` or `SyncDeep`.
- `Delete(name)` removes the child symlink from disk when it exists and evicts the cached item.
- `Require(name)` fails unless a symlink already exists at the child path.
- `LoadDeep(ctx)` and `DiscoverDeep(ctx)` enumerate direct child symlink entries and ignore regular files and directories.

`LinkSlot[T]` is restricted to the built-in link family:

- `layout.Link`
- `layout.FileLink`
- `layout.DirLink`

Use `LinkSlot[layout.Link]` when targets may vary between files and directories.

Link slot item names follow the same direct-child restriction as `Slot[T]`.

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
