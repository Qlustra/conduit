# Formats API

This page documents the exported `github.com/qlustra/conduit/formats` package.

The package exports concrete codec types and concrete typed-file wrappers. The
generic `layout.Codec[T]` and `layout.Format[T, C]` types live in the
`github.com/qlustra/conduit/layout` package.

## Types

### `JSONCodec[T]`

```go
type JSONCodec[T any] struct{}
```

Description:

- codec that marshals and unmarshals typed values as indented JSON

Methods:

- `Marshal(T) ([]byte, error)`: encodes a typed value as indented JSON and appends a trailing newline
- `Unmarshal([]byte) (T, error)`: decodes JSON bytes into a typed value

### `JSONFile[T]`

```go
type JSONFile[T any] struct{}
```

Description:

- typed file that embeds `layout.Format[T, JSONCodec[T]]`

Exposed API:

- all promoted `layout.Format[T, JSONCodec[T]]` methods
- all promoted `layout.File` methods from the embedded `layout.Format`

Notable behavior:

- writes indented JSON
- appends a trailing newline on marshal
- uses `layout.Context`, `layout.ResultCode`, `layout.DiskState`, and `layout.MemoryState` through the embedded `layout.Format`

### `YAMLCodec[T]`

```go
type YAMLCodec[T any] struct{}
```

Description:

- codec that marshals and unmarshals typed values as YAML

Methods:

- `Marshal(T) ([]byte, error)`: encodes a typed value as YAML
- `Unmarshal([]byte) (T, error)`: decodes YAML bytes into a typed value

### `YAMLFile[T]`

```go
type YAMLFile[T any] struct{}
```

Description:

- typed file that embeds `layout.Format[T, YAMLCodec[T]]`

Exposed API:

- all promoted `layout.Format[T, YAMLCodec[T]]` methods
- all promoted `layout.File` methods from the embedded `layout.Format`

Notable behavior:

- uses `gopkg.in/yaml.v3` for marshal and unmarshal

### `TOMLCodec[T]`

```go
type TOMLCodec[T any] struct{}
```

Description:

- codec that marshals and unmarshals typed values as TOML

Methods:

- `Marshal(T) ([]byte, error)`: encodes a typed value as TOML
- `Unmarshal([]byte) (T, error)`: decodes TOML bytes into a typed value

### `TOMLFile[T]`

```go
type TOMLFile[T any] struct{}
```

Description:

- typed file that embeds `layout.Format[T, TOMLCodec[T]]`

Exposed API:

- all promoted `layout.Format[T, TOMLCodec[T]]` methods
- all promoted `layout.File` methods from the embedded `layout.Format`

Notable behavior:

- uses `github.com/pelletier/go-toml/v2` for marshal and unmarshal

`TextTemplate[C]` lives in the `layout` package. See [Layout API](layout.md) for its reference.
