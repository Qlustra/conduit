# Formats API

This page documents the exported typed-file API.

## Types

### `Codec[T]`

```go
type Codec[T any] interface {
	Marshal(T) ([]byte, error)
	Unmarshal([]byte) (T, error)
}
```

Description:

- codec contract used by `Format[T]`

Methods:

- `Marshal(T) ([]byte, error)`: encodes a typed value into bytes
- `Unmarshal([]byte) (T, error)`: decodes bytes into a typed value

### `Format[T]`

```go
type Format[T any] struct{}
```

Description:

- typed file with in-memory content tracking and disk/memory state metadata

Inherited file methods:

- `Path() string`
- `Exists() bool`
- `WriteBytes(data []byte, dirMode os.FileMode, fileMode os.FileMode) error`
- `ReadBytes() ([]byte, error)`
- `ReadBytesIfExists() ([]byte, bool, error)`
- `DeleteIfExists() error`
- `Ensure(ctx Context) error`

Content methods:

- `Get() (T, bool)`: returns the cached value if present
- `MustGet() T`: returns the cached value or panics when no value is loaded
- `Set(value T)`: replaces cached content and marks memory state dirty
- `SetDefault(value T) bool`: stores a default value only when no content is currently cached
- `Clear()`: clears cached content and resets memory state to unknown
- `Delete() error`: removes the file from disk if present, clears cached content, and marks disk state missing
- `Write(value T, ctx Context) error`: marshals and writes a supplied value directly
- `Read() (T, error)`: reads and unmarshals from disk
- `ReadIfExists() (T, bool, error)`: reads and unmarshals when the file exists
- `LoadOrInit(defaultValue T) error`: loads existing content or stores a default value in memory when missing
- `Save(ctx Context) error`: writes the cached value and marks memory state synced
- `Load() (bool, error)`: loads content into memory and reports whether the file existed
- `Discover()`: refreshes disk-state metadata without replacing in-memory content and returns the observed state value
- `HasContent() bool`: reports whether a value is currently cached
- `Unload()`: clears cached content and resets memory state to unknown
- `Sync(ctx Context) error`: writes cached content when present, otherwise no-op
- `DiskState()`: returns current disk-state metadata
- `MemoryState()`: returns current memory-state metadata
- `HasKnownDiskState() bool`: reports whether disk state is not unknown
- `WasObservedOnDisk() bool`: reports whether the last known disk state is present
- `HasBeenLoaded() bool`: reports whether memory state has reached loaded, synced, or dirty
- `IsDirty() bool`: reports whether memory state is dirty
- `Scan()`: refreshes disk-state metadata without replacing in-memory content and returns the observed state value

Notable behavior:

- `Set` does not write to disk
- `SetDefault` returns `false` and leaves content unchanged when content is already cached
- `Save` fails when no content is loaded
- `Sync` is a no-op when no content is loaded
- `Load` on a missing file clears cached content, sets disk state to missing, and resets memory state to unknown
- `Discover` updates disk state but preserves current cached content and memory state
- `Scan` only updates disk state; it preserves current cached content and memory state
- `MustGet` panics when content is absent
- the package does not currently re-export the state enum types or constants directly

### `JSONFile[T]`

```go
type JSONFile[T any] struct{}
```

Description:

- typed file using JSON encoding

Exposed API:

- all `Format[T]` methods
- all inherited `File` methods from embedded `Format[T]`

Notable behavior:

- writes indented JSON
- appends a trailing newline on marshal

### `YAMLFile[T]`

```go
type YAMLFile[T any] struct{}
```

Description:

- typed file using YAML encoding

Exposed API:

- all `Format[T]` methods
- all inherited `File` methods from embedded `Format[T]`

Notable behavior:

- uses `gopkg.in/yaml.v3` for marshal and unmarshal

### `TOMLFile[T]`

```go
type TOMLFile[T any] struct{}
```

Description:

- typed file using TOML encoding

Exposed API:

- all `Format[T]` methods
- all inherited `File` methods from embedded `Format[T]`

Notable behavior:

- uses `github.com/pelletier/go-toml/v2` for marshal and unmarshal

### `TextTemplate[C]`

```go
type TextTemplate[C any] struct{}
```

Description:

- stateful raw-text file with cached render context

Exposed API:

- all string-content operations analogous to `Format[string]`
- all inherited `File` methods from embedded raw-text file state

Content methods:

- `Get() (string, bool)`: returns the cached text if present
- `MustGet() string`: returns the cached text or panics when no text is loaded
- `Set(value string)`: replaces cached text and marks memory state dirty
- `Clear()`: clears cached text and resets memory state to unknown
- `Delete() error`: removes the file from disk if present, clears cached text, and marks disk state missing
- `Write(value string, ctx Context) error`: writes a supplied string directly
- `Read() (string, error)`: reads raw text from disk
- `ReadIfExists() (string, bool, error)`: reads raw text when the file exists
- `LoadOrInit(defaultValue string) error`: loads existing text or stores a default value in memory when missing
- `Save(ctx Context) error`: writes the cached text and marks memory state synced
- `Load() (bool, error)`: loads text into memory and reports whether the file existed
- `Discover()`: refreshes disk-state metadata without replacing in-memory content and returns the observed state value
- `HasContent() bool`: reports whether text is currently cached
- `Unload()`: clears cached text and resets memory state to unknown
- `Sync(ctx Context) error`: writes cached text when present, otherwise no-op
- `DiskState()`: returns current disk-state metadata
- `MemoryState()`: returns current memory-state metadata
- `HasKnownDiskState() bool`: reports whether disk state is not unknown
- `WasObservedOnDisk() bool`: reports whether the last known disk state is present
- `HasBeenLoaded() bool`: reports whether memory state has reached loaded, synced, or dirty
- `IsDirty() bool`: reports whether memory state is dirty
- `Scan()`: refreshes disk-state metadata without replacing in-memory content and returns the observed state value

Context methods:

- `SetContext(ctx C)`: replaces cached render context
- `SetDefaultContext(ctx C) bool`: stores default render context only when no context is currently cached
- `GetContext() (C, bool)`: returns cached render context if present
- `MustContext() C`: returns cached render context or panics when unset
- `HasContext() bool`: reports whether render context is currently cached
- `ClearContext()`: clears cached render context
- `RenderTemplate(tpl string) (string, error)`: executes template text against the current render context using the built-in `text/template` renderer
- `SetRendered(value string)`: stores rendered text in the file's cached content

Notable behavior:

- persists raw text without a structured codec
- mirrors the same disk/memory state model as `Format[T]`
- resets cached render context when `Compose` rebinds the file path
- `SetDefaultContext` returns `false` and leaves context unchanged when context is already cached
- provides the built-in templated render path used by `Templatable`
- leaves custom render validation and semantics to the user-defined `Render()` implementation
