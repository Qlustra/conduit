# Layout usage

A Conduit tree is a Go struct whose participating fields carry role and path tags. `Compile[T]` recognizes the declaration; `Structure.Compose` constructs a `Layout[T]` manager and binds its live tree to a filesystem root.

In code, that usually means:

- import `github.com/qlustra/conduit` for `Compile`, `Structure`, and `Layout` operations
- import `github.com/qlustra/conduit/layout` for `Dir`, `File`, `Link`, `FileLink`, `DirLink`, `Exec`, `Slot[T]`, `FileSlot[T]`, `LinkSlot[T]`, and `TextTemplate[C]`
- import `github.com/qlustra/conduit/formats` for `JSONFile[T]`, `YAMLFile[T]`, `TOMLFile[T]`, and `EnvFile`

## Defining a layout

Use `layout` for the carrier role and `path` for its local filesystem path:

```go
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Service struct {
	Root   layout.Dir                      `layout:"dir" path:"."`
	Config formats.YAMLFile[ServiceConfig] `layout:"content" path:"config.yaml"`
	Logs   layout.Dir                      `layout:"dir" path:"logs"`
}

type Workspace struct {
	Root     layout.Dir             `layout:"dir" path:"."`
	Services layout.Slot[*Service]  `layout:"directories" path:"services"`
}

structure, err := conduit.Compile[Workspace]()
if err != nil {
	return err
}
managed, err := structure.Compose("/srv/workspace")
if err != nil {
	return err
}
ws := managed.Tree()
```

`Structure.Compose` creates a fresh tree and allocates declared pointer paths. Composition does not touch the filesystem. Use `managed.Tree()` to access carriers and `managed.Ensure`, `managed.Load`, or the other [manager operations](operations.md) for whole-tree behavior.

The declaration roles are:

| Role | Carrier |
| --- | --- |
| `dir` | A carrier implementing `layout.DirAPI` |
| `file` | A carrier implementing `layout.FileAPI` |
| `exec` | A carrier implementing `layout.ExecAPI` |
| `content` | A built-in content carrier or embedding with its complete typed API |
| `group` | An explicitly participating nested struct or pointer to a struct |
| `directories` | A directory-member slot or embedding with its complete typed API |
| `documents` | A regular-file-member slot or embedding with its complete typed API |
| `links` | A symlink-member slot or embedding with its complete typed API |

The filesystem roles use native Go interfaces containing every exported carrier
method, including pointer-receiver methods. Extra methods and compatible
overrides are allowed; declared fields may be values or pointers. Embed the
carrier by value to keep its promoted methods ready after composition:

```go
type GoFile struct {
	layout.File
}

var _ layout.FileAPI = (*GoFile)(nil)

type ProgramLayout struct {
	Entrypoint GoFile `layout:"file,required" path:"main.go"`
}
```

For directory extensions, use an alias for the embedded field name. Embedding
`layout.Dir` directly names the field `Dir`, which hides the `Dir(name)` method:

```go
type Directory = layout.Dir
type CacheDir struct{ Directory }

var _ layout.DirAPI = (*CacheDir)(nil)
```

Content and collection families select each carrier's complete instantiated
interface through package-private witnesses. Admission retains the actual
content, render-context, and member types:

```go
type ConfigFile struct{ formats.YAMLFile[ServiceConfig] }
type Services struct{ layout.Slot[*Service] }

var _ layout.FormatAPI[ServiceConfig] = (*ConfigFile)(nil)
var _ layout.SlotAPI[*Service] = (*Services)(nil)

// The layout legend uses these acceptance declarations:
saillegend.AcceptContract[layout.FileAPI]()
saillegend.AcceptContractFamily[layout.ContentFamily]()
saillegend.AcceptContractFamily[layout.CollectionFamily]()
```

`FormatAPI[T]` is independent of the codec. `TextTemplateAPI[C]`, `LinkAPI`,
`FileLinkAPI`, and `DirLinkAPI` retain their additional operations. Each slot
family has its own API: `SlotAPI[T]`, `FileSlotAPI[T]`, and `LinkSlotAPI[T]`.
An incompatible override of `Get`, `Set`, `Entries`, or another carrier method
fails admission even if the wrapper still supplies `Path` and state or member
iteration. User-defined methods cannot replace the private family witness.

