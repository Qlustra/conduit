# Management runtime

`mgmt` is a control surface over bound Conduit layouts. It owns the recurring
sequences for initialization, creation, loading, validation, persistence, and
regeneration. Applications supply ordinary layouts, typed inputs, and domain
rules. Manual bindings and generated wrappers use the same runtime.

This is the first implementation. The executable [manual examples](example_test.go)
and [Explorer example](explorer_example_test.go) show the supported APIs. Run them
with `go test ./mgmt -run Example`.

## Controllers and handles

- **Project** is a Space with recognition signals. Binding creates a candidate;
  `Detect` checks identity, `OpenProject` requires a match, and `LocateProject`
  searches ancestors. Location stops at a broken candidate boundary rather than
  silently selecting a different project above it.
- **Space** establishes a chosen management root and exposes its root `Scope`.
  It does not require a marker. Notes, logs, staging, and output directories fit.
- **Scope** operates over a supplied layout or participant within an existing root.
  `BindScope` creates another view over the same content state.
- **Document[T]** provides typed reads, validated loads, updates, saves, and checks.
- **Collection[L]** lists physical members and binds/creates typed child scopes.
  The initial typed collection adapter accepts `layout.Slot[*L]`.
- **Explorer** observes an arbitrary directory tree without a layout or project
  identity. It supplies metadata scanning and bounded content gathering.

Bindings retain the supplied values. They do not compose again, load content,
create files, or manufacture named fields. `space.Layout()` exposes the original
storage declaration. A field such as `notes.Notebooks` on the control surface comes
from a handwritten wrapper or [conduit-gen](../cmd/conduit-gen/README.md).

## Start with ordinary layouts

```go
type Manifest struct {
    Title string `json:"title"`
}

type Notebook struct {
    Root     layout.Dir                 `layout:"."`
    Manifest formats.JSONFile[Manifest]  `layout:"notebook.json" manage:"required"`
}

type Notes struct {
    Root      layout.Dir                `layout:"."`
    Marker    layout.Dir                `layout:".notes" manage:"detect,required"`
    Notebooks layout.Slot[*Notebook]     `layout:"notebooks"`
}
```

Compose once at an absolute root, then bind the existing layout:

```go
root, err := filepath.Abs("./research")
if err != nil { return err }
var tree Notes
if err := conduit.Compose(root, &tree); err != nil { return err }

project, err := mgmt.BindProject(tree.Root, &tree, mgmt.ProjectOptions{})
if err != nil { return err }

_, err = mgmt.Bootstrap(ctx, project.Scope(), struct{}{},
    mgmt.Initialization[Notes, struct{}]{}, mgmt.BootstrapPolicy{})
if err != nil { return err }

notebooks, err := mgmt.BindCollection(project, &tree.Notebooks,
    mgmt.CollectionOptions{})
if err != nil { return err }
```

`BindSpace` takes `SpaceOptions` and works the same way without recognition.
`BindSubspace(parent, childRoot, childLayout, options)` establishes a narrower root
context over an existing subtree. A zero-valued child filesystem context inherits
the parent's context; an explicit child context replaces it. A child Space neither
removes that subtree from parent operations nor causes automatic controller dispatch.

## Supply values; let the runtime create the member

```go
creation := mgmt.Creation[Notebook, string]{
    Initialize: func(ctx context.Context, n *Notebook, title string) error {
        n.Manifest.Set(Manifest{Title: title})
        return nil
    },
    Validate: func(ctx context.Context, n *Notebook) error {
        value, _ := n.Manifest.Get()
        if value.Title == "" { return fmt.Errorf("title is required") }
        return nil
    },
}

created, err := mgmt.Create(ctx, notebooks, "conduit", "Conduit research",
    creation, mgmt.CreatePolicy{})
if err != nil {
    // created.Value and created.Outcome can describe partially applied work.
    return err
}
```

`Bootstrap` and `Create` share a preparation engine:

1. Discover relevant members and observe existing disk content.
2. Apply node defaults and the typed `Initialize` callback.
3. Restore authoritative existing content, then run `Context` to supply render data.
4. Render missing content, run validation, and encode every eligible write.
5. Materialize structure and write the prepared content.

