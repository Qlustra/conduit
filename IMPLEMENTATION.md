# Implementation Handoff

This file tracks active hardening work and deferred implementation plans. Settled
API details live in the docs:

- `docs/usage/layout.md`
- `docs/usage/pipeline.md`
- `docs/api/layout.md`
- `docs/api/pipeline.md`
- `docs/examples/pipeline.md`

Validation baseline:

```sh
go test ./...
```

## Active Pipeline Hardening

The next pass should make the new `pipeline` APIs feel stable before adding more
features.

1. Add package-level Go documentation examples for `pipeline`.
2. Review `TaskResult.Handover` after real examples and tests. Keep one general
   field unless separate `Bridge`, `Compile`, and `Expand` result fields prove
   clearer.
3. Add atomic write support after the thread-safety contract. Start with
   per-file atomic replacement semantics rather than promising a multi-file
   transaction.

## Handover Follow-Ups

Typed handover tasks are first-class tasks:

- `Bridge(...).Populate(...)` for same-cardinality handover: `O -> T`.
- `Compile(...).Build(...)` for many-to-one handover: `[]O -> T`.
- `Expand(...).Extract(...)` for one-to-many handover: `O -> []T`.

The required handover verbs intentionally remain in the fluent chain so task
definitions read in runtime order. A handover task without `Populate`, `Build`,
or `Extract` is incomplete and fails at `Run` with a configuration error.

Runtime order to preserve:

- `Bridge`: snapshot origins, `Filter`, `Sort`, `Rekey`, duplicate-key policy,
  compose targets with `At`, `Populate`, update target cache with `Put`, deep
  operations.
- `Compile`: snapshot origins, `Filter`, `Sort`, compose target with `At`,
  `Build`, update target cache with `Put`, deep operations.
- `Expand`: snapshot origins, `Filter`, `Sort`, `Extract`, duplicate-key policy
  over emitted keys, compose targets with `At`, emitted populate callbacks,
  update target cache with `Put`, deep operations.

## Deferred Pipeline Work

These are useful but should not block the current hardening pass:

- Change detection, fingerprinting, and checksums.
- Watch mode.
- Richer custom source extensibility.
- Branch/sub-task execution.
- Streaming APIs for large files.
- Temp-file-backed transforms.
- Better multi-file sink commit behavior.

## Deferred Design Notes

### Inputs Builder

The current direction is package-level task providers, not a standalone input
builder. If ergonomic incremental input construction becomes useful later, add a
separate builder while keeping provider APIs primary:

```go
inputs := pipeline.Inputs().
	File(a).
	File(b).
	Source(custom)

pipeline.TaskFromInputs("format", inputs).
	Transform(layout.DefaultContext, fn).
	WriteBack(layout.DefaultContext)
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