Use the role matching the collection's `ChildKind()`. Conduit checks it during
`Compile`, including nested groups. Typed member charts receive the same role
and complete-carrier checks during `Structure.Compose`. Overrides of
`ChildKind()` must be safe on a zero value and independent of configured state.
Composition allocates declared pointer paths; it does not initialize pointers
embedded inside carrier extensions. Use an explicit `Root` field at `path:"."`
when a member composes several participating fields, as in `Service` above.

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
- `Stat()` and `Lstat()` expose `os.Stat` and `os.Lstat` for the directory path.
- `Chown(uid, gid, ctx)` applies `os.Chown` to the directory path.
- `Chmod(mode, ctx)` and `Chtimes(atime, mtime, ctx)` apply mode and timestamp changes.
- `Open(ctx)` opens the directory as an `*os.File`.
- `OpenRoot(ctx)` opens the directory as an `*os.Root` for rooted filesystem operations.
- `Join(...)` builds a descendant path.
- `List()` returns the directory's direct children.
- `ChangeTo()` changes the process working directory to that path.
- `Dir(name)` and `File(name)` derive child handles.
- `CopyToPath(path, opts)`, `CopyToDir(dir, opts)`, and `CopyIntoDir(parent, opts)` copy directory trees with explicit overwrite, source-symlink, destination-path-safety, and mode policy.
- `Ensure(ctx)` creates the directory tree.
- `Empty(ctx)` removes all children while preserving the directory itself. It rejects symlink roots and removes symlink children as entries.
- `DeleteIfExists(ctx)` removes the directory recursively when it exists.

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
- `Stat()` and `Lstat()` expose `os.Stat` and `os.Lstat` for the file path
- `Chown(uid, gid, ctx)` for ownership changes
- `Chmod(mode, ctx)` and `Chtimes(atime, mtime, ctx)` for mode and timestamp changes
- `IsExecutable()` to check execute bits on regular files
- `Truncate(size, ctx)` to resize the file in place
- `OpenRead(ctx, op)`, `OpenWrite(ctx, op)`, `OpenRewrite(ctx, op)`, `OpenReadWrite(ctx, op)`, and append/rewrite variants for lower-level `*os.File` access
- `AppendReader(src, ctx)` to stream any reader into the file in append mode
- `AppendBytes(data, ctx)`, `AppendString(content, ctx)`, `AppendFile(src, ctx)`, and `AppendFiles(ctx, srcs...)` for append and concat workflows
- `ConcatReaders(ctx, opts, srcs...)`, `ConcatBytes(ctx, opts, srcs...)`, `ConcatStrings(ctx, opts, srcs...)`, and `ConcatFiles(ctx, opts, srcs...)` for buffered all-or-nothing rewrites from multiple inputs
- `TransformReader(ctx, src, fn)`, `TransformBytes(ctx, data, fn)`, `TransformString(ctx, data, fn)`, `TransformFile(ctx, src, fn)`, and `Transform(ctx, fn)` for buffered rewrites after a transform succeeds
- `Inspect(ctx, fn)` to stream file content through caller-defined inspection logic without buffering the whole file in Conduit
- `Match(ctx, fn)` to stream file content through caller-defined boolean matching logic
- `MatchIfExists(ctx, fn)` to treat a missing file as `false` instead of an error
- `InspectTokens(ctx, opts, fn)`, `MatchTokens(ctx, opts, fn)`, and `MatchTokensIfExists(ctx, opts, fn)` for token-oriented scanning through `bufio.Scanner`
- `InspectLines(ctx, fn)`, `MatchLines(ctx, fn)`, and `MatchLinesIfExists(ctx, fn)` for the common line-oriented case
- `Hash(ctx, h)` and `HashHex(ctx, h)` to digest file content through a supplied `hash.Hash`
- `Fingerprint()` to create a stateful observer for repeated metadata and checksum scans
- `WriteBytes(data, ctx)`
- `CopyToPath(path, opts)`, `CopyToFile(dst, opts)`, and `CopyIntoDir(dir, opts)` for streamed file copies
- `Ensure(ctx)` to create the file and its parent directories
- `DeleteIfExists(ctx)`

Use `File` when you want raw bytes and do not need codec-backed state tracking.

`layout.OpenPolicy` controls whether file open helpers require an existing file or add creation flags:

- `OpenExisting` adds no creation flags.
- `OpenOrCreate` adds `os.O_CREATE`.
- `OpenOrCreateExclusive` adds `os.O_CREATE | os.O_EXCL`.

When creation is enabled, the open helpers create parent directories with `Context.DirMode`. Existing symlink leaves are rejected, and symlink parents follow `Context.PathSafetyPolicy`.

The package-level helpers `layout.ConcatReaders`, `layout.ConcatBytes`, `layout.ConcatStrings`, `layout.TransformReader`, `layout.TransformBytes`, `layout.TransformString`, `layout.InspectReader`, `layout.InspectBytes`, `layout.InspectString`, `layout.MatchReader`, `layout.MatchBytes`, `layout.MatchString`, `layout.InspectTokensReader`, `layout.InspectTokensBytes`, `layout.InspectTokensString`, `layout.MatchTokensReader`, `layout.MatchTokensBytes`, `layout.MatchTokensString`, `layout.InspectLinesReader`, `layout.InspectLinesBytes`, `layout.InspectLinesString`, `layout.MatchLinesReader`, `layout.MatchLinesBytes`, `layout.MatchLinesString`, and the `layout.Hash*` functions are useful when you want the same processing behavior without writing through a `File` handle.

