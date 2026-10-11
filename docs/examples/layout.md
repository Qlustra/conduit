# Layout examples

These examples model filesystem trees using explicit role and path tags. Import `conduit`, `layout`, and `formats` from `github.com/qlustra/conduit`.

Compile each tree type and compose it before calling its slot APIs. For the Workspace below:

```go
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

Composition configures member construction and binds paths in memory. Use `managed.Ensure`, `managed.Load`, and the other Layout methods for whole-tree operations.

## Multi-service workspace

This is a common pattern for local orchestration, test fixtures, or generated environments.

```go
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Service struct {
	Root    layout.Dir                      `layout:"dir" path:"."`
	Config  formats.YAMLFile[ServiceConfig] `layout:"content" path:"config.yaml"`
	Logs    layout.Dir                      `layout:"dir" path:"logs"`
	Runner  layout.Exec                     `layout:"exec" path:"bin/run"`
}

type Workspace struct {
	Root     layout.Dir            `layout:"dir" path:"."`
	Services layout.Slot[*Service] `layout:"directories" path:"services"`
}
```

On disk, the layout looks like:

```text
workspace/
  services/
    api/
      config.yaml
      logs/
      bin/
        run
    worker/
      config.yaml
      logs/
      bin/
        run
```

Why this works well:

- `Slot[*Service]` keeps each service isolated under its own root
- `Dir`, `YAMLFile`, and `Exec` let one service describe both data and tooling
- the same struct works for bootstrap, load, and sync flows

## Deployment environment with static and dynamic parts

```go
type AppManifest struct {
	Image string `json:"image"`
	Tag   string `json:"tag"`
}

type Deployment struct {
	Root      layout.Dir                     `layout:"dir" path:"."`
	Env       formats.TOMLFile[map[string]string] `layout:"content" path:"env.toml"`
	Manifests layout.Slot[*ManifestDir]      `layout:"directories" path:"manifests"`
}

type ManifestDir struct {
	Root    layout.Dir                    `layout:"dir" path:"."`
	Current formats.JSONFile[AppManifest] `layout:"content" path:"current.json"`
}
```

This gives you:

- one static root-level environment file
- one dynamic manifest directory per app or region
- typed JSON payloads for generated machine-readable artifacts

## Embedded tooling next to configuration

```go
type Toolchain struct {
	Root     layout.Dir                    `layout:"dir" path:"."`
	Settings formats.YAMLFile[map[string]any] `layout:"content" path:"settings.yaml"`
	Scripts struct {
		Build  layout.Exec `layout:"exec" path:"build"`
		Deploy layout.Exec `layout:"exec" path:"deploy"`
	} `layout:"group" path:"bin"`
}
```

This is useful when the filesystem itself is the contract:

- config lives with the scripts that act on it
- `Layout.Ensure` can create empty script files with executable permissions
- `Exec` lets the layout invoke those files without hard-coding extra paths elsewhere

## Inspect raw file content without loading it all at once

Use `File.Inspect` when the check is content-sensitive but you do not want Conduit to define the parsing rules for you:

```go
notes := layout.NewFile("/srv/workspace/NOTES.txt")

err := notes.Inspect(layout.DefaultContext, func(src io.Reader) error {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fmt.Println("first meaningful line:", line)
		return nil
	}
	return scanner.Err()
})
```

Why this works well:

- the callback gets a normal `io.Reader`, so callers can use `bufio.Scanner`, `bufio.Reader`, `io.ReadAll`, decoders, or custom parsing
- Conduit handles the file open and path-safety behavior
- you can stop as soon as your own logic has learned enough

## Reuse the same matcher for files, bytes, and strings

Use `MatchLines` when the check is line-oriented and the result should be a boolean:

```go
hasMeaningfulContent := func(line string) (bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return false, nil
	}
	return true, nil
}

matched, _ := layout.MatchLinesString("# note\n\nvalue\n", hasMeaningfulContent)
present, _ := layout.NewFile("/srv/workspace/NOTES.txt").MatchLines(layout.DefaultContext, hasMeaningfulContent)

_ = matched
_ = present
```

Why this works well:

- the same matcher can be reused in tests, helpers, and real file checks
- callers keep control of what counts as meaningful content
- later convenience helpers can build on the same boolean shape

## Scan non-line tokens with a custom split

Use token helpers when the input should be split some other way:

```go
var words []string

_ = layout.InspectTokensString("alpha beta\ngamma", layout.TokenOptions{
	Split: bufio.ScanWords,
}, func(token string) error {
	words = append(words, token)
	return nil
})

_ = words
```

Why this works well:

- the token helpers reuse the same callback shape across strings, bytes, readers, and files
- `TokenOptions` exposes `bufio.Scanner` split and sizing controls without dropping back to low-level reader plumbing
- line helpers stay available as the simplest case

## Flat config bundles with `FileSlot[T]`

```go
type ServiceOverride struct {
	Replicas int      `yaml:"replicas"`
	Flags    []string `yaml:"flags"`
}

type Environment struct {
	Root      layout.Dir                                           `layout:"dir" path:"."`
	Overrides layout.FileSlot[*formats.YAMLFile[ServiceOverride]]   `layout:"documents" path:"overrides"`
}
```

On disk, each key maps directly to one file instead of a child directory:

```text
environment/
  overrides/
    api.yaml
    worker.yaml
```

Why this works well:

- `FileSlot[T]` keeps the dynamic key space explicit without inventing an extra directory layer per item
- each file still participates in the normal typed-file lifecycle (`Load`, `Discover`, `Sync`, `Scan`)
- it matches cases where the filename itself is the contract, such as `*.yaml` overrides or per-tenant manifests

## Shared assets through links

```go
type Service struct {
	Root      layout.Dir      `layout:"dir" path:"."`
	Config    formats.YAMLFile[ServiceConfig] `layout:"content" path:"config.yaml"`
	SharedEnv layout.FileLink `layout:"content" path:".env"`
	Cache     layout.DirLink  `layout:"content" path:"cache"`
}
```

This works well when the layout owns the link entry but not the target payload:

- `SharedEnv` can point at a checked-in `.env` template or a generated secret file
- `Cache` can point several services at one shared dependency cache directory
- `Layout.Sync` manages only the symlink nodes; the linked file or directory stays under separate ownership

## Tenant directories

```go
type Tenant struct {
	Profile formats.YAMLFile[map[string]string] `layout:"content" path:"profile.yaml"`
}

type Tenants struct {
	Root  layout.Dir             `layout:"dir" path:"."`
	Items layout.Slot[*Tenant]   `layout:"directories" path:"tenants"`
}
```

This pattern is useful when keys come from user input or disk discovery.
For a composed `Layout[Tenants]`:

- `managed.Tree().Items.ComposeMember("acme")` composes and caches a tenant without touching disk
- `managed.Discover(ctx)` discovers tenants present on disk without loading their typed files
- `managed.Load(ctx)` discovers tenants present on disk and loads their typed files
- `managed.Tree().Items.Keys()` gives you a stable list of cached tenant names

`managed.Ensure(ctx)` prepares the whole tree, including roots for cached tenant members. `ComposeMember` itself touches memory only.
