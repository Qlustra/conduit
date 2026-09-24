package mgmt_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qlustra/conduit/formats"
	"github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/mgmt"
)

type workflowManifest struct {
	Name string `json:"name"`
}
type workflowConfig struct {
	formats.JSONFile[workflowManifest]
}

func (f *workflowConfig) Default() error { f.SetDefault(workflowManifest{Name: "default"}); return nil }
func (f *workflowConfig) Validate(options layout.ValidateOptions) error {
	if err := f.JSONFile.Validate(options); err != nil {
		return err
	}
	if v, ok := f.Get(); ok && v.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

type workflowText struct {
	layout.TextTemplate[workflowManifest]
}

func (f *workflowText) Template() string { return "name={{.Name}}\n" }

type workflowMember struct {
	Root     layout.Dir     `layout:"."`
	Manifest workflowConfig `layout:"manifest.json" manage:"required"`
	Seed     workflowText   `layout:"notes.txt" manage:"role=seeded"`
	Derived  workflowText   `layout:"summary.txt" manage:"role=derived"`
}

func workflowInit() mgmt.Initialization[workflowMember, string] {
	return mgmt.Initialization[workflowMember, string]{
		Initialize: func(ctx context.Context, tree *workflowMember, name string) error {
			tree.Manifest.Set(workflowManifest{Name: name})
			return nil
		},
		Context: func(ctx context.Context, tree *workflowMember) error {
			v := tree.Manifest.MustGet()
			tree.Seed.SetContext(v)
			tree.Derived.SetContext(v)
			return nil
		},
	}
}
func workflowSpace(t *testing.T, root string) (*workflowMember, mgmt.Space[workflowMember]) {
	t.Helper()
	tree := new(workflowMember)
	if err := layout.Compose(root, tree); err != nil {
		t.Fatal(err)
	}
	space, err := mgmt.BindSpace(layout.NewDir(root), tree, mgmt.SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return tree, space
}
func readWorkflow(t *testing.T, path string) string {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func writeWorkflow(t *testing.T, path, value string) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, []byte(value), 0644); e != nil {
		t.Fatal(e)
	}
}
func TestBootstrapUsesExistingAuthorityAndPreservesSeededBytes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new")
	tree, space := workflowSpace(t, root)
	out, err := mgmt.Bootstrap(context.Background(), space.Scope(), "initial", workflowInit(), mgmt.BootstrapPolicy{})
	if err != nil || !out.Applied() {
		t.Fatalf("bootstrap: %+v, %v", out, err)
	}
	if got := readWorkflow(t, tree.Derived.Path()); got != "name=initial\n" {
		t.Fatal(got)
	}
	manifest := `{ "name": "disk-authority" }` + "\n"
	writeWorkflow(t, tree.Manifest.Path(), manifest)
	writeWorkflow(t, tree.Seed.Path(), "user-written notes\n")
	if err := os.Remove(tree.Derived.Path()); err != nil {
		t.Fatal(err)
	}
	out, err = mgmt.Bootstrap(context.Background(), space.Scope(), "ignored-input", workflowInit(), mgmt.BootstrapPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if got := readWorkflow(t, tree.Manifest.Path()); got != manifest {
		t.Fatal("existing manifest rewritten", got)
	}
	if got := readWorkflow(t, tree.Seed.Path()); got != "user-written notes\n" {
		t.Fatal("seed overwritten", got)
	}
	if got := readWorkflow(t, tree.Derived.Path()); got != "name=disk-authority\n" {
		t.Fatal("context did not use preserved content", got)
	}
	out, err = mgmt.Bootstrap(context.Background(), space.Scope(), "ignored", workflowInit(), mgmt.BootstrapPolicy{})
	if err != nil || out.Applied() {
		t.Fatalf("repeat bootstrap should preserve all bytes: %+v, %v", out, err)
	}
}
func TestBootstrapPreparationErrorsNeverCreateStorage(t *testing.T) {
	for _, failure := range []string{"invalid-input", "malformed-existing", "initializer-error"} {
		t.Run(failure, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "new")
			tree, space := workflowSpace(t, root)
			init := workflowInit()
			name := "valid"
			switch failure {
			case "invalid-input":
				name = ""
			case "malformed-existing":
				writeWorkflow(t, tree.Manifest.Path(), "{")
			case "initializer-error":
				init.Initialize = func(context.Context, *workflowMember, string) error { return errors.New("rejected") }
			}
			out, err := mgmt.Bootstrap(context.Background(), space.Scope(), name, init, mgmt.BootstrapPolicy{})
			if err == nil || out.Applied() {
				t.Fatalf("expected preparation failure without effects: %+v %v", out, err)
			}
			if _, e := os.Stat(tree.Seed.Path()); !os.IsNotExist(e) {
				t.Fatal("seed was created", e)
			}
			if failure != "malformed-existing" {
				if _, e := os.Stat(root); !os.IsNotExist(e) {
					t.Fatal("root was created", e)
				}
			}
		})
	}
}
func TestCollectionCreateAdmissionAndPostApplyFailure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	var tree struct {
		Members layout.Slot[*workflowMember] `layout:"members"`
	}
	if err := layout.Compose(root, &tree); err != nil {
		t.Fatal(err)
	}
	space, err := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	members, err := mgmt.BindCollection(space, &tree.Members, mgmt.CollectionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	recipe := mgmt.Creation[workflowMember, string](workflowInit())
	failed, err := mgmt.Create(ctx, members, "bad", "", recipe, mgmt.CreatePolicy{})
	if err == nil || failed.Outcome.Applied() {
		t.Fatal("invalid creation had effects", failed, err)
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("invalid creation made root", e)
	}
	created, err := mgmt.Create(ctx, members, "good", "Conduit", recipe, mgmt.CreatePolicy{})
	if err != nil || !created.Outcome.Applied() {
		t.Fatal(created, err)
	}
	again, err := members.At("good", mgmt.BindPolicy{RequireExisting: true})
	if err != nil {
		t.Fatal(err)
	}
	if again.Layout() != created.Value.Layout() {
		t.Fatal("At did not retain cached layout")
	}
	if _, err = mgmt.Create(ctx, members, "good", "other", recipe, mgmt.CreatePolicy{}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("collision = %v", err)
	}
	hookErr := errors.New("index unavailable")
	recipe.AfterApply = func(context.Context, mgmt.Outcome) error { return hookErr }
	created, err = mgmt.Create(ctx, members, "committed", "post", recipe, mgmt.CreatePolicy{})
	var after *mgmt.PostApplyError
	if !errors.As(err, &after) || !errors.Is(err, hookErr) || created.Value == nil || !created.Outcome.Applied() {
		t.Fatalf("post-apply result lost: %+v %v", created, err)
	}
	if !strings.Contains(readWorkflow(t, created.Value.Layout().Manifest.Path()), "post") {
		t.Fatal("post-apply failure lost committed content")
	}
	// Cached phantom does not appear in physical listing.
	inventory, err := members.List(ctx, mgmt.ListPolicy{Validate: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Entries) != 2 {
		t.Fatalf("physical listing includes phantom: %+v", inventory)
	}
}
func TestSavePreparesAllBytesAndDoesNotRewriteLoadedSiblings(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		First  formats.JSONFile[map[string]any] `layout:"first.json"`
		Second formats.JSONFile[map[string]any] `layout:"second.json"`
		Unused layout.Dir                       `layout:"unused"`
	}
	if e := layout.Compose(root, &tree); e != nil {
		t.Fatal(e)
	}
	space, e := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{})
	if e != nil {
		t.Fatal(e)
	}
	original := `{ "value": "original" }`
	writeWorkflow(t, tree.First.Path(), original)
	writeWorkflow(t, tree.Second.Path(), original)
	if _, e = space.Scope().Load(context.Background(), mgmt.LoadPolicy{}); e != nil {
		t.Fatal(e)
	}
	tree.Second.Set(map[string]any{"value": "changed"})
	if _, e = space.Scope().Save(context.Background(), mgmt.SavePolicy{}); e != nil {
		t.Fatal(e)
	}
	if got := readWorkflow(t, tree.First.Path()); got != original {
		t.Fatal("loaded sibling rewritten", got)
	}
	if _, e = os.Stat(tree.Unused.Path()); !os.IsNotExist(e) {
		t.Fatal("Save scaffolded unused structure", e)
	}
	before := readWorkflow(t, tree.Second.Path())
	tree.First.Set(map[string]any{"value": "not committed"})
	tree.Second.Set(map[string]any{"unsupported": make(chan int)})
	out, e := space.Scope().Save(context.Background(), mgmt.SavePolicy{})
	if e == nil || out.Applied() {
		t.Fatalf("encoding failure allowed writes: %+v %v", out, e)
	}
	if readWorkflow(t, tree.First.Path()) != original || readWorkflow(t, tree.Second.Path()) != before {
		t.Fatal("encoding failed after partial writes")
	}
}
func TestDiskValidationCollectsIndependentFailuresAndPreservesEdits(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		First  workflowConfig `layout:"first.json" manage:"required"`
		Second workflowConfig `layout:"second.json" manage:"required"`
	}
	if e := layout.Compose(root, &tree); e != nil {
		t.Fatal(e)
	}
	space, e := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{})
	if e != nil {
		t.Fatal(e)
	}
	tree.First.Set(workflowManifest{Name: "live"})
	tree.Second.Set(workflowManifest{Name: "also live"})
	writeWorkflow(t, tree.First.Path(), "{")
	writeWorkflow(t, tree.Second.Path(), `{"name":""}`)
	d, e := space.Scope().Validate(context.Background(), mgmt.ValidationPolicy{Source: mgmt.Disk, Diagnostics: mgmt.CollectAll})
	if e == nil || len(d.Entries) != 2 {
		t.Fatalf("independent diagnostics: %+v %v", d, e)
	}
	if tree.First.MustGet().Name != "live" || tree.Second.MustGet().Name != "also live" || tree.Second.MemoryState() != layout.MemoryDirty {
		t.Fatal("disk validation changed live edits")
	}
}
func TestRefreshChangesOnlyDerivedOutputs(t *testing.T) {
	tree, space := workflowSpace(t, filepath.Join(t.TempDir(), "workspace"))
	ctx := context.Background()
	if _, e := mgmt.Bootstrap(ctx, space.Scope(), "original", workflowInit(), mgmt.BootstrapPolicy{}); e != nil {
		t.Fatal(e)
	}
	writeWorkflow(t, tree.Seed.Path(), "authored")
	tree.Derived.SetContext(workflowManifest{Name: "next"})
	if _, e := space.Scope().Refresh(ctx, mgmt.RefreshPolicy{Effects: mgmt.MemoryOnly}); e != nil {
		t.Fatal(e)
	}
	if readWorkflow(t, tree.Derived.Path()) != "name=original\n" {
		t.Fatal("memory refresh persisted")
	}
	if _, e := space.Scope().Refresh(ctx, mgmt.RefreshPolicy{}); e != nil {
		t.Fatal(e)
	}
	if readWorkflow(t, tree.Derived.Path()) != "name=next\n" || readWorkflow(t, tree.Seed.Path()) != "authored" {
		t.Fatal("derived selection incorrect")
	}
}
func TestProjectDetectionStopsAtBrokenNearestBoundary(t *testing.T) {
	type projectLayout struct {
		Marker workflowConfig `layout:"project.json" manage:"detect,required"`
	}
	root := t.TempDir()
	writeWorkflow(t, filepath.Join(root, "project.json"), `{"name":"outer"}`)
	nested := filepath.Join(root, "nested")
	writeWorkflow(t, filepath.Join(nested, "project.json"), "malformed")
	if e := os.MkdirAll(filepath.Join(nested, "deep"), 0755); e != nil {
		t.Fatal(e)
	}
	p, d, e := mgmt.LocateProject[projectLayout](context.Background(), filepath.Join(nested, "deep"), mgmt.ProjectOptions{}, mgmt.DetectionPolicy{})
	if e != nil || !d.Match || p.Root().Path() != nested {
		t.Fatalf("presence detection %v %+v", e, d)
	}
	if _, e = p.Scope().Validate(context.Background(), mgmt.ValidationPolicy{Source: mgmt.Disk}); e == nil {
		t.Fatal("malformed detected marker validated")
	}
	if e = os.Remove(filepath.Join(nested, "project.json")); e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(filepath.Join(nested, "project.json"), 0755); e != nil {
		t.Fatal(e)
	}
	p, d, e = mgmt.LocateProject[projectLayout](context.Background(), filepath.Join(nested, "deep"), mgmt.ProjectOptions{}, mgmt.DetectionPolicy{})
	if e == nil || d.Match || !d.Boundary || p.Root().Path() != nested {
		t.Fatalf("skipped broken boundary: %v %+v", e, d)
	}
}

