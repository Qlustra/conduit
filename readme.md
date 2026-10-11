Conduit
=======

Conduit is a contract-based content manager for Go. Declare a filesystem tree as Go types, compose its carriers, then move state explicitly between disk and memory.

The module provides these public packages:

- `github.com/qlustra/conduit` for `Structure[T]`, `Layout[T]`, operations, policies, validation, and reporting
- `github.com/qlustra/conduit/layout` for carriers such as `Dir`, `File`, `Exec`, `Link`, `Slot[T]`, `FileSlot[T]`, and `LinkSlot[T]`
- `github.com/qlustra/conduit/formats` for codec-backed typed files such as `JSONFile[T]`, `YAMLFile[T]`, `TOMLFile[T]`, and `EnvFile`
- `github.com/qlustra/conduit/legend` for the Sail layout declaration vocabulary

Conduit uses [Sail](https://github.com/qlustra/sail) for declaration recognition, binding, and live traversal. Conduit owns filesystem paths, carrier behavior, collection caches, and operation policies.

## Install

```bash
go get github.com/qlustra/conduit
```

## Quick start

```go
package main

import (
	"log"

	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

type AppConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type App struct {
	Root   layout.Dir                 `layout:"dir" path:"."`
	Config formats.YAMLFile[AppConfig] `layout:"content" path:"config.yaml"`
}

func main() {
	structure, err := conduit.Compile[App]()
	if err != nil {
		log.Fatal(err)
	}
	managed, err := structure.Compose("/workspace/app")
	if err != nil {
		log.Fatal(err)
	}

	if err := managed.Tree().Config.LoadOrInit(AppConfig{
		Name: "billing",
		Port: 8080,
	}); err != nil {
		log.Fatal(err)
	}
	if _, err := managed.Sync(conduit.DefaultContext); err != nil {
		log.Fatal(err)
	}
}
```

`Compile` recognizes the declaration. `Structure.Compose` constructs a fresh tree, allocates declared pointer paths, binds paths, and configures its slots without touching the filesystem. `Layout.Tree()` exposes that live tree for content and cache updates.

`layout` tags select participation roles; `path` tags supply local paths. Untagged fields and embeddings do not participate. Nested structure uses `layout:"group"`, and collection carriers use `directories`, `documents`, or `links`.

## Whole-tree operations

| Operation | Purpose |
| --- | --- |
| `Ensure(ctx)` | Prepare declared structure and cached member roots. |
| `Load(ctx)` | Discover physical collection members and load content. |
| `Discover(ctx)` | Discover physical members and observe presence while preserving content. |
| `Scan(ctx)` | Observe presence of already composed and cached nodes. |
| `Sync(ctx)` | Persist eligible cached content according to sync policy. |
| `Default()` | Apply in-memory defaults to composed and cached nodes. |
| `Render()` | Derive text into cached content. |
| `Unload()` | Clear cached content while retaining collection membership. |
| `Validate(opts)` | Check composed and cached nodes without changing their state. |

`Ensure` and `Sync` are separate passes. `Sync` writes content; call `Ensure` when you need scaffolding such as empty directories or executable files. Load and Discover accumulate members without evicting stale cache entries. Other passes visit cached members.

## Dynamic collections

```go
type Workspace struct {
	Apps    layout.Slot[*App]                       `layout:"directories" path:"apps"`
	Configs layout.FileSlot[*formats.YAMLFile[AppConfig]] `layout:"documents" path:"configs"`
	Assets  layout.LinkSlot[*layout.Link]            `layout:"links" path:"assets"`
}

structure, err := conduit.Compile[Workspace]()
if err != nil {
	return err
}
managed, err := structure.Compose("/srv/workspace")
if err != nil {
	return err
}

app, err := managed.Tree().Apps.ComposeMember("billing")
if err != nil {
	return err
}
app.Config.Set(AppConfig{Name: "billing", Port: 8080})
_, err = managed.Sync(conduit.DefaultContext)
```

`ComposeMember` constructs and caches one named member in memory. Slots provide cache APIs, typed member entry, and local filesystem behavior. The owning `Layout` performs recursive operations. Use pointer members when operations should mutate cached stateful instances.

## Managed executables

```go
type Tooling struct {
	Root  layout.Dir  `layout:"dir" path:"."`
	Build layout.Exec `layout:"exec" path:"bin/build"`
}

structure, err := conduit.Compile[Tooling]()
if err != nil {
	return err
}
managed, err := structure.Compose("/workspace")
if err != nil {
	return err
}

out, err := managed.Tree().Build.Output(context.Background(), layout.RunOptions{
	Args: []string{"--check"},
	Dir:  managed.Tree().Root.Path(),
})
```

This fragment also imports Go's `context` package. `Exec` retains the raw `File` APIs and adds explicit process execution.

## Documentation

- [Docs index](docs/readme.md)
- [Layout usage](docs/usage/layout.md)
- [Operations usage](docs/usage/operations.md)
- [Formats usage](docs/usage/formats.md)
- [States usage](docs/usage/states.md)
- [API index](docs/api/readme.md)
- [Layout examples](docs/examples/layout.md)
- [Operations examples](docs/examples/operations.md)
- [Formats examples](docs/examples/formats.md)

## License

Apache 2.0

Conduit is maintained within the Qlustra Engineering Tooling Lab.
https://github.com/Qlustra
