# Implementation Handoff

This file tracks deferred implementation plans. Settled API details live in the
docs:

- `docs/usage/layout.md`
- `docs/usage/pipeline.md`
- `docs/api/layout.md`
- `docs/api/pipeline.md`
- `docs/examples/pipeline.md`

Validation baseline:

```sh
go test ./...
```

## Current State

The pipeline hardening pass and source/target API cleanup are complete. Pipeline
tasks now configure sources and targets through fluent methods, public task names
use source/target terminology, and task sources snapshot at runtime.

The public shape is:

- `Byte("name")` / `Bytes("name")` for byte single/multi tasks.
- `Slot[T]("name")` / `Slots[T]("name")` for typed single/multi tasks.
- `Bridge[O, T]("name")`, `Compile[O, T]("name")`, and `Expand[O, T]("name")`
  with explicit type arguments and fluent `Take*` / `Target*` methods.
- Byte sink methods named `WriteTo*` so source/target configuration is distinct
  from persistence actions.

All task sources snapshot when the task runs, not when the task is constructed.
That lets earlier pipeline tasks populate slot entries or byte targets that later
tasks consume predictably.

## Handover Design Notes

Typed handover tasks are first-class tasks:

- `Bridge[O, T]("name").TakeSlotEntries(...).TargetSlotEntries(...).Populate(...)`
  for same-cardinality handover: `O -> T`.
- `Compile[O, T]("name").TakeSlotEntries(...).TargetSlotEntry(...).Build(...)`
  for many-to-one handover: `[]O -> T`.
- `Expand[O, T]("name").TakeSlotEntries(...).TargetSlotEntries(...).Extract(...)`
  for one-to-many handover: `O -> []T`.

The required handover verbs intentionally remain in the fluent chain so task
definitions read in runtime order. A handover task without `Populate`, `Build`,
or `Extract` is incomplete and fails at `Run` with a configuration error.

Runtime order to preserve:

- `Bridge`: snapshot sources, `Filter`, `Sort`, `Rekey`, duplicate-key policy,
  compose targets with `At`, `Populate`, update target cache with `Put`, deep
  operations.
- `Compile`: snapshot sources, `Filter`, `Sort`, compose target with `At`,
  `Build`, update target cache with `Put`, deep operations.
- `Expand`: snapshot sources, `Filter`, `Sort`, `Extract`, duplicate-key policy
  over emitted keys, compose targets with `At`, emitted populate callbacks,
  update target cache with `Put`, deep operations.

## Source/Target Follow-Up Cleanup

Postponed follow-up items:

- Rename public result fields that still say origin, such as `OriginKey`, to
  source terminology.
- Internally rename remaining `origin` fields/functions to `source` where it does
  not obscure handover logic.
- Keep `subject` as an internal term only for backing entities when useful.

## Deferred Pipeline Work

These are useful but are not part of the current pipeline surface:

- Change detection, fingerprinting, and checksums.
- Watch mode.
- Richer custom source extensibility.
- Branch/sub-task execution.
- Streaming APIs for large files.
- Temp-file-backed transforms.
- Better multi-file sink commit behavior.

## Deferred Design Notes

### Inputs Builder

The current direction is fluent source methods on shape-specific tasks, not a
standalone input builder. If ergonomic incremental input construction becomes
useful later, add a separate builder without replacing `Byte`, `Bytes`, `Slot`,
or `Slots` as the primary task entry points:

```go
inputs := pipeline.Inputs().
	File(a).
	File(b).
	Source(custom)

pipeline.Bytes("format").
	TakeInputs(inputs).
	Transform(fn).
	WriteToSources()
```

### Branch

`Branch` remains deferred.

Likely shape:

- `Split` stays a primitive subject-shape operation.
- `Branch` opens sub-task execution.
- `SingleStep.Branch` branches once over the single item.
- `MultiStep.Branch` branches per item.

Avoid matrix semantics unless there is a concrete need.

### Change Detection And Watch Mode

Hash helpers exist in `layout`, but pipeline-level change detection is not
implemented.

Future design should include task fingerprints, source hashes, task version
strings, a cache interface, and likely a JSON cache implementation backed by a
`layout.File`.

Watch mode should wait until source discovery and change detection exist.