`Create` also reserves its child destination after preparation. Existing content is
preserved during Bootstrap, including seeded text. A dirty cache for an existing
file is rejected by default; `ReplaceDirty` explicitly accepts disk authority.
Prepared content for a missing file is legitimate initialization input.

The optional `Context` callback receives the effective values after existing disk
authority has been restored. `AfterApply` runs only after successful application;
its errors have type `*mgmt.PostApplyError` and retain the applied outcome.
Preparation callbacks must not write files or rebind participant paths. They may
leave newly prepared values in memory when preparation fails.

Defaults honor existing `DefaultDeep`/`Default` traversal boundaries. Validation
first admits required content and filesystem structure, then honors custom
`ValidateDeep`/`Validate` domain boundaries. A custom deep validator owns its
subtree's domain checks and may return its own aggregate error; management cannot
invent independent diagnostics inside that opaque contract.

## Typed document operations

Pointers to the built-in JSON, YAML, TOML, Env, and other Format wrappers satisfy
`DocumentNode[T]`; ordinary formats need no custom adapter:

```go
manifest, err := mgmt.BindDocument(created.Value,
    &created.Value.Layout().Manifest, mgmt.DocumentOptions[Manifest]{})
if err != nil { return err }

outcome, err := manifest.Update(ctx, func(value *Manifest) error {
    value.Title = "Control surface research"
    return nil
}, mgmt.UpdatePolicy{CheckConflict: true})
```

Updates transform an isolated codec clone, validate and encode it, then change the
bound cache and persist it. `Effects: mgmt.MemoryOnly` leaves the result dirty for
a later Save. Failed transformation or validation preserves the original cache.
After a persistence attempt fails, the prepared candidate remains available for
inspection or retry. Persisted Update requires an existing file.

`DocumentOptions[T].Validate` supplies a typed domain rule used by Load, Save,
Update, and Validate. Read obtains a value without domain validation, allowing
inspection and repair. Memory reads and editable candidates use codec cloning;
only fields represented by the codec participate in these values.

`CheckConflict` compares observed disk bytes before transformation and immediately
before writing. A change returns `ErrConflict`. It is a best-effort check, not an
atomic compare-and-swap.

## Policy defaults

Policies are per invocation. Their zero values have these meanings:

- **Inspect:** declared nodes and cached members; collect metadata without loading
  content. `Traversal: Discovered` also binds physical members into the cache.
- **Load:** discover physical members, then load content; reject dirty caches before
  replacing content. `Traversal: Cached` limits the operation to existing bindings.
- **Validate:** use memory and cached members. `Source: Disk` defaults to discovery,
  temporarily loads content for rules, and restores existing content state.
- **Scaffold:** create directories and raw files explicitly marked `empty`.
  `Files: true` also materializes other raw File/Exec declarations; typed content
  is not created as empty placeholders.
- **Save:** validate memory and write dirty cached content only. Missing targets
  fail; `Missing: CreateMissing` permits exclusive creation. `Eligibility: Rewrite`
  also writes loaded or already-synced content.
- **Bootstrap:** complete missing content, preserve existing bytes, reject dirty
  existing caches, and fail when a missing participant has no initialization
  content. `Unsupported: SkipUnsupported` records and skips such participants.
- **Create:** require an absent destination and create its parent when needed.
  `Destination: AbsentOrEmpty` permits an empty existing child directory;
  `Parent: FailIfMissing` requires the collection directory to exist.
- **Refresh:** render declared derived content from current memory and cached
  contexts, then persist it; missing outputs may be created. `Source: Disk` loads
  layout content first. Applications still supply the render contexts their nodes need.
- **Document Read/Update:** source values from disk. `Source: Memory` selects the
  current cached value. Update persists by default and rejects replacing dirty
  memory when sourcing from disk.
- **Collection At/List:** At binds without requiring presence. List observes
  physical entries; `Validate: true` additionally validates child content from disk.
