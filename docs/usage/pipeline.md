# Pipeline usage

The `pipeline` package runs buffered byte and typed-layout tasks in order.

## Mental model

```go
p := pipeline.New(
	pipeline.TaskFromFiles("format", files...),
	pipeline.TaskFromSlotEntries("services", &ws.Services),
)

_, err := p.Run(ctx)
```

Execution is buffered:

1. Snapshot task inputs.
2. Execute task steps in memory.
3. Plan output destinations.
4. Apply duplicate-output policy.
5. Materialize bytes when needed.
6. Run sinks.

If a step, mapping function, duplicate check, or byte materialization fails, the
task performs no sink writes.

`pipeline.DefaultContext` embeds `layout.DefaultContext` and uses
`DuplicateOutputFail`. Copy it to override policy:

```go
pctx := pipeline.DefaultContext
pctx.DuplicateOutputs = pipeline.DuplicateOutputLastWins
```

Operation-level layout contexts merge with `pctx.Layout`. Zero-value fields
inherit from the run context, so this is enough to request atomic writes for one
sink:

```go
task.WriteToFileWith(layout.Context{WritePolicy: layout.WriteAtomicReplace}, out)
```

## Byte tasks

Use file tasks when inputs are `layout.File` handles:

```go
p := pipeline.New(
	pipeline.TaskFromFiles("format", files...).
		Transform(gofmtTransform).
		WriteToSources(),
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
	}).WriteToFile(out),
)
```

Byte task operations:

- `Transform`: rewrite one or many byte items.
- `Split`: expand one byte item into many.
- `Filter`: keep only matching items.
- `Sort`: reorder byte items.
- `Pick`: keep the first item matching a predicate.
- `Select`: choose one item from the full set.
- `Concat`: reduce many items into one.

Byte task sinks:

- `WriteToSource` / `WriteToSources`
- `WriteToFile`
- `WriteToDir`
- `WriteToDirPreserve`
- `WriteToFiles`

`WriteToSource` and `WriteToSources` require file-backed source items.
Each byte task accepts exactly one sink.

## Splitting bytes

```go
func splitLines(ctx context.Context, lctx layout.Context, split pipeline.ByteSplitter, item pipeline.Item[pipeline.Blob]) error {
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
		Split(splitLines).
		WriteToDir(outputDir),
)
```

`Split.Read()` caches file-backed bytes for repeated reads during the callback.

## Selecting one byte item

`Pick` keeps the first matching item:

```go
task := pipeline.TaskFromBlobs("pick", blobs...).
	Pick(func(item pipeline.Item[pipeline.Blob]) bool {
		return item.Name == "README.md"
	}).
	WriteToFile(out)
```

`Select` chooses from the whole item set:

```go
task := pipeline.TaskFromBlobs("select", blobs...).
	Select(func(items []pipeline.Item[pipeline.Blob]) pipeline.Item[pipeline.Blob] {
		return items[len(items)-1]
	}).
	WriteToFile(out)
```

`Pick` returns an error when no items match.

## Mapping byte outputs

`WriteToFiles` maps destinations before bytes are staged:

```go
p := pipeline.New(
	pipeline.TaskFromFiles("compile", inputs...).
		Transform(compile).
		WriteToFiles("compiled", func(ctx context.Context, lctx layout.Context, item pipeline.Item[pipeline.Blob]) (layout.File, error) {
			return out.File(item.Name + ".out"), nil
		}),
)
```

`WriteToDir(out)` writes by basename. `WriteToDirPreserve(out)` preserves each
item's relative `Path`.

## Typed tasks

Typed tasks operate on cached slot entries from `layout.Slot[T]` and
`layout.FileSlot[T]`.

```go
p := pipeline.New(
	pipeline.TaskFromSlotEntries("services", &ws.Services).
		Filter(func(ctx context.Context, lctx layout.Context, item pipeline.Item[*Service]) (bool, error) {
			cfg := item.Value.Config.MustGet()
			return cfg.Enabled, nil
		}).
		Process(func(ctx context.Context, lctx layout.Context, item pipeline.Item[*Service]) (*Service, error) {
			cfg := item.Value.Config.MustGet()
			cfg.Port = cfg.Port + 1000
			item.Value.Config.Set(cfg)
			return item.Value, nil
		}).
		SyncDeep(),
)
```

Typed task operations:

- `Process`: `T -> T`
- `Filter`
- `Sort`
- `Split`: `T -> []T`

Typed task sinks:

- `EnsureDeep`
- `DefaultDeep`
- `RenderDeep`
- `SyncDeep`
- `ValidateDeep`

Typed entry sources snapshot cached entries when the task runs. Call
`DiscoverDeep` or `LoadDeep` before running when entries should be loaded from
disk first.

## Deferred features

The current package intentionally does not yet include:

- generic layout traversal
- glob-first source discovery
- change detection or task caching
- watch mode
- streaming or temp-file-backed execution
- branch/sub-task execution
