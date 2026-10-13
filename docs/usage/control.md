# Control surfaces

Control surfaces assemble typed access and operations over a composed layout.
`ScopeCtrl[L]` retains a focused `Layout[L]`; `ProjectCtrl[L]` embeds that scope
and gives it project meaning. Project detection is still developing.
Scope operations cover the entire selected layout, including participants that
have no exposed control field.

The following example combines unique-type matching, an explicit typed selector,
and recursive member controls:

```go
package example

import (
	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/control"
	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

type Config struct {
	Name string `yaml:"name"`
}
type ConfigFile struct{ formats.YAMLFile[Config] }

type ProgramLayout struct {
	Root     layout.Dir                  `layout:"dir" path:"."`
	Primary  ConfigFile                  `layout:"content" path:"primary.yaml"`
	Backup   ConfigFile                  `layout:"content" path:"backup.yaml"`
	Commands layout.Slot[*CommandLayout] `layout:"directories" path:"commands"`
}

type CommandLayout struct {
	Root     layout.Dir                  `layout:"dir" path:"."`
	Config   ConfigFile                  `layout:"content" path:"config.yaml"`
	Commands layout.Slot[*CommandLayout] `layout:"directories" path:"commands"`
}

type BackupControl struct {
	control.SelectedDocumentCtrl[ProgramLayout, ConfigFile]
}

func (*BackupControl) Control(tree *ProgramLayout) *ConfigFile {
	return &tree.Backup
}

type ProgramControl struct {
	control.ProjectCtrl[ProgramLayout]
	Config   BackupControl                                                       `control:"document"`
	Commands control.CollectionCtrl[CommandControl, layout.Slot[*CommandLayout]] `control:"collection"`
}

type CommandControl struct {
	control.ScopeCtrl[CommandLayout]
	Config   control.DocumentCtrl[ConfigFile]                                    `control:"document"`
	Commands control.CollectionCtrl[CommandControl, layout.Slot[*CommandLayout]] `control:"collection"`
}

func Prepare(root string) (*ProgramControl, error) {
	structure, err := conduit.Compile[ProgramLayout]()
	if err != nil {
		return nil, err
	}
	program, err := conduit.ControlPath[ProgramControl](structure, root)
	if err != nil {
		return nil, err
	}
	program.Config.Layout().Set(Config{Name: "backup"})
	command, err := program.Commands.ComposeMember("build")
	if err != nil {
		return nil, err
	}
	command.Config.Layout().Set(Config{Name: "build"})
	if _, err := program.Ensure(layout.DefaultContext); err != nil {
		return nil, err
	}
	if _, err := program.Sync(layout.DefaultContext); err != nil {
		return nil, err
	}
	return program, nil
}
```

Ordinary control carriers match a unique declared carrier type within their
scope, including static layout groups. Field names are convention. Opaque
collection members are addressed through their source's keys. A scope controlling
the enclosing layout's own type binds its root.

`SelectedDocumentCtrl[Parent, Target]` supplies the expected
`Control(*Parent) *Target` contract. Its outer authored struct supplies the hook.
The shared control legend uses Sail's `AcceptContractFamily` to validate each
carrier's instantiated contract without running its selector. Conduit then checks
the declared parent against the enclosing scope and invokes the hook on live
state. Nil results fail binding. Go field references in the hook participate in
renaming and compiler checks.

`SelectedScopeCtrl[Parent, Target]`, `SelectedProjectCtrl[Parent, Target]`, and
`SelectedCollectionCtrl[Parent, MemberControl, Source]` provide the same explicit
selection form for the other roles. A collection selector returns its source;
member keys are still selected through collection operations.

## Independent and shared construction

`ControlPath[C](structure, rootPath)` composes fresh state and constructs a
surface. Separate calls at the same filesystem path have independent layout
state and controller registries. L is inferred from `*Structure[L]`.

`ControlLayout[C](structure, live)` binds existing state. A whole-layout scope
retains the supplied `*Layout[L]` exactly. Focused scopes retain Layout views
over selected existing trees, preserving their paths, content, and member caches.
C can be a scope/project itself, as above, or an envelope of fields marked
`control:"project"` and `control:"scope"`.

For example, a master layout can contain two `ProgramLayout` groups:

```go
type Workspace struct {
	Primary ProgramLayout `layout:"group" path:"primary"`
	Archive ProgramLayout `layout:"group" path:"archive"`
}

type ArchiveControl struct {
	control.SelectedScopeCtrl[Workspace, ProgramLayout]
	Config BackupControl `control:"document"`
}

func (*ArchiveControl) Control(tree *Workspace) *ProgramLayout {
	return &tree.Archive
}

// With structure, err := conduit.Compile[Workspace]() and a composed master:
// archive, err := conduit.ControlLayout[ArchiveControl](structure, master)
// archive.Layout().Tree() == &master.Tree().Archive
// archive.Ensure(ctx) covers Archive, including its unexposed Commands source.
```

## Collections and operation boundaries

`CollectionCtrl[MemberControl, Source]` retains a source and a keyed registry of
member controllers. Each member controller retains its own focused Layout. This
initial implementation uses pointer members and scope-shaped member controls.
It prepares recursive member declarations before composing live members.

- `ComposeMember(name)` constructs and registers a member control in memory.
  Repeated calls return the registered controller; composition does not touch disk.
- `Require(name)` checks the member's physical existence and returns its control.
- `Get`, `Keys`, `Len`, and `All` inspect registered controls without discovery.
- `Remove` and `Clear` evict both source members and their controls in memory.
  `Delete(name, ctx)` also removes the physical member.
- `Load` and `Discover` compose physical members and register controls, recursively.
  Scope-level Load and Discover also register the members produced by their Layout
  operation, including those composed before a later traversal failure.
- `Ensure`, `Scan`, `Sync`, `Validate`, `Default`, `Render`, and `Unload` operate
  on registered member controls. Load and Discover accumulate members; they do
  not evict stale entries.

Document `Layout()` exposes the exact native content carrier, preserving its
codec or template methods. Scope `Layout()` exposes its focused live handler;
collection `Source()` exposes its live source through the controller.

Controller operations coordinate their own composition and registration. Direct
independent changes to layouts or sources carry no automatic synchronization
guarantee. Callers own reconciliation; ordinary access does not detect drift or
refresh controller identities. Code generation is outside this implementation.
