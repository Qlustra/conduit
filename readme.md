Conduit
=======

A contract-based content manager for Go.

`conduit` lets you describe a filesystem structure as semantic Go types, then interact with it through explicit, directional operations.

It does not try to reconcile state.
It does not apply hidden policy.
It does not guess intent.

You decide which side is authoritative, and when.

---

## Core idea

Define your structure once:

```go
type AppConfig struct {
    Name string `yaml:"name"`
    Port int    `yaml:"port"`
}

type App struct {
    Root   conduit.Dir                 `layout:"."`
    Config conduit.YAMLFile[AppConfig] `layout:"config.yaml"`
}

type Workspace struct {
    Root conduit.Dir        `layout:"."`
    Apps conduit.Slot[*App] `layout:"apps"`
}
```

Compose it:

```go
var ws Workspace
_ = conduit.Compose("/workspace", &ws)
```

And operate on it explicitly:

```go
ctx := conduit.DefaultContext

// prepare structure
_ = conduit.EnsureDeep(&ws, ctx)

// discover and load from disk
_ = conduit.LoadDeep(&ws, ctx)

// mutate in memory
app := ws.Apps.MustAt("billing")
cfg := app.Config.MustGet()
cfg.Port = 9000
app.Config.Set(cfg)

// write back to disk
_ = conduit.SyncDeep(&ws, ctx)
```

---

## Philosophy

`conduit` is built around a simple rule:

> Filesystem and memory are separate sources of truth.
> Operations move data between them.
> Nothing happens implicitly.

There is no reconciliation engine.
There is no automatic merge.
There is no hidden sync loop.

Instead, you get a small set of explicit operations:

* **Ensure** → materialize structure on disk
* **Load** → read from disk into memory
* **Sync** → write memory state to disk
* **Scan** → observe disk state without mutating memory

You choose the direction.
You choose the timing.

---

## Concepts

### Dir and File

Stateless path handles.

```go
type Dir
type File
```

They represent locations, not state.

---

### Format[T, C]

A stateful file with typed content and a codec.

```go
type Format[T any, C Codec[T]] struct { ... }
```

Concrete types are simple aliases:

```go
type YAMLFile[T any] struct {
    conduit.Format[T, YAMLCodec[T]]
}
```

Formats track two independent state axes:

* **DiskState** — what we observed on disk
* **MemoryState** — what we know about in-memory content

---

### Slot[T]

A dynamic container for repeated child structures.

```go
type Slot[T any]
```

Used for layouts like:

```
apps/<app>/config.yaml
```

Access:

```go
app := ws.Apps.MustAt("billing")
```

Create:

```go
app, _ := ws.Apps.Add("billing", ctx)
```

Slots cache created and discovered entries and can load them from disk.

---

## State model

State is explicit and observable.

### Disk state

```go
DiskUnknown
DiskMissing
DiskPresent
```

### Memory state

```go
MemoryUnknown
MemoryLoaded
MemorySynced
MemoryDirty
```

These are independent.

Example:

* file exists but not loaded → `DiskPresent`, `MemoryUnknown`
* file loaded and modified → `DiskPresent`, `MemoryDirty`
* file written by us → `DiskPresent`, `MemorySynced`

---

## Operations

All operations are explicit and directional.

### EnsureDeep

Materializes structure on disk.

```go
conduit.EnsureDeep(&ws, ctx)
```

Creates directories and files declared in the structure.

---

### LoadDeep

Loads content from disk into memory.

```go
conduit.LoadDeep(&ws, ctx)
```

* populates typed files
* scans slots for existing entries
* does not create missing files

---

### SyncDeep

Writes in-memory content to disk.

```go
conduit.SyncDeep(&ws, ctx)
```

* writes only loaded/dirty content
* does not invent missing entries
* does not delete anything

---

### ScanDeep

Observes disk state.

```go
conduit.ScanDeep(&ws, ctx)
```

* updates disk state
* discovers slot entries
* does not load content

---

## Context

Operations accept a context:

```go
type Context struct {
    DirMode  os.FileMode
    FileMode os.FileMode
}
```

Example:

```go
ctx := conduit.Context{
    DirMode:  0o755,
    FileMode: 0o644,
}
```

Context is passed explicitly.
It is not encoded in tags.

---

## What this is not

* Not a config framework
* Not a filesystem abstraction layer
* Not a sync/reconciliation engine
* Not a policy system

If you need implicit behavior, this is not the tool.

---

## Use cases

* Project scaffolding
* Structured configuration management
* Workspace and environment layout handling
* Tooling where filesystem shape matters

---

## Design goals

* Explicit over implicit
* Composition over configuration
* Small primitives
* No hidden state transitions
* Predictable behavior

---

## Status

Early-stage. APIs may evolve.
