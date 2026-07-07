# Pipeline API

The `pipeline` package provides a concrete runner for heterogeneous buffered
tasks over byte subjects and typed layout subjects.

## Runner

```go
func New(tasks ...Runnable) *Pipeline
```

`Pipeline` methods:

- `Add(tasks ...Runnable)`: appends tasks in run order.
- `Run(ctx context.Context, opts RunOptions) (Result, error)`: runs tasks and stops on the first error.

```go
type Runnable interface {
	Name() string
	Run(ctx context.Context, opts RunOptions) (TaskResult, error)
}
```

## Thread Safety

`Pipeline` and built-in task values are safe to configure and run from multiple
goroutines under this contract:

- `Pipeline.Add` may run concurrently with `Pipeline.Run`.
- `Pipeline.Run` snapshots the task list and built-in task definitions before executing tasks.
- concurrent `Pipeline.Run` calls on the same pipeline are serialized.
- built-in task `Run` calls are serialized per task value.
- fluent task mutations are locked and do not affect an already-snapshotted run.
- separate pipelines that share the same layout/workspace are not globally
  synchronized by `pipeline`.
- custom `Runnable` implementations are responsible for their own internal
  synchronization.
- user callbacks are responsible for synchronizing any external shared state or
  goroutines they create.

The contract covers `pipeline`'s own task configuration and execution state. It
does not make mutable typed layout values from `layout` safe for concurrent
mutation across independent pipelines.

## Context

```go
type RunOptions struct {
	Context Context
}

type Context struct {
	Layout           layout.Context
	DuplicateOutputs DuplicateOutputPolicy
}
```

`Context` is required. `DefaultContext` embeds `layout.DefaultContext` and uses
`DuplicateOutputFail`.

Byte sinks and typed `SyncDeep` use their operation-level `layout.Context` for
filesystem write behavior. Set `layout.Context.WritePolicy` to
`layout.WriteAtomicReplace` to request per-file atomic replacement.

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

	Handover HandoverResult
}

type DeepOperationResult struct {
	Items []DeepItemResult
}

