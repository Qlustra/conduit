package mgmt

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
)

func TestManagementVisitsFirstStructParticipant(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Manifest formats.JSONFile[map[string]string] `layout:"manifest.json" manage:"required"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	diagnostics, err := space.Scope().Validate(context.Background(), ValidationPolicy{})
	if err == nil || !diagnostics.HasErrors() {
		t.Fatal("required first field was not visited")
	}
	tree.Manifest.Set(map[string]string{"name": "first"})
	if _, err := space.Scope().Save(context.Background(), SavePolicy{Missing: CreateMissing}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tree.Manifest.Path()); err != nil {
		t.Fatalf("first participant was not saved: %v", err)
	}
}

func TestManagementScopeKeepsFirstFieldDeclaration(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Manifest formats.JSONFile[map[string]string] `layout:"manifest.json" manage:"required"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := BindScope(space, &tree.Manifest, ScopeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Binding().Path() != tree.Manifest.Path() {
		t.Fatalf("bound scope path = %s, want %s", manifest.Binding().Path(), tree.Manifest.Path())
	}
	if _, err := manifest.Validate(context.Background(), ValidationPolicy{}); err == nil {
		t.Fatal("binding scope lost required declaration")
	}
}

func TestManagementRestrictedSaveRunsItsOwnValidation(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Root     layout.Dir                          `layout:"."`
		Manifest formats.JSONFile[map[string]string] `layout:"manifest.json"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	tree.Manifest.Set(map[string]string{"name": "save"})
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{Scope: ScopeOptions{Ops: []Operation{OpSave}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := space.Scope().Validate(context.Background(), ValidationPolicy{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("public Validate error = %v, want unsupported", err)
	}
	if _, err := space.Scope().Save(context.Background(), SavePolicy{Missing: CreateMissing}); err != nil {
		t.Fatalf("Save's internal validation incorrectly required public Validate capability: %v", err)
	}
	if _, err := os.Stat(tree.Manifest.Path()); err != nil {
		t.Fatal(err)
	}
}

func TestManagementCancelledDiscoveryDoesNotPopulateCache(t *testing.T) {
	type member struct {
		Manifest formats.JSONFile[map[string]string] `layout:"manifest.json"`
	}
	root := t.TempDir()
	var tree struct {
		Root  layout.Dir           `layout:"."`
		Items layout.Slot[*member] `layout:"items"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "items", "found"), 0o755); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := space.Scope().Inspect(ctx, InspectionPolicy{Traversal: Discovered}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect error = %v, want canceled", err)
	}
	if tree.Items.Len() != 0 {
		t.Fatal("already canceled inspection populated collection cache")
	}
}

func TestManagementScaffoldPreservesExecutableKind(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Root  layout.Dir  `layout:"."`
		Build layout.Exec `layout:"build.sh"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := space.Scope().Scaffold(context.Background(), ScaffoldPolicy{Files: true}); err != nil {
		t.Fatal(err)
	}
	if !tree.Build.IsExecutable() {
		t.Fatal("Scaffold created a declared executable without execution permissions")
	}
}

func TestManagementLoadFailFastReportsDirtyPath(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Root     layout.Dir                          `layout:"."`
		Manifest formats.JSONFile[map[string]string] `layout:"manifest.json"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	tree.Manifest.Set(map[string]string{"dirty": "content"})
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := space.Scope().Load(context.Background(), LoadPolicy{})
	if !errors.Is(err, ErrDirty) {
		t.Fatalf("Load error = %v, want dirty conflict", err)
	}
	if len(out.Effects) != 1 || out.Effects[0].Path != tree.Manifest.Path() || !errors.Is(out.Effects[0].Err, ErrDirty) {
		t.Fatalf("Load lost failing path: %#v", out)
	}
}

func TestManagementBootstrapPlansRequiredStructure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	type plannedLayout struct {
		Root     layout.Dir                          `layout:"." manage:"required"`
		Marker   layout.Dir                          `layout:".project" manage:"detect,required"`
		Empty    layout.File                         `layout:".project/ready" manage:"empty,required"`
		Manifest formats.JSONFile[map[string]string] `layout:"manifest.json" manage:"required"`
	}
	var tree plannedLayout
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	tree.Manifest.Set(map[string]string{"name": "bootstrap"})
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Bootstrap(context.Background(), space.Scope(), struct{}{}, Initialization[plannedLayout, struct{}]{}, BootstrapPolicy{})
	if err != nil {
		t.Fatalf("planned required structure blocked bootstrap: %v", err)
	}
	if !out.Applied() {
		t.Fatal("bootstrap did not report applied effects")
	}
	for _, path := range []string{tree.Root.Path(), tree.Marker.Path(), tree.Empty.Path(), tree.Manifest.Path()} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing initialized participant %s: %v", path, err)
		}
	}
	if _, err := space.Scope().Validate(context.Background(), ValidationPolicy{Source: Disk}); err != nil {
		t.Fatalf("initialized layout did not validate on disk: %v", err)
	}
}

func TestManagementListCollectAllRetainsValidationFailures(t *testing.T) {
	type member struct {
		Root     layout.Dir                          `layout:"."`
		Manifest formats.JSONFile[map[string]string] `layout:"manifest.json"`
	}
	root := t.TempDir()
	var tree struct {
		Root  layout.Dir           `layout:"."`
		Items layout.Slot[*member] `layout:"items"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	memberRoot := filepath.Join(root, "items", "bad")
	if err := os.MkdirAll(memberRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memberRoot, "manifest.json"), []byte("{ invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	collection, err := BindCollection(space, &tree.Items, CollectionOptions{Scope: ScopeOptions{Ops: []Operation{OpSave}}})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := collection.List(context.Background(), ListPolicy{Validate: true, Diagnostics: CollectAll})
	if err == nil || !inventory.Diagnostics.HasErrors() {
		t.Fatal("collect-all List silently discarded child validation failure")
	}
	if errors.Is(err, ErrUnsupported) {
		t.Fatalf("List required child to publicly expose Validate: %v", err)
	}
}

func TestManagementDiskValidationPreservesUnserializedCache(t *testing.T) {
	type content struct {
		Name    string `json:"name"`
		Runtime *int   `json:"-"`
	}
	root := t.TempDir()
	var tree struct {
		Root     layout.Dir                `layout:"."`
		Manifest formats.JSONFile[content] `layout:"manifest.json"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tree.Manifest.Path(), []byte(`{"name":"disk"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runtime := 42
	tree.Manifest.Set(content{Name: "memory", Runtime: &runtime})
	space, err := BindSpace(layout.NewDir(root), &tree, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := space.Scope().Validate(context.Background(), ValidationPolicy{Source: Disk}); err != nil {
		t.Fatal(err)
	}
	got := tree.Manifest.MustGet()
	if got.Name != "memory" || got.Runtime != &runtime || tree.Manifest.MemoryState() != layout.MemoryDirty || tree.Manifest.DiskState() != layout.DiskUnknown {
		t.Fatalf("disk validation changed existing cache: %#v", got)
	}
}
