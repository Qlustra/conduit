# Pipeline API

The `pipeline` package provides a concrete buffered runner for byte-oriented and
typed layout tasks.

## Runner

```go
func New(tasks ...Runtime) *Pipeline
```

`Pipeline` methods:

- `Add(tasks ...Runtime)` appends tasks in run order.
- `Run(ctx context.Context, contexts ...Context) (Result, error)` runs tasks and stops on the first error.

```go
type Runtime interface {
	Name() string
	Run(ctx context.Context, contexts ...Context) (TaskResult, error)
}
```

## Context

```go
type Context struct {
	Layout           layout.Context
	DuplicateOutputs DuplicateOutputPolicy
}
```

If `Run` receives no `Context`, it uses `DefaultContext`.

Operation-level contexts are merged with `Context.Layout`:

- zero-value fields inherit from the run context
- non-zero fields override only that operation

This means callers can set only `WritePolicy`, `Reporter`, or another single
field without rebuilding the full `layout.Context`.

Duplicate policies:

- `DuplicateOutputUnset`: invalid for execution.
- `DuplicateOutputFail`: rejects duplicate output paths.
- `DuplicateOutputLastWins`: keeps only the last planned item for each path.

## Results

```go
type Result struct {
	Tasks []TaskResult
}

type TaskResult struct {
	Name   string
	Status Status
	Err    error

	EnsureDeep   DeepOperationResult
	DefaultDeep  DeepOperationResult
	RenderDeep   DeepOperationResult
	SyncDeep     DeepOperationResult
	ValidateDeep DeepOperationResult

	WriteBack ByteWriteResult
	To        ByteWriteResult
	ToDir     ByteWriteResult
	ToFiles   ByteWriteResult
}

type DeepOperationResult struct {
	Items []SlotOperationResult
}

type SlotOperationResult struct {
	Key     string
	Name    string
	Path    string
	File    string
	Dir     string
	Result  layout.ResultCode
	Entries []layout.Entry
	Err     error
}

type ByteWriteResult struct {
	Items []ByteWriteItemResult
}

type ByteWriteItemResult struct {
	Key   string
	Name  string
	Path  string
	File  string
	Bytes int
	Err   error
}
```

Statuses:

- `StatusRan`
- `StatusFailed`

Byte result fields are populated for the task sink that ran. Typed deep results
are populated for each requested deep operation.

## Items

```go
type Item[T any] struct {
	Key   string
	Name  string
	Path  string
	File  layout.File
	Dir   layout.Dir
	Value T
	Data  []byte
}
```

`Key` is stable source identity. `Name` is filename-ish when one exists. `Path`
is the output path candidate. `File` and `Dir` are populated for file-backed and
directory-backed items. `Data` is used by byte tasks.

## Byte Tasks

```go
type Blob struct {
	Key  string
	Name string
	Path string
	Data []byte
}
```

Constructors:

- `TaskFromFile(name string, file layout.File) *SingleContentTask`
- `TaskFromBlob(name string, blob Blob) *SingleContentTask`
- `TaskFromFiles(name string, files ...layout.File) *ContentCollectionTask`
- `TaskFromDir(name string, dir layout.Dir) *ContentCollectionTask`
- `TaskFromBlobs(name string, blobs ...Blob) *ContentCollectionTask`

`TaskFromDir` snapshots the directory's direct regular files when the task runs.
`WriteToSource` and `WriteToSources` require file-backed source items.

Byte callbacks:

```go
type TransformFunc = layout.TransformFunc
type SortContentFunc func(a Item[Blob], b Item[Blob]) bool
type PickContentFunc func(item Item[Blob]) bool
type SelectContentFunc func(items []Item[Blob]) Item[Blob]
type SplitContentFunc func(ctx context.Context, lctx layout.Context, split ByteSplitter, item Item[Blob]) error
type FilterContentFunc func(ctx context.Context, lctx layout.Context, filter ByteFilter, item Item[Blob]) (bool, error)
type MapContentFunc func(ctx context.Context, lctx layout.Context, item Item[Blob]) (layout.File, error)
```

`SingleContentTask` methods:

- `Transform(fn TransformFunc) *SingleContentTask`
- `TransformWith(lctx layout.Context, fn TransformFunc) *SingleContentTask`
- `Split(fn SplitContentFunc) *ContentCollectionTask`
- `SplitWith(lctx layout.Context, fn SplitContentFunc) *ContentCollectionTask`
- `WriteToSource() *SingleContentTask`
- `WriteToSourceWith(lctx layout.Context) *SingleContentTask`
- `WriteToFile(dest layout.File) *SingleContentTask`
- `WriteToFileWith(lctx layout.Context, dest layout.File) *SingleContentTask`

`ContentCollectionTask` methods:

