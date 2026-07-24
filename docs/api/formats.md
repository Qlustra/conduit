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

### `EnvCodec`

```go
type EnvCodec struct{}
```

Description:

- codec that marshals and unmarshals managed `.env` content as `map[string]string`

Methods:

- `Marshal(map[string]string) ([]byte, error)`: encodes key/value content as normalized dotenv text with a trailing newline when content is present
- `Unmarshal([]byte) (map[string]string, error)`: decodes dotenv bytes through `github.com/joho/godotenv`

Notable behavior:

- accepts `godotenv`-compatible input syntax on load
- rewrites saved content into normalized managed output
- does not preserve comments, duplicate entries, original quoting, separator style, or original ordering
- does not evaluate interpolation or shell expressions

### `EnvPrecedence`

```go
type EnvPrecedence uint8
```

Description:

- selects whether process values or file values win when both provide the same key

Constants:

- `EnvProcessWins`
- `EnvFileWins`

### `EnvScope`

```go
type EnvScope uint8
```

Description:

- bitmask that controls which keys outside the defaults baseline may be admitted into a resolved environment

Constants:

- `EnvBaselineOnly`
- `EnvKeepProcess`
- `EnvKeepFile`
- `EnvKeepAll`

### `EnvResolveOptions`

```go
type EnvResolveOptions struct {
	Precedence EnvPrecedence
	Scope      EnvScope
}
```

Description:

- groups precedence and scope controls for env resolution helpers

### `EnvFile`

```go
type EnvFile struct{}
```

Description:

- typed file that embeds `layout.Format[map[string]string, EnvCodec]`

Exposed API:

- all promoted `layout.Format[map[string]string, EnvCodec]` methods
- all promoted `layout.File` methods from the embedded `layout.Format`
- `ResolveWithProcess(defaults, processEnv, opts) (map[string]string, bool)` to resolve cached file values together with a supplied process environment snapshot

### Functions

#### `EnvMap(entries)`

```go
func EnvMap(entries []string) (map[string]string, error)
```

Description:

- parses `KEY=VALUE` entries into a map with last-wins behavior for repeated keys
- returns an error for malformed entries or empty keys

#### `ResolveEnv(defaults, processEnv, fileEnv, opts)`

```go
func ResolveEnv(defaults, processEnv, fileEnv map[string]string, opts EnvResolveOptions) map[string]string
```

Description:

- resolves defaults, process values, and file values into one map
- defaults define the baseline key set and fallback values
- `opts.Precedence` selects the winner between process and file when both are present
- `opts.Scope` controls which keys outside the defaults baseline may be admitted

#### `EnvList(env)`

```go
func EnvList(env map[string]string) []string
```

Description:

- renders a deterministic sorted `[]string` of `KEY=VALUE` entries

`TextTemplate[C]` lives in the `layout` package. See [Layout API](layout.md) for its reference.
