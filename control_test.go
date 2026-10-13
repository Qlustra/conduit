package conduit_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/qlustra/conduit"
	"github.com/qlustra/conduit/control"
	"github.com/qlustra/conduit/layout"
)

type controlFile struct{ layout.TextTemplate[string] }
type controlMain struct{ layout.TextTemplate[string] }
type controlMember struct {
	Root     layout.Dir                  `layout:"dir" path:"."`
	File     controlFile                 `layout:"content" path:"command.go"`
	Children layout.Slot[*controlMember] `layout:"directories" path:"commands"`
}
type controlProgram struct {
	Root     layout.Dir                  `layout:"dir" path:"."`
	File     controlFile                 `layout:"content" path:"program.go"`
	Main     controlMain                 `layout:"content" path:"main.go"`
	Commands layout.Slot[*controlMember] `layout:"directories" path:"commands"`
}
type memberControl struct {
	control.ScopeCtrl[controlMember]
	Source   control.DocumentCtrl[controlFile]                                  `control:"document"`
	Commands control.CollectionCtrl[memberControl, layout.Slot[*controlMember]] `control:"collection"`
}
type programControl struct {
	control.ProjectCtrl[controlProgram]
	Source   control.DocumentCtrl[controlFile]                                  `control:"document"`
	Entry    control.DocumentCtrl[controlMain]                                  `control:"document"`
	Commands control.CollectionCtrl[memberControl, layout.Slot[*controlMember]] `control:"collection"`
}
type controls struct {
	Program programControl `control:"project"`
}

