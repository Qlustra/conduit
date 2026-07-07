# Pipeline usage

The `pipeline` package describes buffered processing tasks over explicit layout
nodes and byte blobs. It sits above `layout`: callers select typed layout nodes
from their own structs, then hand those subjects to pipeline task providers.

## Mental model

`Pipeline` is a concrete runner for heterogeneous tasks:

```go
p := pipeline.New(
	pipeline.TaskFromFiles("format", files...),
	pipeline.TaskFromSlotEntries("services", &ws.Services),
)

_, err := p.Run(ctx, pipeline.RunOptions{Context: pipeline.DefaultContext})
```

Execution is buffered:

1. Execute task steps in memory.
2. Plan destinations.
3. Apply duplicate-output policy.
4. Stage bytes when the task writes byte outputs.
5. Write sinks.

If steps, mapping, duplicate detection, or byte materialization fail, no sink
writes are attempted.

`Pipeline` and built-in tasks snapshot their configuration when a run starts.
Concurrent `Run` calls on the same pipeline or task are serialized, and fluent
mutations made during an in-flight run apply only to later runs. Pipelines do not
globally synchronize separate pipelines that share the same layout/workspace.

`pipeline.DefaultContext` embeds `layout.DefaultContext` and uses
`DuplicateOutputFail`. Copy it to override policy:

```go
pctx := pipeline.DefaultContext
pctx.DuplicateOutputs = pipeline.DuplicateOutputLastWins
```

Pipeline writes inherit `layout.Context` write behavior. To request per-file
atomic replacement for byte sinks or typed `SyncDeep`, set
`pctx.Layout.WritePolicy = layout.WriteAtomicReplace` or pass an operation-level
layout context with that policy.

## Byte tasks

Use file tasks when inputs are `layout.File` handles:

```go
p := pipeline.New(
	pipeline.TaskFromFiles("format", files...).
		Transform(layout.DefaultContext, gofmtTransform).
		WriteBack(layout.DefaultContext),
)
```

Use blob tasks for opaque in-memory bytes:

```go
p := pipeline.New(
	pipeline.TaskFromBlob("generated", pipeline.Blob{
		Key:  "readme",
		Name: "README.md",
		Path: "docs/README.md",
		Data: data,
	}).To(layout.DefaultContext, out),
)
```

Use shared blob subjects when one task should produce bytes for a later task in
the same pipeline run:

```go
bundle := pipeline.BlobSubjectFromBlob(pipeline.Blob{
	Key:  "bundle",
	Name: "bundle.json",
	Path: "bundle.json",
})

p := pipeline.New(
	pipeline.CompileToFile("bundle", pipeline.SlotEntries(&ws.Services), bundle).
		Build(layout.DefaultContext, buildBundleJSON),
	pipeline.TaskFromBlobSubject("gzip-bundle", bundle).
		Transform(layout.DefaultContext, gzipTransform).
		To(layout.DefaultContext, ws.Artifacts.File("bundle.json.gz")),
)
```

Byte tasks support:

- `Transform`: bytes to bytes.
- `Split`: one byte item to many byte items.
- `Filter` and `Sort` on multi-subject byte tasks.
- `Concat`: many byte items to one byte item.
- `WriteBack`, `To`, `ToDir`, and `ToFiles` sinks.

## Splitting bytes

```go
func splitLines(ctx context.Context, lctx layout.Context, split pipeline.Split, item pipeline.Item[pipeline.Blob]) error {
	data, err := split.Read()
	if err != nil {
		return err
	}
	for i, line := range bytes.Split(data, []byte("\n")) {
		split.EmitBytes(fmt.Sprintf("%03d.txt", i), line)
	}
	return nil
}

p := pipeline.New(
	pipeline.TaskFromFile("lines", source).
		Split(layout.DefaultContext, splitLines).
		ToDir(layout.DefaultContext, outputDir, pipeline.Flatten()),
)
```

`Split.Read()` reads the current item and caches bytes for repeated reads.

## Mapping byte outputs

`ToFiles` maps destinations before bytes are staged:

```go
p := pipeline.New(
	pipeline.TaskFromFiles("compile", inputs...).
		Transform(layout.DefaultContext, compile).
		ToFiles(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item pipeline.Item[pipeline.Blob]) (layout.File, error) {
			return out.File(item.Name + ".out"), nil
		}),
)
```

Use `ToDir(layout.DefaultContext, out, pipeline.Flatten())` to write by
basename, or `ToDir(layout.DefaultContext, out, pipeline.PreserveStructure())`
to preserve item paths. `ToDir`
rejects empty, absolute, `.`/`..`, and escaping paths during planning.