- **Diagnostics:** management operations stop on the first failure unless
  `Diagnostics: CollectAll` is selected where supported. Collected diagnostics
  still produce a non-nil error.

`ScopeOptions.Ops` narrows exposed runtime operations. Nil uses the applicable set;
a non-nil empty slice exposes none. Internal recipe phases do not require callers
to separately expose those phases as operations. `Capabilities()` describes the
bound surface; it is not a prediction that the next filesystem operation succeeds.

## Lasting node semantics

Management reads the `manage` tag separately from Conduit's `layout` tag. Multiple
annotations are separated with commas or semicolons:

```go
Marker   layout.Dir         `layout:".notes" manage:"detect,required"`
Ready    layout.File        `layout:"ready"  manage:"empty"`
Readme   ReadmeTemplate      `layout:"README.md" manage:"role=seeded"`
Index    IndexTemplate       `layout:"INDEX.md"  manage:"role=derived"`
```

- `detect` identifies a physical project recognition signal. ProjectOptions can
  instead supply explicit root-relative signal paths. Detection defaults to any
  valid signal; `DetectionPolicy{All: true}` requires all signals.
- `required` requires content/presence during validation. Initialization validates
  declared structural nodes against its planned scaffold before writing.
- `empty` explicitly permits a raw file to be created without authored content.
- `role=seeded` identifies content initially supplied for later user maintenance.
- `role=derived` selects content for Refresh. The participant must supply a
  `layout.Renderable` or `layout.Templatable` implementation.

Roles apply to individual file participants, not collections or whole groups, and
are not inherited by descendants. A role does not prevent an explicit Document
Update or ordinary Save. Unknown management annotations are rejected. Existing raw
Conduit deep operations keep their existing behavior and do not interpret these tags.

## Exploration without a layout

```go
explorer, err := mgmt.NewExplorer(root, mgmt.ExplorerOptions{
    Selection: mgmt.ExploreSelection{
        Select: func(entry mgmt.ExploreEntry) bool {
            return entry.IsRegular() && strings.HasSuffix(entry.RelativePath, ".md")
        },
        Prune: func(entry mgmt.ExploreEntry) bool {
            return filepath.Base(entry.Path) == ".git"
        },
    },
})
if err != nil { return err }
result, err := explorer.Gather(ctx, mgmt.GatherPolicy{
    MaxFileBytes: 1 << 20, MaxTotalBytes: 8 << 20,
})
```

Scan returns metadata; Gather additionally returns selected regular-file bytes.
Selection and directory pruning are independent. Results have deterministic
lexical traversal order and retain partial entries and diagnostics on failure.
Explorer defaults to collecting diagnostics and never follows symlinks. Gather
limits default to unlimited when zero; an exceeded limit diagnoses and omits the
file instead of truncating it. See the [executable example](explorer_example_test.go).

## Current boundaries

The runtime supports ordinary layout structs, raw Dir/File/Exec nodes, built-in
Format-backed content, pointer-backed directory/file slot traversal, and typed
directory collections. Stateful value-backed slot members do not provide stable
mutable references and are rejected for operations that need them. Link mutation,
file-slot collection adapters, manifest-member collections, removal/reset recipes,
and cross-project registration orchestration are outside this first surface.

Management rejects paths outside the chosen root and observed symlink paths.
These checks do not pin filesystem descriptors or protect against hostile path
replacement races. Explorer similarly observes a changing tree rather than a
snapshot. Operations are synchronous; callers must serialize mutations that share
layout state.

Preparation precedes persistence, but application is not a multi-file transaction.
`Outcome.Effects`, `Outcome.Applied()`, and diagnostics report confirmed work.
`Outcome.MayHaveApplied()` also includes uncertain effects after failed mutation
attempts; use it when deciding whether the filesystem needs reinspection. An
unsuccessful direct write can truncate or partially change a file even when
`Applied()` is false. These flags describe disk effects, not in-memory edits. Existing
file writes honor the configured Conduit write policy; exclusive missing-file
creation currently requires `WriteDirect`. Atomic exclusive publication is not
implemented. Existing raw deep APIs remain available for workflows outside these
management recipes.
