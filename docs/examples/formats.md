# Formats examples

These examples focus on when each typed file format is a good fit.

## YAML for operator-edited service config

```go
type ServiceConfig struct {
	Name      string   `yaml:"name"`
	Port      int      `yaml:"port"`
	Upstreams []string `yaml:"upstreams"`
}

type Service struct {
	Config formats.YAMLFile[ServiceConfig] `layout:"config.yaml"`
}
```

Typical flow:

```go
svc := ws.Services.MustAt("api")

_ = svc.Config.LoadOrInit(ServiceConfig{
	Name:      "api",
	Port:      8080,
	Upstreams: []string{"worker:7000"},
})

cfg := svc.Config.MustGet()
cfg.Upstreams = append(cfg.Upstreams, "search:7200")
svc.Config.Set(cfg)

_ = svc.Config.Save(conduit.DefaultContext)
```

Why YAML here:

- friendly for hand editing
- good for nested operational config
- common in infrastructure workflows

## JSON for generated manifests

```go
type Manifest struct {
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Labels   map[string]string `json:"labels"`
}

type ArtifactDir struct {
	Manifest formats.JSONFile[Manifest] `layout:"manifest.json"`
}
```

Typical flow:

```go
artifact.Manifest.Set(Manifest{
	Name:  "api",
	Image: "registry.example/api:1.4.2",
	Labels: map[string]string{
		"team": "platform",
	},
})

_ = artifact.Manifest.Save(conduit.DefaultContext)
```

Why JSON here:

- easy for downstream tooling to parse
- deterministic pretty-printed output
- good fit for generated artifacts and lock-style files

## TOML for settings files

```go
type DevSettings struct {
	Profile string `toml:"profile"`
	Debug   bool   `toml:"debug"`
}

type Environment struct {
	Settings formats.TOMLFile[DevSettings] `layout:"dev.toml"`
}
```

Typical flow:

```go
loaded, _ := env.Settings.Load()
if !loaded {
	env.Settings.Set(DevSettings{
		Profile: "local",
		Debug:   true,
	})
}

_ = env.Settings.Sync(conduit.DefaultContext)
```

Why TOML here:

- compact for small settings documents
- readable for humans
- useful when the file is mostly key/value configuration

## Mixing formats in one layout

You can use different file formats in the same tree based on who consumes each file:

```go
type Project struct {
	Runtime  formats.YAMLFile[RuntimeConfig] `layout:"runtime.yaml"`
	Manifest formats.JSONFile[Manifest]      `layout:"manifest.json"`
	Local    formats.TOMLFile[LocalDev]      `layout:"dev.toml"`
}
```

A practical rule:

- YAML for shared runtime config
- JSON for machine-generated output
- TOML for local developer settings
