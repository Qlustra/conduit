Conduit
=======

Conduit is a contract-based content manager for Go.

The module is split into three public packages:

- `github.com/qlustra/conduit` for operations, `Context`, and sync policy
- `github.com/qlustra/conduit/layout` for structural nodes such as `Dir`, `File`, `Exec`, `Slot[T]`, and `TextTemplate[C]`
- `github.com/qlustra/conduit/formats` for codec-backed typed files such as `JSONFile[T]`, `YAMLFile[T]`, and `TOMLFile[T]`

It lets you describe a filesystem as semantic Go types, then move state explicitly between disk and memory:

- `Compose` binds paths to a layout.
- `EnsureDeep` materializes declared structure.
- `DiscoverDeep` discovers declared structure from disk without loading typed content.
- `LoadDeep` reads disk content into memory.
- `SyncDeep` writes sync-eligible typed memory state back to disk.
- `ScanDeep` observes disk presence for already composed items.

There is no implicit reconciliation loop, merge policy, or background sync. You decide which side is authoritative and when data moves.

## Install

```bash
go get github.com/qlustra/conduit
```

## Quick start

```go
package main

import (
	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

type AppConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type App struct {
	Root   layout.Dir                  `layout:"."`
	Config formats.YAMLFile[AppConfig] `layout:"config.yaml"`
}

func main() {
	var app App

	_ = conduit.Compose("/workspace/app", &app)
	_ = conduit.EnsureDeep(&app, conduit.DefaultContext)

	_ = app.Config.LoadOrInit(AppConfig{
		Name: "billing",
		Port: 8080,
	})

	cfg := app.Config.MustGet()
	cfg.Port = 9000
	app.Config.Set(cfg)

	_ = conduit.SyncDeep(&app, conduit.DefaultContext)
}
```

## Representative examples

Dynamic collections use `Slot[T]` to model repeated children:

```go
type Workspace struct {
	Root layout.Dir         `layout:"."`
	Apps layout.Slot[*App]  `layout:"apps"`
}

var ws Workspace
_ = conduit.Compose("/workspace", &ws)

app, _ := ws.Apps.Add("billing", conduit.DefaultContext)
_ = app.Config.LoadOrInit(AppConfig{Name: "billing", Port: 8080})
_ = conduit.SyncDeep(&ws, conduit.DefaultContext)
```

Managed executables stay part of the layout and can be run through `Exec`:

```go
import (
	"context"

	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/layout"
)

type Tooling struct {
	Root  layout.Dir   `layout:"."`
	Build layout.Exec  `layout:"bin/build"`
}

var tools Tooling
_ = conduit.Compose("/workspace", &tools)

out, _ := tools.Build.Output(context.Background(), layout.RunOptions{
	Args: []string{"--check"},
	Dir:  tools.Root.Path(),
})

_ = out
```

## Documentation

Start with the usage guides:

- [Docs index](docs/readme.md)
- [Layout usage](docs/usage/layout.md)
- [Operations usage](docs/usage/operations.md)
- [Formats usage](docs/usage/formats.md)
- [States usage](docs/usage/states.md)

For raw exported API reference:

- [API index](docs/api/readme.md)
- [Operations API](docs/api/operations.md)
- [Layout API](docs/api/layout.md)
- [Formats API](docs/api/formats.md)

Then browse the longer, real-world examples:

- [Layout examples](docs/examples/layout.md)
- [Operations examples](docs/examples/operations.md)
- [Formats examples](docs/examples/formats.md)

## License

Apache 2.0

Conduit is maintained within the Qlustra Engineering Tooling Lab.
https://github.com/Qlustra