## Typed layout tasks

Typed tasks preserve Conduit's typed layout model. They process known shapes,
not arbitrary bytes.

Task providers include:

- `TaskFromSlot` / `TaskFromSlots` for slot objects.
- `TaskFromSlotEntries` for cached slot entries from one or more slots.
- `TaskFromFileSlot` / `TaskFromFileSlots` for file-slot objects.
- `TaskFromFileSlotEntries` for cached file-slot entries from one or more file slots.

Entry providers snapshot `Entries()` at task construction. Call `DiscoverDeep` or
`LoadDeep` first when entries should be discovered from disk.

Typed tasks support:

- `Process`: `T -> T`.
- `Split`: `T -> []T`.
- `Concat`: `[]T -> T`.
- `Filter` and `Sort` on multi-subject typed tasks.
- `EnsureDeep`, `DefaultDeep`, `RenderDeep`, `SyncDeep`, and `ValidateDeep` sinks.

Example:

```go
p := pipeline.New(
	pipeline.TaskFromSlotEntries("services", &ws.Services).
		Filter(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item pipeline.Item[*Service]) (bool, error) {
			return item.Value.Enabled, nil
		}).
		Process(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, item pipeline.Item[*Service]) (*Service, error) {
			cfg := item.Value.Config.MustGet()
			cfg.Generated = true
			item.Value.Config.Set(cfg)
			return item.Value, nil
		}).
		SyncDeep(layout.DefaultContext),
)
```

Operation methods take a `layout.Context`. If it is the zero value, the task
falls back to `RunOptions.Context.Layout`; passing one explicitly makes the
operation's filesystem policy local to that step or sink. `TaskResult` records
byte write outcomes and typed deep-operation outcomes, including reported deep
layout entries where the underlying layout operation supports reports.

## Typed handover tasks

Use handover tasks when a later subject is derived from slot entries in another
layout reference. Handover tasks read origin entries when they run, populate
target slot caches in memory, and persist only through explicit deep operations.

```go
p := pipeline.New(
	pipeline.TaskFromSlotEntries("configs", &app.Configs).
		Process(layout.DefaultContext, processConfig),

	pipeline.Bridge("configure-servers",
		pipeline.SlotEntries(&app.Configs),
		pipeline.SlotEntries(&app.Servers),
	).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, config pipeline.Item[*Config], server *pipeline.Item[*Server]) error {
			server.Value.Config.Set(ServerConfig{Service: config.Key})
			return nil
		}).
		EnsureDeep(layout.DefaultContext).
		SyncDeep(layout.DefaultContext),
)
```

Use `Compile(...).Build(...)` for many origin entries into one target entry, and
`Expand(...).Extract(...)` for one origin entry into many target entries. This
reference-driven model is most useful with pointer slot entries, such as
`layout.Slot[*Service]`, so in-memory changes are visible across ordered tasks.

`Populate`, `Build`, and `Extract` are required handover definition steps. They
remain in the fluent chain so the declaration follows runtime order: origin
snapshot, optional filtering/sorting/rekeying, handover logic, then optional deep
operations. Running a handover task without its required verb fails as a
configuration error.

## Byte-producing handover tasks

Use `CompileToFile`, `BridgeToFiles`, and `ExpandToFiles` when origins are typed
entries but the produced targets are byte artifacts rather than typed slot
entries.

These tasks:

- read typed origin entries at execution time
- populate `BlobSubject` or `BlobSubjectSet` targets in memory
- optionally write those produced bytes through byte-style sinks
- allow later byte tasks to consume the produced subjects even when no sink is attached

Example:

```go
targets := pipeline.BlobSubjects()

p := pipeline.New(
	pipeline.BridgeToFiles("configs", pipeline.SlotEntries(&ws.Services), targets).
		Populate(layout.DefaultContext, func(ctx context.Context, lctx layout.Context, origin pipeline.Item[*Service], target *pipeline.Item[pipeline.Blob]) error {
			target.File = ws.Output.File(origin.Key + ".json")
			target.Path = origin.Key + ".json"
			target.Data = mustJSON(ServiceConfig{Name: origin.Key})
			return nil
		}).
		ToTargets(layout.DefaultContext),
)
```

`ToTarget` and `ToTargets` write to files already attached to the produced blob
subjects. `To`, `ToDir`, and `ToFiles` redirect output exactly like byte-task
sinks. Missing byte sinks are allowed for these tasks so subject sharing stays
useful.

## Deferred features

The current package intentionally does not yet include:

- generic layout traversal
- glob-first source discovery
- change detection or task caching
- watch mode
- streaming or temp-file-backed execution
- branch/sub-task execution