type dependentValidation struct {
	Manifest workflowConfig `layout:"manifest.json" manage:"required"`
}

func (v *dependentValidation) Validate(layout.ValidateOptions) error {
	_ = v.Manifest.MustGet()
	return nil
}
func TestRequiredAdmissionPrecedesContainerRules(t *testing.T) {
	root := t.TempDir()
	var tree dependentValidation
	if e := layout.Compose(root, &tree); e != nil {
		t.Fatal(e)
	}
	space, e := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{})
	if e != nil {
		t.Fatal(e)
	}
	for _, source := range []mgmt.Source{mgmt.Memory, mgmt.Disk} {
		d, e := space.Scope().Validate(context.Background(), mgmt.ValidationPolicy{Source: source, Diagnostics: mgmt.CollectAll})
		if e == nil || len(d.Entries) != 1 {
			t.Fatalf("missing input should block container rule: %+v %v", d, e)
		}
	}
}
func TestUnsupportedAtomicCreationIsRejectedBeforeAnySave(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Existing formats.JSONFile[string] `layout:"existing.json"`
		Missing  formats.JSONFile[string] `layout:"missing.json"`
	}
	if e := layout.Compose(root, &tree); e != nil {
		t.Fatal(e)
	}
	writeWorkflow(t, tree.Existing.Path(), `"original"`)
	tree.Existing.Set("new")
	tree.Missing.Set("new")
	space, e := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{Context: layout.Context{WritePolicy: layout.WriteAtomicReplace, TempFilePlacement: layout.TempFileAdjacent}})
	if e != nil {
		t.Fatal(e)
	}
	out, e := space.Scope().Save(context.Background(), mgmt.SavePolicy{Missing: mgmt.CreateMissing})
	if !errors.Is(e, mgmt.ErrUnsupported) || out.Applied() {
		t.Fatalf("unsupported creation policy partially saved: %+v %v", out, e)
	}
	if readWorkflow(t, tree.Existing.Path()) != `"original"` {
		t.Fatal("preceding member overwritten")
	}
}
func TestCreateConfinesManuallyReboundChildren(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Members layout.Slot[*workflowMember] `layout:"members"`
	}
	if e := layout.Compose(root, &tree); e != nil {
		t.Fatal(e)
	}
	space, e := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{})
	if e != nil {
		t.Fatal(e)
	}
	members, e := mgmt.BindCollection(space, &tree.Members, mgmt.CollectionOptions{})
	if e != nil {
		t.Fatal(e)
	}
	child, e := tree.Members.At("new")
	if e != nil {
		t.Fatal(e)
	}
	child.Manifest.ComposePath(filepath.Join(root, "outside-member.json"))
	out, e := mgmt.Create(context.Background(), members, "new", "input", mgmt.Creation[workflowMember, string](workflowInit()), mgmt.CreatePolicy{})
	if e == nil || out.Outcome.Applied() {
		t.Fatal("creation escaped selected member", out, e)
	}
	if _, e = os.Stat(child.Manifest.Path()); !os.IsNotExist(e) {
		t.Fatal("outside path touched", e)
	}
}
func TestScaffoldReportsChangedExecutablePermissions(t *testing.T) {
	root := t.TempDir()
	var tree struct {
		Tool layout.Exec `layout:"run" manage:"empty"`
	}
	if e := layout.Compose(root, &tree); e != nil {
		t.Fatal(e)
	}
	writeWorkflow(t, tree.Tool.Path(), "echo test\n")
	if e := os.Chmod(tree.Tool.Path(), 0600); e != nil {
		t.Fatal(e)
	}
	space, e := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{})
	if e != nil {
		t.Fatal(e)
	}
	out, e := space.Scope().Scaffold(context.Background(), mgmt.ScaffoldPolicy{})
	if e != nil || !out.Applied() {
		t.Fatalf("permission change not reported: %+v %v", out, e)
	}
}

func TestBindingRejectsImpossibleMissingStructure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	var tree struct {
		File   layout.File              `layout:"parent" manage:"empty"`
		Nested formats.JSONFile[string] `layout:"parent/child.json"`
	}
	if e := layout.Compose(root, &tree); e != nil {
		t.Fatal(e)
	}
	if _, e := mgmt.BindSpace(layout.NewDir(root), &tree, mgmt.SpaceOptions{}); e == nil {
		t.Fatal("bound file as another participant's parent")
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("binding touched storage", e)
	}
}