func TestControlLayoutRetainsStateAndBuildsRecursiveMembers(t *testing.T) {
	structure, err := conduit.Compile[controlProgram]()
	if err != nil {
		t.Fatal(err)
	}
	live, err := structure.Compose(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	live.Tree().File.Set("existing state")
	preexisting, err := live.Tree().Commands.ComposeMember("existing")
	if err != nil {
		t.Fatal(err)
	}
	preexisting.File.Set("existing member")
	surface, err := conduit.ControlLayout[controls](structure, live)
	if err != nil {
		t.Fatal(err)
	}
	p := &surface.Program
	if p.Layout() != live || p.Source.Layout() != &live.Tree().File || p.Source.Layout().MustGet() != "existing state" {
		t.Fatal("binding copied or recomposed existing state")
	}
	existing, ok := p.Commands.Get("existing")
	if !ok || existing.Layout().Tree() != preexisting || existing.Source.Layout() != &preexisting.File {
		t.Fatal("existing cached member did not receive its exact control")
	}
	member, err := p.Commands.ComposeMember("new")
	if err != nil {
		t.Fatal(err)
	}
	if again, err := p.Commands.ComposeMember("new"); err != nil || again != member {
		t.Fatal("repeated composition did not return the registered control", err)
	}
	raw, ok := p.Commands.Source().Get("new")
	if !ok || member.Layout().Tree() != raw {
		t.Fatal("member control lost source identity")
	}
	if _, err := os.Stat(member.Source.Layout().Path()); !os.IsNotExist(err) {
		t.Fatal("composition touched disk", err)
	}
	child, err := member.Commands.ComposeMember("nested")
	if err != nil {
		t.Fatal(err)
	}
	child.Source.Layout().Set("nested content")
	if _, err := p.Ensure(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Sync(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(child.Source.Layout().Path())
	if err != nil || string(data) != "nested content" {
		t.Fatalf("nested write = %q, %v", data, err)
	}
	if required, err := member.Commands.Require("nested"); err != nil || required != child {
		t.Fatal("Require lost controller identity", err)
	}
	if err := member.Commands.Delete("nested", layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	if _, ok := member.Commands.Get("nested"); ok {
		t.Fatal("deleted control retained in registry")
	}
	if _, ok := raw.Children.Get("nested"); ok {
		t.Fatal("deleted member retained in source")
	}
	if _, err := os.Stat(child.Source.Layout().Path()); !os.IsNotExist(err) {
		t.Fatal("Delete did not remove disk member", err)
	}
}

func TestControlPathIsIndependentAndScopeDiscoveryRegistersMembers(t *testing.T) {
	structure, err := conduit.Compile[controlProgram]()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	first, err := conduit.ControlPath[programControl](structure, root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := conduit.ControlPath[programControl](structure, root)
	if err != nil {
		t.Fatal(err)
	}
	first.Source.Layout().Set("only first")
	if first.Layout().Tree() == second.Layout().Tree() || second.Source.Layout().HasContent() {
		t.Fatal("independent control construction shared state")
	}
	path := filepath.Join(root, "commands", "found", "commands", "nested")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "command.go"), []byte("disk state"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Discover(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	found, ok := first.Commands.Get("found")
	if !ok {
		t.Fatal("scope discovery omitted member control")
	}
	nested, ok := found.Commands.Get("nested")
	if !ok || nested.Source.Layout().HasContent() {
		t.Fatal("Discover should register without loading")
	}
	if _, err := first.Load(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	if nested.Source.Layout().MustGet() != "disk state" {
		t.Fatal("scope Load lost shared member state")
	}
	if first.Commands.Len() != 1 || second.Commands.Len() != 0 {
		t.Fatal("independent registries shared membership")
	}
}

type selectedProgram struct {
	Primary controlProgram `layout:"group" path:"primary"`
	Backup  controlProgram `layout:"group" path:"backup"`
}
type backupControl struct {
	control.SelectedScopeCtrl[selectedProgram, controlProgram]
	Source control.DocumentCtrl[controlFile] `control:"document"`
}

func (*backupControl) Control(tree *selectedProgram) *controlProgram { return &tree.Backup }

type selectedControls struct {
	Backup backupControl `control:"scope"`
}

func TestSelectedScopeControlsSharedSubtreeWithFocusedOperations(t *testing.T) {
	structure, err := conduit.Compile[selectedProgram]()
	if err != nil {
		t.Fatal(err)
	}
	live, err := structure.Compose(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	live.Tree().Backup.File.Set("retained")
	surface, err := conduit.ControlLayout[selectedControls](structure, live)
	if err != nil {
		t.Fatal(err)
	}
	if surface.Backup.Layout().Tree() != &live.Tree().Backup || surface.Backup.Source.Layout() != &live.Tree().Backup.File {
		t.Fatal("selected scope did not retain shared subtree")
	}
	// Main is not exposed as a document control, but remains in the chosen
	// Layout's deep operation extent.
	live.Tree().Backup.Main.Set("unexposed content")
	if _, err := surface.Backup.Ensure(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	if _, err := surface.Backup.Sync(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(live.Tree().Backup.Main.Path())
	if err != nil || string(data) != "unexposed content" {
		t.Fatal("scope operations omitted unexposed layout member", err)
	}
	if _, err := os.Stat(live.Tree().Primary.Root.Path()); !os.IsNotExist(err) {
		t.Fatal("focused operations reached sibling", err)
	}
}

type repeatedFiles struct {
	First  controlFile `layout:"content" path:"first.go"`
	Second controlFile `layout:"content" path:"second.go"`
	Main   controlMain `layout:"content" path:"main.go"`
}
type chosenFile struct {
	control.SelectedDocumentCtrl[repeatedFiles, controlFile]
}

func (*chosenFile) Control(tree *repeatedFiles) *controlFile { return &tree.Second }

type chosenMain struct {
	control.SelectedDocumentCtrl[repeatedFiles, controlMain]
}

func (*chosenMain) Control(tree *repeatedFiles) *controlMain { return &tree.Main }

type chosenDocuments struct {
	control.ScopeCtrl[repeatedFiles]
	Source chosenFile `control:"document"`
	Entry  chosenMain `control:"document"`
}

func TestSharedLegendBindsHeterogeneousSelectors(t *testing.T) {
	structure, err := conduit.Compile[repeatedFiles]()
	if err != nil {
		t.Fatal(err)
	}
	surface, err := conduit.ControlPath[chosenDocuments](structure, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if surface.Source.Layout() != &surface.Layout().Tree().Second || surface.Entry.Layout() != &surface.Layout().Tree().Main {
		t.Fatal("heterogeneous selector contract selected wrong fields")
	}
}

type ambiguousDocuments struct {
	control.ScopeCtrl[repeatedFiles]
	Second control.DocumentCtrl[controlFile] `control:"document"`
}
type incorrectParent struct {
	control.SelectedDocumentCtrl[controlProgram, controlFile]
}

func (*incorrectParent) Control(tree *controlProgram) *controlFile { return &tree.File }

type wrongContext struct {
	control.ScopeCtrl[repeatedFiles]
	Source incorrectParent `control:"document"`
}
type nilDocument struct {
	control.SelectedDocumentCtrl[repeatedFiles, controlFile]
}

func (*nilDocument) Control(*repeatedFiles) *controlFile { return nil }

type nilSurface struct {
	control.ScopeCtrl[repeatedFiles]
	Source nilDocument `control:"document"`
}

func TestControlSelectionRejectsAmbiguityWrongParentAndNil(t *testing.T) {
	structure, err := conduit.Compile[repeatedFiles]()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conduit.ControlPath[ambiguousDocuments](structure, t.TempDir()); err == nil || !strings.Contains(err.Error(), "2 declared carriers") {
		t.Fatal("field-name convention must not resolve ambiguity", err)
	}
	if _, err := conduit.ControlPath[wrongContext](structure, t.TempDir()); err == nil || !strings.Contains(err.Error(), "enclosing scope") {
		t.Fatal("selector parent must match actual scope", err)
	}
	if _, err := conduit.ControlPath[nilSurface](structure, t.TempDir()); err == nil || !strings.Contains(err.Error(), "returned nil") {
		t.Fatal("nil selection should fail live binding", err)
	}
}

func TestCollectionControlOperationsAndExternalMutationBoundary(t *testing.T) {
	structure, err := conduit.Compile[controlProgram]()
	if err != nil {
		t.Fatal(err)
	}
	surface, err := conduit.ControlPath[programControl](structure, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	collection := &surface.Commands
	member, err := collection.ComposeMember("through-control")
	if err != nil {
		t.Fatal(err)
	}
	member.Source.Layout().Set("from controller")
	if _, err := collection.Ensure(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	if _, err := collection.Sync(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	collection.Clear()
	if collection.Len() != 0 || collection.Source().Len() != 0 {
		t.Fatal("Clear did not clear both registries")
	}
	if _, err := collection.Load(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	loaded, ok := collection.Get("through-control")
	if !ok || loaded.Source.Layout().MustGet() != "from controller" {
		t.Fatal("collection Load did not compose registered controls")
	}
	external, err := collection.Source().ComposeMember("external")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := collection.Get("external"); ok {
		t.Fatal("registry lookup silently synchronized external changes")
	}
	if collection.Len() != 1 {
		t.Fatal("registry length follows raw source rather than registered controls")
	}
	// An explicit controller request can adopt an existing source member.
	adopted, err := collection.ComposeMember("external")
	if err != nil || adopted.Layout().Tree() != external {
		t.Fatal("explicit adoption lost live member", err)
	}
	collection.Source().Remove("external")
	if kept, ok := collection.Get("external"); !ok || kept != adopted {
		t.Fatal("raw eviction silently reconciled controller")
	}
	collection.Remove("external")
	if _, ok := collection.Get("external"); ok {
		t.Fatal("controller eviction retained member")
	}
}

type repeatedCollections struct {
	First  layout.Slot[*controlMember] `layout:"directories" path:"first"`
	Second layout.Slot[*controlMember] `layout:"directories" path:"second"`
}
type chosenCollection struct {
	control.SelectedCollectionCtrl[repeatedCollections, memberControl, layout.Slot[*controlMember]]
}

func (*chosenCollection) Control(tree *repeatedCollections) *layout.Slot[*controlMember] {
	return &tree.Second
}

type collectionSelection struct {
	control.ScopeCtrl[repeatedCollections]
	Commands chosenCollection `control:"collection"`
}

func TestSelectedCollectionConstructsControlsFromChosenSource(t *testing.T) {
	structure, err := conduit.Compile[repeatedCollections]()
	if err != nil {
		t.Fatal(err)
	}
	surface, err := conduit.ControlPath[collectionSelection](structure, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	member, err := surface.Commands.ComposeMember("chosen")
	if err != nil {
		t.Fatal(err)
	}
	if surface.Commands.Source() != &surface.Layout().Tree().Second || surface.Layout().Tree().First.Len() != 0 {
		t.Fatal("selected collection used wrong source")
	}
	raw, ok := surface.Layout().Tree().Second.Get("chosen")
	if !ok || member.Layout().Tree() != raw {
		t.Fatal("selected member lost source identity")
	}
}

type ambiguousMember struct {
	Root   layout.Dir  `layout:"dir" path:"."`
	First  controlFile `layout:"content" path:"first.go"`
	Second controlFile `layout:"content" path:"second.go"`
}
type invalidMemberControl struct {
	control.ScopeCtrl[ambiguousMember]
	File control.DocumentCtrl[controlFile] `control:"document"`
}
type invalidMemberLayout struct {
	Members layout.Slot[*ambiguousMember] `layout:"directories" path:"members"`
}
type invalidMemberSurface struct {
	control.ScopeCtrl[invalidMemberLayout]
	Members control.CollectionCtrl[invalidMemberControl, layout.Slot[*ambiguousMember]] `control:"collection"`
}

func TestCollectionValidatesMemberDeclarationBeforeComposition(t *testing.T) {
	structure, err := conduit.Compile[invalidMemberLayout]()
	if err != nil {
		t.Fatal(err)
	}
	live, err := structure.Compose(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conduit.ControlLayout[invalidMemberSurface](structure, live); err == nil || !strings.Contains(err.Error(), "2 declared carriers") {
		t.Fatal("bad nested declaration accepted", err)
	}
	if live.Tree().Members.Len() != 0 {
		t.Fatal("validation composed source members")
	}
}

type failedFile struct{ layout.TextTemplate[string] }

func (*failedFile) Load() (bool, error) { return false, os.ErrPermission }

type failingMember struct {
	Root layout.Dir `layout:"dir" path:"."`
	File failedFile `layout:"content" path:"file"`
}
type failingMemberControl struct {
	control.ScopeCtrl[failingMember]
}
type failingProgram struct {
	Members layout.Slot[*failingMember] `layout:"directories" path:"members"`
}
type failingControl struct {
	control.ScopeCtrl[failingProgram]
	Members control.CollectionCtrl[failingMemberControl, layout.Slot[*failingMember]] `control:"collection"`
}

func TestFailedScopeLoadRegistersMembersItComposed(t *testing.T) {
	structure, err := conduit.Compile[failingProgram]()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "members", "partial"), 0o755); err != nil {
		t.Fatal(err)
	}
	surface, err := conduit.ControlPath[failingControl](structure, root)
	if err != nil {
		t.Fatal(err)
	}
	if code, err := surface.Load(layout.DefaultContext); err == nil || code != layout.LoadFailed {
		t.Fatal("original load failure disappeared", code, err)
	}
	member, ok := surface.Members.Get("partial")
	if !ok {
		t.Fatal("failed controller operation composed an unregistered member")
	}
	raw, ok := surface.Members.Source().Get("partial")
	if !ok || member.Layout().Tree() != raw {
		t.Fatal("partial registration lost live member")
	}
}

func TestConcurrentCollectionCompositionReturnsOneController(t *testing.T) {
	structure, err := conduit.Compile[controlProgram]()
	if err != nil {
		t.Fatal(err)
	}
	surface, err := conduit.ControlPath[programControl](structure, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const callers = 12
	results := make([]*memberControl, callers)
	errors := make([]error, callers)
	var workers sync.WaitGroup
	for i := range callers {
		workers.Go(func() { results[i], errors[i] = surface.Commands.ComposeMember("same") })
	}
	workers.Wait()
	for i := range callers {
		if errors[i] != nil || results[i] == nil || results[i] != results[0] {
			t.Fatal("concurrent callers did not converge on one controller", errors[i])
		}
	}
	if surface.Commands.Len() != 1 || surface.Commands.Source().Len() != 1 {
		t.Fatal("concurrent composition duplicated membership")
	}
}
