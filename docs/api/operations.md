# Operations API

This page documents the exported package-level operations and configuration types.

## Variables

### `DefaultContext`

Type:

```go
var DefaultContext conduit.Context
```

Default value:

```go
conduit.Context{
	DirMode:  0o755,
	FileMode: 0o644,
	ExecMode: 0o755,
}
```

Notable behavior:

- used as the default permission set in most examples
- `ExecMode` is applied by `Exec` operations

## Types

### `Context`

```go
type Context struct {
	DirMode  os.FileMode
	FileMode os.FileMode
	ExecMode os.FileMode
}
```

Fields:

- `DirMode`: mode used when creating directories
- `FileMode`: mode used when creating regular files
- `ExecMode`: mode used when creating or ensuring `Exec` files

Notable behavior:

- when `ExecMode` is zero, `Exec` falls back to `FileMode` and adds execute bits automatically

## Functions

### `Compose`

```go
func Compose(root string, target any) error
```

Description:

- binds a layout struct to a filesystem root by assigning resolved paths to tagged fields

Arguments:

- `root`: base path for the layout
- `target`: non-nil pointer to a struct

Notable behavior:

- does not touch the filesystem
- returns an error if `target` is nil, not a pointer, or does not point to a struct
- allocates tagged pointer-to-struct fields as needed
- only fields tagged with `layout:"..."` are composed

### `EnsureDeep`

```go
func EnsureDeep(target any, ctx Context) error
```

Description:

- recursively materializes declared structure on disk

Arguments:

- `target`: composed struct or node tree
- `ctx`: directory and file permission settings

Notable behavior:

- creates directories, regular files, and executable files as required by the layout
- only ensures cached `Slot[T]` items
- does not load typed file content
- does not discover new slot items from disk
- returns an error if `target` is nil

### `LoadDeep`

```go
func LoadDeep(target any, ctx Context) error
```

Description:

- recursively loads typed content from disk into memory

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep loaders

Notable behavior:

- loads `Format`-backed files such as `JSONFile[T]`, `YAMLFile[T]`, and `TOMLFile[T]`
- discovers `Slot[T]` entries by reading child directories from disk
- does not create missing files
- leaves uncached missing slot entries undiscovered until they exist on disk
- returns an error if `target` is nil

### `SyncDeep`

```go
func SyncDeep(target any, ctx Context) error
```

Description:

- recursively writes loaded or dirty in-memory content back to disk

Arguments:

- `target`: composed struct or node tree
- `ctx`: directory and file permission settings

Notable behavior:

- ensures parent structure before syncing cached content
- only writes typed files that currently have content loaded in memory
- only syncs cached `Slot[T]` items
- does not delete files or directories
- returns an error if `target` is nil

### `ScanDeep`

```go
func ScanDeep(target any, ctx Context) error
```

Description:

- recursively refreshes disk-presence metadata for composed items

Arguments:

- `target`: composed struct or node tree
- `ctx`: passed through to deep scanners

Notable behavior:

- updates disk state without loading file content
- preserves current in-memory values and memory state
- only scans cached `Slot[T]` items
- does not discover new slot entries from disk
- returns an error if `target` is nil