Use `TokenOptions` when you want a different `bufio.Scanner` split function or need to raise the maximum token size for long lines or other large tokens.

`File.Fingerprint()` gives you a stateful scan loop without making `File` itself stateful:

```go
cfg := layout.NewFile("config.yaml")
fp := cfg.Fingerprint()

if _, err := fp.ScanMetadata(); err != nil {
	return err
}

// ... later ...

if _, err := fp.Scan(); err != nil {
	return err
}

if fp.PresenceChanged() {
	// file appeared or disappeared
}

if fp.ChangedMetadata() {
	// size or mtime changed while the file stayed present
}

if fp.ChangedContent() {
	// content checksum changed
}

if fp.Changed() {
	// any metadata or content change
}
```

Mutating `File`, `Dir`, and `Exec` operations are type-strict: they expect the on-disk leaf to match the handle kind and reject symlink leaves instead of following them. `Link` is the type that intentionally manages symlink leaves.

`Context.PathSafetyPolicy` controls whether mutating operations reject symlink parents during path resolution. The default is `layout.PathSafetyRejectSymlinkParents`; set `layout.PathSafetyFollowSymlinks` only when following symlink parents is intentional.

Copy helpers use `layout.CopyOptions`. `layout.DefaultCopyOptions` preserves source modes, preserves source symlinks as symlinks, rejects symlink parents in destination paths, and fails when the destination already exists. Switch `Overwrite` to `layout.CopyOverwriteReplace` when you want replacement semantics, switch `Symlinks` to `layout.CopySymlinkFollow` or `layout.CopySymlinkReject` for source-tree behavior, and switch `PathSafetyPolicy` to `layout.PathSafetyFollowSymlinks` when destination symlink parents are intentional.

### `Exec`

`Exec` is a `File` with executable semantics.

Useful methods:

- `Ensure(ctx)` and `EnsureExecutable(ctx)` create the file with executable permissions.
- raw `File` helpers such as `Stat`, `Lstat`, `Chmod`, `Chtimes`, `Open*`, `Append*`, `Concat*`, `Transform*`, `Hash`, `HashHex`, `ReadBytes`, `WriteBytes`, and `CopyTo*` are promoted from the embedded `File`.
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
- `Lstat()` exposes `os.Lstat` for the symlink path.
- `Readlink()` exposes `os.Readlink` for the symlink path.
- `Target()`, `MustTarget()`, `SetTarget(...)`, `SetDefaultTarget(...)`, `HasTarget()`, and `ClearTarget()` manage the in-memory target string.
- `ResolvedTargetPath()` resolves relative targets from the link's parent directory.
- `TargetExists()` and `IsDangling()` inspect the current in-memory target.
- `Load()`, `Discover()`, `Scan()`, `Sync(ctx)`, `Delete(ctx)`, `Unload()`, `DiskState()`, and `MemoryState()` follow the same state model used by typed files.

Useful methods on the typed wrappers:

- `FileLink.TargetFile()` / `MustTargetFile()`
- `DirLink.TargetDir()` / `MustTargetDir()`

Notable behavior:

- `Link` has no Ensure hook. Set its target, then use Sync to materialize the link entry.
- `Load()` succeeds for dangling symlinks and loads the raw target string from `os.Readlink`.
- `Scan()` and `Load()` fail when the path exists but is not a symlink.
- `Sync(ctx)` manages only the symlink entry at the declared path. It never creates or validates the target payload.
- `FileLink` and `DirLink` describe how you intend to use the target handle; they do not change the fact that the node at `Path()` is a symlink.

### Typed files

`formats.JSONFile[T]`, `formats.YAMLFile[T]`, `formats.TOMLFile[T]`, and `formats.EnvFile` are codec-backed files that embed `layout.Format[T, C]`.

They behave like regular layout nodes, but also keep typed content in memory:

```go
type App struct {
	Config formats.JSONFile[map[string]any] `layout:"content" path:"config.json"`
}
```

See [Formats usage](formats.md) for the full content API.

### `Slot[T]`

`Slot[T]` models repeated child structures under one directory:

