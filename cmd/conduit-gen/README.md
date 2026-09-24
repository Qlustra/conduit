# conduit-gen

`conduit-gen` generates a named, typed control surface over existing Conduit
layouts. The generated methods delegate to `mgmt`; generation does not copy
operation recipes or inspect the managed filesystem.

Run inside the package containing the topology declaration:

```sh
go run github.com/qlustra/conduit/cmd/conduit-gen -type NotesManagement -output notes_gen.go
```

The command uses the current Go build configuration and compiler type information.
Dependencies must build. Unresolved references to the file being generated are
allowed, so first generation does not require placeholder runtime wrappers. The
output is deterministic. An existing handwritten output file is never replaced.

```go
type NotesManagement struct {
    Space     spec.Space[NotesLayout]
    Notebooks spec.Collection[NotebookManagement] `scope:"from=Space;bind=Notebooks" create:"NotebookCreation"`
    Logs      spec.Scope[LogsLayout]              `scope:"from=Space;bind=Logs" ops:"inspect,scaffold"`
    All       spec.Scope[NotesLayout]             `scope:"from=Space;bind=." ops:"inspect,validate"`
}

type NotebookManagement struct {
    Scope    spec.Scope[NotebookLayout]
    Manifest spec.Document[Manifest] `scope:"from=Scope;bind=Manifest"`
    View     spec.Document[Manifest] `scope:"from=Scope;bind=Manifest" ops:"read"`
}

var NotebookCreation = mgmt.Creation[NotebookLayout, NewNotebook]{
    Initialize: func(ctx context.Context, tree *NotebookLayout, input NewNotebook) error {
        tree.Manifest.Set(Manifest{Title: input.Title})
        return nil
    },
}
```

Generated names remove a `Management` suffix, or append `Controller` when there
is no suffix. `NotesManagement` produces `Notes`, `NewNotes`, and `BindNotes`.
Named handles such as `notes.Notebooks` are real generated fields. The root token
is private implementation state; root operations appear directly on `notes`.

```go
notes, err := NewNotes(root, mgmt.SpaceOptions{}) // Compose once; no disk writes.
created, err := notes.Notebooks.Create(ctx, "conduit", input, mgmt.CreatePolicy{
    Parent: mgmt.CreateMissing,
})
manifest, err := created.Value.Manifest.Read(ctx, mgmt.ReadPolicy{})
child, err := notes.Notebooks.At("conduit", mgmt.BindPolicy{})
// child and created.Value reference the same cached child layout.
```

`BindNotes(root layout.Dir, tree *NotesLayout, options mgmt.SpaceOptions)` wraps an
already composed layout while preserving its content. `NewNotes` resolves an
absolute root before composition. Root `spec.Project` uses `mgmt.ProjectOptions`.
Root `spec.Scope` has only a `Bind...` constructor accepting an existing
`mgmt.Bound` owner and `mgmt.ScopeOptions`.

## Declaration support

- One root `spec.Project[L]`, `spec.Space[L]`, or `spec.Scope[L]` per topology.
- Exported named fields, `scope:"from=Handle;bind=Field.Path"`, and `bind=.`.
  `from` defaults to the root. Field order does not control dependency order.
- Directory `layout.Slot[*L]` collections whose token argument names a local
  Scope-root topology. `At` returns the generated child wrapper. A `create`
  binding names a local `mgmt.Creation[L,I]` variable and supplies typed `Create`.
  Its result retains both the generated child and the runtime outcome, including
  partial effects on error.
- Typed documents satisfying the runtime content contract, including the built-in
  JSON/YAML/TOML formats. Different views may bind the same document.
- Nested local topology structs with Scope or Space roots. Child Spaces use
  `mgmt.BindSubspace` and require an exported `layout.Dir` tagged `layout:"."`.
- A `bootstrap` binding names a local `mgmt.Initialization[L,I]` variable. Without
  one, Bootstrap uses the runtime's no-input conventional initialization.
- `ops:"..."` narrows generated methods; an empty `ops` tag exposes no operations.
  Normal root/scope operations are inspect, validate, scaffold, and bootstrap;
  load/save require matching content contracts, refresh requires declared derived
  renderable content. Projects add detect. Documents expose read, load, save,
  update, validate. Collections expose at/list, plus create when configured.
  Invocation policies remain ordinary typed `mgmt` policy values.

Storage roles and recognition signals stay on layouts: `manage:"detect"`,
`manage:"required"`, `manage:"role=seeded"`, and `manage:"role=derived"`.
The generator supplies named topology; `mgmt` remains responsible for enforcing
node semantics and operation applicability at runtime.

## Initial boundaries

The initial generator does not support imported/generic topology declarations,
embedded topology fields, file-slot collections, nested Projects, external-root
bindings, raw Blob tokens, or arbitrary recipe expression strings. Project
recognition uses layout annotations or runtime `ProjectOptions`; a `project` tag
is rejected. Custom refresh bindings and scope-default declaration syntax are
also deferred. Unsupported explicit operations and recognized unsupported tags
produce errors instead of silently disappearing from the generated surface.

Child bindings currently use zero-valued options under the parent's runtime
context. Use manual `mgmt` bindings or a handwritten wrapper when a child requires
custom document validation callbacks or custom scope options. Explorer is used
through the runtime API; it needs no static topology generation.

Generated bindings preserve state, and operations inherit the runtime's ordinary
synchronous behavior and partial-effect reporting. Generated wrappers do not add
transactions or concurrency guarantees.