type DeepItemResult struct {
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

type HandoverResult struct {
	Kind  HandoverKind
	Items []HandoverItemResult
}

type HandoverItemResult struct {
	OriginKey  string
	OriginName string
	OriginPath string
	OriginFile string
	OriginDir  string
	OriginKeys []string
	TargetKey  string
	TargetName string
	TargetPath string
	TargetFile string
	TargetDir  string
	Err        error
}
```

Statuses:

- `StatusRan`
- `StatusFailed`

Byte sink fields are populated for the terminal sink used by the task. Typed
deep-operation fields are populated for each requested deep operation. Deep
layout entries are captured for `EnsureDeep`, `SyncDeep`, and `ValidateDeep`.
Handover task results are recorded in `Handover`.

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
is the output path candidate. `File` and `Dir` are populated when the subject is
file- or directory-backed. `Data` is used by byte tasks.

## Byte Tasks

```go
type Blob struct {
	Key  string
	Name string
	Path string
	Data []byte
}
```

Providers:

- `TaskFromFile(name string, file layout.File) *ByteSingleTask`
- `TaskFromFiles(name string, files ...layout.File) *ByteMultiTask`
- `TaskFromBlob(name string, blob Blob) *ByteSingleTask`
- `TaskFromBlobs(name string, blobs ...Blob) *ByteMultiTask`

Byte callbacks:

```go
type TransformFunc = layout.TransformFunc
type SplitFunc func(ctx context.Context, lctx layout.Context, split Split, item Item[Blob]) error
type FilterFunc func(ctx context.Context, lctx layout.Context, filter Filter, item Item[Blob]) (bool, error)
type SortFunc func(a Item[Blob], b Item[Blob]) bool
type FileMapper func(ctx context.Context, lctx layout.Context, item Item[Blob]) (layout.File, error)
```

`ByteSingleTask` methods:

- `Transform(lctx layout.Context, fn TransformFunc) *ByteSingleTask`
- `Split(lctx layout.Context, fn SplitFunc) *ByteMultiTask`
- `WriteBack(lctx layout.Context) *ByteSingleTask`
- `To(lctx layout.Context, dest layout.File) *ByteSingleTask`

`ByteMultiTask` methods:

- `Transform(lctx layout.Context, fn TransformFunc) *ByteMultiTask`
- `Filter(lctx layout.Context, fn FilterFunc) *ByteMultiTask`
- `Sort(fn SortFunc) *ByteMultiTask`
- `Concat(lctx layout.Context, opts layout.ConcatOptions) *ByteSingleTask`
- `WriteBack(lctx layout.Context) *ByteMultiTask`
- `ToDir(lctx layout.Context, dest layout.Dir, opt DestinationOption) *ByteMultiTask`
- `ToFiles(lctx layout.Context, mapper FileMapper) *ByteMultiTask`

`Split` helpers:

- `Read() ([]byte, error)`
- `Emit(item Item[Blob])`
- `EmitBytes(name string, data []byte)`
- `EmitString(name string, data string)`
- `EmitFile(file layout.File)`
- `EmitBlob(blob Blob)`

## Typed Tasks

Providers:

- `TaskFromSlot[T](name string, slot *layout.Slot[T]) *TypedSingleTask[*layout.Slot[T]]`
- `TaskFromSlots[T](name string, slots ...*layout.Slot[T]) *TypedMultiTask[*layout.Slot[T]]`
- `TaskFromSlotEntries[T](name string, slots ...*layout.Slot[T]) *TypedMultiTask[T]`
- `TaskFromFileSlot[T](name string, slot *layout.FileSlot[T]) *TypedSingleTask[*layout.FileSlot[T]]`
- `TaskFromFileSlots[T](name string, slots ...*layout.FileSlot[T]) *TypedMultiTask[*layout.FileSlot[T]]`
- `TaskFromFileSlotEntries[T](name string, slots ...*layout.FileSlot[T]) *TypedMultiTask[T]`

Typed callbacks:

```go
type ProcessFunc[T any] func(ctx context.Context, lctx layout.Context, item Item[T]) (T, error)
type TypedFilterFunc[T any] func(ctx context.Context, lctx layout.Context, item Item[T]) (bool, error)
type TypedSortFunc[T any] func(a Item[T], b Item[T]) bool
type TypedSplitFunc[T any] func(ctx context.Context, lctx layout.Context, split TypedSplit[T], item Item[T]) error
type TypedConcatFunc[T any] func(ctx context.Context, lctx layout.Context, items []Item[T]) (T, error)
```

`TypedSingleTask[T]` methods:

- `Process(lctx layout.Context, fn ProcessFunc[T]) *TypedSingleTask[T]`
- `Split(lctx layout.Context, fn TypedSplitFunc[T]) *TypedMultiTask[T]`
- `EnsureDeep(lctx layout.Context) *TypedSingleTask[T]`
- `DefaultDeep() *TypedSingleTask[T]`
- `RenderDeep() *TypedSingleTask[T]`
- `SyncDeep(lctx layout.Context) *TypedSingleTask[T]`
- `ValidateDeep(opts layout.ValidateOptions) *TypedSingleTask[T]`

`TypedMultiTask[T]` methods:

- `Process(lctx layout.Context, fn ProcessFunc[T]) *TypedMultiTask[T]`
- `Filter(lctx layout.Context, fn TypedFilterFunc[T]) *TypedMultiTask[T]`
- `Sort(fn TypedSortFunc[T]) *TypedMultiTask[T]`
- `Split(lctx layout.Context, fn TypedSplitFunc[T]) *TypedMultiTask[T]`
- `Concat(lctx layout.Context, fn TypedConcatFunc[T]) *TypedSingleTask[T]`
- `EnsureDeep(lctx layout.Context) *TypedMultiTask[T]`
- `DefaultDeep() *TypedMultiTask[T]`
- `RenderDeep() *TypedMultiTask[T]`
- `SyncDeep(lctx layout.Context) *TypedMultiTask[T]`
- `ValidateDeep(opts layout.ValidateOptions) *TypedMultiTask[T]`

`TypedSplit[T]` helpers:

- `Emit(item Item[T])`
- `EmitValue(key string, value T)`

## Typed Handover Tasks

Entry descriptors:

- `SlotEntries[T](slot *layout.Slot[T]) Entries[T]`
- `FileSlotEntries[T](slot *layout.FileSlot[T]) Entries[T]`
- `SlotEntry[T](slot *layout.Slot[T], name string) Entry[T]`
- `FileSlotEntry[T](slot *layout.FileSlot[T], name string) Entry[T]`

Callbacks:

```go
type HandoverKeyFunc[O any] func(ctx context.Context, lctx layout.Context, origin Item[O]) (string, error)
type BridgeFunc[O, T any] func(ctx context.Context, lctx layout.Context, origin Item[O], target *Item[T]) error
type ExtractFunc[O, T any] func(ctx context.Context, lctx layout.Context, origin Item[O], emit EntryEmitter[T]) error
type BuildFunc[O, T any] func(ctx context.Context, lctx layout.Context, origins []Item[O], target *Item[T]) error
```

Constructors:

- `Bridge[O, T](name string, origin Entries[O], target Entries[T]) *BridgeTask[O, T]`
- `Expand[O, T](name string, origin Entries[O], target Entries[T]) *ExpandTask[O, T]`
- `Compile[O, T](name string, origin Entries[O], target Entry[T]) *CompileTask[O, T]`

Task verbs:

- `Bridge`: `Filter`, `Sort`, `Rekey`, required `Populate`, and typed deep operations.
- `Expand`: `Filter`, `Sort`, required `Extract`, and typed deep operations.
- `Compile`: `Filter`, `Sort`, required `Build`, and typed deep operations.

Handover tasks read origin slot entries at execution time, compose targets with
`At`, update target caches with `Put`, and only persist when explicit deep
operations are attached. `Bridge` defaults target keys to `origin.Key`; `Rekey`
overrides that. Duplicate target keys use `Context.DuplicateOutputs`.

The required handover verb stays in the fluent chain so task definitions read in
runtime order. Running a handover task without `Populate`, `Extract`, or `Build`
fails with a configuration error. Runtime order is:

- `Bridge`: snapshot origins, `Filter`, `Sort`, `Rekey`, duplicate-key policy, compose targets, `Populate`, update target cache, deep operations.
- `Expand`: snapshot origins, `Filter`, `Sort`, `Extract`, duplicate-key policy, compose targets, emitted populate callbacks, update target cache, deep operations.
- `Compile`: snapshot origins, `Filter`, `Sort`, compose target, `Build`, update target cache, deep operations.

## Output Shaping

Byte `ToDir` uses explicit destination options:

- `Flatten() DestinationOption`
- `PreserveStructure() DestinationOption`

Destination modes:

- `DestinationFlatten`
- `DestinationPreserveStructure`

Each task accepts one byte sink. Registering a second byte sink is a task
configuration error surfaced by `Run`. Typed tasks can chain different deep
operations in order, but registering the same deep operation more than once is a
configuration error.