```go
type Service struct {
	Config formats.YAMLFile[ServiceConfig] `layout:"content" path:"config.yaml"`
}

type Workspace struct {
	Services layout.Slot[*Service] `layout:"directories" path:"services"`
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

- `ComposeMember(name)` composes and caches an item lazily.
- `MustComposeMember(name)` is the panicking version of `ComposeMember`.
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

Slots provide local `Ensure(ctx)` for their containing root. The owning `conduit.Layout` performs whole-tree operations over cached members. `ComposeMember` changes memory only.

Slots must be configured by `Structure.Compose` before constructing missing members. Their member composer retains the declared member chart and composition root, so each member follows the same role and path vocabulary. See [Layout manager](operations.md#layout-manager) for setup.

Slot item names must identify one direct child only. Empty names, absolute paths, `.` / `..`, and names containing path separators are rejected.

The composed-path helpers return `ok == false` until a node has been attached through `Structure.Compose`. When they are available, the compose base is the same root that anchored the whole composed tree, not the nearest nested struct or slot item.

The declared-path helpers are different: they return the node's own declared layout fragment only. They do not reconstruct ancestor fragments. For example, an executable declared with `path:"build"` reports `build`, not `bin/build`, even inside a group with `path:"bin"`.

The ordinary relative helpers are different again: they ignore composition metadata and declared layout metadata entirely. They just perform `filepath.Rel` and optional joining against another path-bearing node or a raw base path.

### `FileSlot[T]`

`FileSlot[T]` models repeated direct-child files under one directory:

```go
type Workspace struct {
	Configs layout.FileSlot[*formats.YAMLFile[ServiceConfig]] `layout:"documents" path:"configs"`
}
```

Each key becomes a file path directly under the slot path:

```text
configs/
  api.yaml
  worker.yaml
```

Useful methods mirror `Slot[T]`, but with file semantics:

- `ComposeMember(name)` composes and caches an item lazily at `slotRoot/<name>`.
- `Delete(name)` removes the child file from disk when it exists and evicts the cached item.
- `Require(name)` fails unless the child path already exists on disk as a regular file.

As with `Slot[T]`, `Entries()`, `All()`, `Len()`, and `Keys()` are cache-based only. The owning `Layout.Load` and `Layout.Discover` enumerate regular files, excluding subdirectories and symlinks.

File slot item names follow the same direct-child restriction as `Slot[T]`.

### `LinkSlot[T]`

`LinkSlot[T]` models repeated direct-child symlink entries under one directory:

```go
type Workspace struct {
	Context layout.LinkSlot[*layout.Link] `layout:"links" path:"context"`
}
```

Each key becomes one symlink path directly under the slot path:

```text
context/
  README
  assets
```

Useful methods mirror `FileSlot[T]`, but with symlink-entry semantics:

- `ComposeMember(name)` composes and caches a link item lazily at `slotRoot/<name>`.
- `Delete(name)` removes the child symlink from disk when it exists and evicts the cached item.
- `Require(name)` fails unless a symlink already exists at the child path.

The owning `Layout.Load` and `Layout.Discover` enumerate symlink entries, excluding regular files and directories. `Layout.Sync` materializes links once their targets are cached; Ensure excludes link entries.

`LinkSlot[T]` is restricted to the built-in link family:

- `layout.Link` or `*layout.Link`
- `layout.FileLink` or `*layout.FileLink`
- `layout.DirLink` or `*layout.DirLink`

Use `LinkSlot[*layout.Link]` when targets may vary between files and directories.
Pointer members let operations update the cached instances directly. Value
members remain available when copy semantics are desired.

Link slot item names follow the same direct-child restriction as `Slot[T]`.

## Composition rules

Only exported, explicitly tagged fields participate. Untagged fields and anonymous embeddings are not inferred from their carrier types. Declare nested structure with `layout:"group"` and a path for its children.

`path:"."` binds a carrier to its containing root. Relative paths resolve from the nearest group or slot member root. Absolute paths are rejected, and a path that escapes the original composition root fails composition.

```go
type Tooling struct {
	Scripts struct {
		Build layout.Exec `layout:"exec" path:"build"`
		Test  layout.Exec `layout:"exec" path:"test"`
	} `layout:"group" path:"bin"`
}
```

After composing `Tooling` at `/workspace`, the executable paths are `/workspace/bin/build` and `/workspace/bin/test`. Tag an embedded struct as a group when it should participate too.

## Common layout patterns

Static structure:

```go
type Repo struct {
	Root   layout.Dir                 `layout:"dir" path:"."`
	Config formats.TOMLFile[Settings] `layout:"content" path:"settings.toml"`
	Hooks  layout.Dir                 `layout:"dir" path:"hooks"`
}
```

Static structure plus dynamic children:

```go
type Project struct {
	Config formats.YAMLFile[ProjectConfig] `layout:"content" path:"project.yaml"`
}

type Monorepo struct {
	Root     layout.Dir             `layout:"dir" path:"."`
	Projects layout.Slot[*Project]  `layout:"directories" path:"projects"`
}
```

Managed tools next to data:

```go
type Environment struct {
	Root   layout.Dir                  `layout:"dir" path:"."`
	Env    formats.YAMLFile[EnvConfig] `layout:"content" path:"env.yaml"`
	Deploy layout.Exec                 `layout:"exec" path:"bin/deploy"`
}
```