- `Transform(fn TransformFunc) *ContentCollectionTask`
- `TransformWith(lctx layout.Context, fn TransformFunc) *ContentCollectionTask`
- `Filter(fn FilterContentFunc) *ContentCollectionTask`
- `FilterWith(lctx layout.Context, fn FilterContentFunc) *ContentCollectionTask`
- `Sort(fn SortContentFunc) *ContentCollectionTask`
- `Pick(fn PickContentFunc) *SingleContentTask`
- `PickWith(lctx layout.Context, fn PickContentFunc) *SingleContentTask`
- `Select(fn SelectContentFunc) *SingleContentTask`
- `SelectWith(lctx layout.Context, fn SelectContentFunc) *SingleContentTask`
- `Concat(opts layout.ConcatOptions) *SingleContentTask`
- `ConcatWith(lctx layout.Context, opts layout.ConcatOptions) *SingleContentTask`
- `WriteToSources() *ContentCollectionTask`
- `WriteToSourcesWith(lctx layout.Context) *ContentCollectionTask`
- `WriteToDir(dest layout.Dir) *ContentCollectionTask`
- `WriteToDirWith(lctx layout.Context, dest layout.Dir) *ContentCollectionTask`
- `WriteToDirPreserve(dest layout.Dir) *ContentCollectionTask`
- `WriteToDirPreserveWith(lctx layout.Context, dest layout.Dir) *ContentCollectionTask`
- `WriteToFiles(sinkLabel string, mapper MapContentFunc) *ContentCollectionTask`
- `WriteToFilesWith(lctx layout.Context, sinkLabel string, mapper MapContentFunc) *ContentCollectionTask`

Each byte task accepts exactly one sink. Registering a second sink is a task
configuration error.

`Pick` fails when no items match. `Select` uses the item returned by the callback
as-is.

`ByteSplitter` helpers:

- `Read() ([]byte, error)`
- `Emit(item Item[Blob])`
- `EmitBytes(name string, data []byte)`
- `EmitString(name string, data string)`
- `EmitFile(file layout.File)`
- `EmitBlob(blob Blob)`

`ByteFilter` helpers:

- `Read() ([]byte, error)`

## Typed Tasks

Constructors:

- `TaskFromSlotEntries[T any](name string, slot *layout.Slot[T]) *MultiSlotTask[T]`
- `TaskFromFileSlotEntries[T any](name string, slot *layout.FileSlot[T]) *MultiSlotTask[T]`

Typed sources snapshot cached slot entries when the task runs. They do not
discover from disk automatically; call `DiscoverDeep` or `LoadDeep` first when
the slot cache should be populated from the filesystem.

Typed callbacks:

```go
type ProcessTypedFunc[I any] func(ctx context.Context, lctx layout.Context, item Item[I]) (I, error)
type FilterTypedFunc[I any] func(ctx context.Context, lctx layout.Context, item Item[I]) (bool, error)
type SortTypedFunc[I any] func(a Item[I], b Item[I]) bool
type SplitTypedFunc[I any] func(ctx context.Context, lctx layout.Context, splitter TypedSplitter[I], item Item[I]) error
```

`MultiSlotTask[I]` methods:

- `Process(fn ProcessTypedFunc[I]) *MultiSlotTask[I]`
- `ProcessWith(lctx layout.Context, fn ProcessTypedFunc[I]) *MultiSlotTask[I]`
- `Filter(fn FilterTypedFunc[I]) *MultiSlotTask[I]`
- `FilterWith(lctx layout.Context, fn FilterTypedFunc[I]) *MultiSlotTask[I]`
- `Sort(fn SortTypedFunc[I]) *MultiSlotTask[I]`
- `Split(fn SplitTypedFunc[I]) *MultiSlotTask[I]`
- `SplitWith(lctx layout.Context, fn SplitTypedFunc[I]) *MultiSlotTask[I]`
- `EnsureDeep() *MultiSlotTask[I]`
- `EnsureDeepWith(lctx layout.Context) *MultiSlotTask[I]`
- `DefaultDeep() *MultiSlotTask[I]`
- `RenderDeep() *MultiSlotTask[I]`
- `SyncDeep() *MultiSlotTask[I]`
- `SyncDeepWith(lctx layout.Context) *MultiSlotTask[I]`
- `ValidateDeep(opts layout.ValidateOptions) *MultiSlotTask[I]`

Typed tasks can chain multiple different deep operations. Registering the same
deep operation more than once is a configuration error.

`TypedSplitter[I]` helpers:

- `Emit(item Item[I])`
- `EmitValue(key string, value I)`

## Output Shaping

`WriteToDir` flattens item paths by basename. `WriteToDirPreserve` keeps each
item's relative `Path` under the destination root.

Directory sinks reject empty, absolute, `.` / `..`, and escaping relative paths
during planning.
