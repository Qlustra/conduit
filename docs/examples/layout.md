# Layout examples

These examples focus on how to model real directory structures with Conduit.

## Multi-service workspace

This is a common pattern for local orchestration, test fixtures, or generated environments.

```go
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type Service struct {
	Root    layout.Dir                      `layout:"."`
	Config  formats.YAMLFile[ServiceConfig] `layout:"config.yaml"`
	Logs    layout.Dir                      `layout:"logs"`
	Runner  layout.Exec                     `layout:"bin/run"`
}

type Workspace struct {
	Root     layout.Dir            `layout:"."`
	Services layout.Slot[*Service] `layout:"services"`
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
	Root      layout.Dir                     `layout:"."`
	Env       formats.TOMLFile[map[string]string] `layout:"env.toml"`
	Manifests layout.Slot[*ManifestDir]      `layout:"manifests"`
}

type ManifestDir struct {
	Root    layout.Dir                    `layout:"."`
	Current formats.JSONFile[AppManifest] `layout:"current.json"`
}
```

This gives you:

- one static root-level environment file
- one dynamic manifest directory per app or region
- typed JSON payloads for generated machine-readable artifacts

## Embedded tooling next to configuration

```go
type Toolchain struct {
	Root     layout.Dir                    `layout:"."`
	Settings formats.YAMLFile[map[string]any] `layout:"settings.yaml"`
	Scripts struct {
		Build  layout.Exec `layout:"build"`
		Deploy layout.Exec `layout:"deploy"`
	} `layout:"bin"`
}
```

This is useful when the filesystem itself is the contract:

- config lives with the scripts that act on it
- `EnsureDeep` can create empty script files with executable permissions
- `Exec` lets the layout invoke those files without hard-coding extra paths elsewhere

## Tenant directories

```go
type Tenant struct {
	Profile formats.YAMLFile[map[string]string] `layout:"profile.yaml"`
}

type Tenants struct {
	Root  layout.Dir             `layout:"."`
	Items layout.Slot[*Tenant]   `layout:"tenants"`
}
```

This pattern is useful when keys come from user input or disk discovery:

- `Items.Add("acme", ctx)` creates a new tenant
- `DiscoverDeep(&tenants, ctx)` discovers tenants already present on disk without loading their typed files
- `LoadDeep(&tenants, ctx)` discovers tenants already present on disk
- `Items.Keys()` gives you a stable list of cached tenant names
