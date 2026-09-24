package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fixturePackage(t *testing.T, source string) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mod := "module example.com/generatedfixture\n\ngo 1.26\n\nrequire github.com/qlustra/conduit v0.0.0\n\nreplace github.com/qlustra/conduit => " + filepath.ToSlash(root) + "\n"
	for name, data := range map[string]string{"go.mod": mod, "fixture.go": source} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Let Go resolve the parent module's ordinary codec dependencies. Network
	// access is unnecessary when running within this repository's test setup.
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture dependencies: %v\n%s", err, data)
	}
	return dir
}

const integrationSource = `package fixture
import (
 "context"
 "github.com/qlustra/conduit/layout"
 "github.com/qlustra/conduit/formats"
 "github.com/qlustra/conduit/mgmt"
 "github.com/qlustra/conduit/spec"
)
type Manifest struct { Title string ` + "`json:\"title\"`" + ` }
type NotebookLayout struct {
 Root layout.Dir ` + "`layout:\".\"`" + `
 Manifest formats.JSONFile[Manifest] ` + "`layout:\"notebook.json\" manage:\"required\"`" + `
}
type LogsLayout struct { Root layout.Dir ` + "`layout:\".\"`" + ` }
type NotesIndex struct { layout.TextTemplate[struct{}] }
func (*NotesIndex) Template() string { return "generated index\n" }
type NotesLayout struct {
 Root layout.Dir ` + "`layout:\".\"`" + `
 Notebooks layout.Slot[*NotebookLayout] ` + "`layout:\"notebooks\"`" + `
 Index NotesIndex ` + "`layout:\"INDEX.md\" manage:\"role=derived\"`" + `
 Logs LogsLayout ` + "`layout:\"logs\"`" + `
}
type RepoLayout struct {
 Root layout.Dir ` + "`layout:\".\"`" + `
 Marker layout.Dir ` + "`layout:\".project\" manage:\"detect\"`" + `
 Notes *NotesLayout ` + "`layout:\"research\"`" + `
}
type NewNotebook struct { Title string }
var NotebookCreation = mgmt.Creation[NotebookLayout,NewNotebook]{
 Initialize:func(ctx context.Context, tree *NotebookLayout,input NewNotebook) error { tree.Manifest.Set(Manifest{Title:input.Title}); return nil },
}
var NotebookInitialization = mgmt.Initialization[NotebookLayout,NewNotebook]{
 Initialize:func(ctx context.Context, tree *NotebookLayout,input NewNotebook) error { tree.Manifest.Set(Manifest{Title:input.Title}); return nil },
}
type NotebookManagement struct {
 Scope spec.Scope[NotebookLayout] ` + "`bootstrap:\"NotebookInitialization\"`" + `
 Manifest spec.Document[Manifest] ` + "`scope:\"from=Scope;bind=Manifest\"`" + `
 ReadOnly spec.Document[Manifest] ` + "`scope:\"from=Scope;bind=Manifest\" ops:\"read\"`" + `
}
type NotesManagement struct {
 Space spec.Space[NotesLayout]
 Notebooks spec.Collection[NotebookManagement] ` + "`scope:\"from=Space;bind=Notebooks\" create:\"NotebookCreation\"`" + `
 Logs spec.Scope[LogsLayout] ` + "`scope:\"from=Space;bind=Logs\" ops:\"inspect,scaffold\"`" + `
 Later spec.Scope[LogsLayout] ` + "`scope:\"from=Whole;bind=Logs\" ops:\"inspect\"`" + `
 Whole spec.Scope[NotesLayout] ` + "`scope:\"from=Space;bind=.\" ops:\"inspect\"`" + `
}
func OpenForCLI(root string) (*Repository,error) { return NewRepository(root,mgmt.ProjectOptions{}) }
type RepositoryManagement struct {
 Project spec.Project[RepoLayout]
 Notes NotesManagement ` + "`scope:\"from=Project;bind=Notes\"`" + `
}
`

const integrationTest = `package fixture
import (
 "context"
 "os"
 "path/filepath"
 "reflect"
 "testing"
 "github.com/qlustra/conduit/mgmt"
)
func TestGeneratedWorkflow(t *testing.T) {
 ctx := context.Background()
 root := filepath.Join(t.TempDir(),"repo")
 repo,err := NewRepository(root,mgmt.ProjectOptions{})
 if err != nil { t.Fatal(err) }
 if _,err := os.Stat(root); !os.IsNotExist(err) { t.Fatalf("binding created storage: %v",err) }
 if !repo.Capabilities().Has(mgmt.OpDetect) { t.Fatal("Project omitted detection capability") }
 for _,op := range []mgmt.Operation{mgmt.OpAt,mgmt.OpList,mgmt.OpCreate} {
  if !repo.Notes.Notebooks.Capabilities().Has(op) { t.Fatalf("collection omitted %s capability",op) }
 }
 if caps := repo.Notes.Logs.Capabilities(); len(caps) != 2 || !caps.Has(mgmt.OpInspect) || !caps.Has(mgmt.OpScaffold) { t.Fatalf("scope capabilities ignore restriction: %v",caps) }
 if repo.Notes.Root().Path() != filepath.Join(root,"research") { t.Fatalf("wrong child Space root: %s",repo.Notes.Root().Path()) }
 if repo.Notes.Whole.Layout() != repo.Notes.Layout() { t.Fatal("overlapping scope copied layout") }
 if repo.Notes.Logs.Layout() != &repo.Notes.Layout().Logs { t.Fatal("scope copied layout") }
 if repo.Notes.Later.Layout() != repo.Notes.Logs.Layout() { t.Fatal("dependency ordering changed binding") }
 created,err := repo.Notes.Notebooks.Create(ctx,"conduit",NewNotebook{Title:"Design"},mgmt.CreatePolicy{Parent:mgmt.CreateMissing})
 if err != nil { t.Fatal(err) }
 if created.Value == nil { t.Fatal("missing generated created member") }
 value,err := created.Value.Manifest.Read(ctx,mgmt.ReadPolicy{})
 if err != nil || value.Title != "Design" { t.Fatalf("created manifest: %+v %v",value,err) }
 created.Value.Layout().Manifest.Set(Manifest{Title:"Unsaved"})
 again,err := repo.Notes.Notebooks.At("conduit",mgmt.BindPolicy{})
 if err != nil { t.Fatal(err) }
 if again.Layout() != created.Value.Layout() { t.Fatal("At discarded child state") }
 if value,ok := again.Layout().Manifest.Get(); !ok || value.Title != "Unsaved" { t.Fatal("At changed dirty content") }
 if _,ok := reflect.TypeOf(again.ReadOnly).MethodByName("Update"); ok { t.Fatal("read-only declaration exposes Update") }
 if caps := again.ReadOnly.Capabilities(); len(caps) != 1 || !caps.Has(mgmt.OpRead) { t.Fatalf("document capabilities ignore restriction: %v",caps) }
 if _,ok := reflect.TypeOf(repo.Notes.Logs).MethodByName("Save"); ok { t.Fatal("restricted scope exposes Save") }
 if _,err := again.Save(ctx,mgmt.SavePolicy{}); err != nil { t.Fatal(err) }
 value,err = again.ReadOnly.Read(ctx,mgmt.ReadPolicy{})
 if err != nil || value.Title != "Unsaved" { t.Fatalf("saved manifest: %+v %v",value,err) }
 if !repo.Notes.Capabilities().Has(mgmt.OpRefresh) { t.Fatal("Templatable output omitted refresh capability") }
 if _,err := repo.Notes.Refresh(ctx,mgmt.RefreshPolicy{}); err != nil { t.Fatal(err) }
 if data,err := os.ReadFile(filepath.Join(root,"research","INDEX.md")); err != nil || string(data) != "generated index\n" { t.Fatalf("generated Refresh: %q %v",data,err) }
 tree := repo.Layout(); tree.Notes = nil
 if _,err := BindRepository(repo.Root(),tree,mgmt.ProjectOptions{}); err == nil { t.Fatal("nil nested Space accepted") }
 inventory,err := repo.Notes.Notebooks.List(ctx,mgmt.ListPolicy{})
 if err != nil || len(inventory.Entries) != 1 { t.Fatalf("inventory: %+v %v",inventory,err) }
}
`

func TestGeneratedTopologyCompilesAndRuns(t *testing.T) {
	dir := fixturePackage(t, integrationSource)
	if err := generateFile(dir, "RepositoryManagement", "managed_gen.go"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "managed_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := generateFile(dir, "RepositoryManagement", "managed_gen.go"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, "managed_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("generation is not deterministic")
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture_test.go"), []byte(integrationTest), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated fixture: %v\n%s", err, data)
	}
}

func TestUnsupportedDeclarationsFailBeforeWriting(t *testing.T) {
	cases := []struct{ name, old, replacement, want string }{
		{"unknown_operation", `ops:"inspect,scaffold"`, `ops:"teleport"`, "operation \"teleport\" is not applicable"},
		{"refresh_without_derivation", `ops:"inspect,scaffold"`, `ops:"refresh"`, "operation \"refresh\" is not applicable"},
		{"invalid_binding", `bind=Logs`, `bind=Missing`, "field Missing does not exist"},
		{"wrong_scope_type", `spec.Scope[LogsLayout]`, `spec.Scope[NotebookLayout]`, "does not match declared layout"},
		{"invalid_recipe", `create:"NotebookCreation"`, `create:"MissingCreation"`, "must name a local variable"},
		{"unknown_binding_term", `from=Space;bind=Logs`, `from=Space;path=Logs`, "unknown scope term"},
		{"unsupported_project_tag", `Project spec.Project[RepoLayout]`, `Project spec.Project[RepoLayout] ` + "`project:\"detect=any(Marker)\"`", "unsupported project topology tag"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(integrationSource, tc.old, tc.replacement, 1)
			if source == integrationSource {
				t.Fatal("test replacement did not match")
			}
			dir := fixturePackage(t, source)
			err := generateFile(dir, "RepositoryManagement", "managed_gen.go")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
			if _, err := os.Stat(filepath.Join(dir, "managed_gen.go")); !os.IsNotExist(err) {
				t.Fatal("failed generation wrote an output file")
			}
		})
	}
}

func TestRefusesHandwrittenOutput(t *testing.T) {
	dir := fixturePackage(t, `package fixture
import("github.com/qlustra/conduit/layout"; "github.com/qlustra/conduit/spec")
type Tree struct { Root layout.Dir }
type TestManagement struct { Space spec.Space[Tree] }
`)
	path := filepath.Join(dir, "managed_gen.go")
	before := []byte("package fixture\n// Keep this handwritten file.\n")
	if err := os.WriteFile(path, before, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFile(dir, "TestManagement", "managed_gen.go"); err == nil || !strings.Contains(err.Error(), "non-generated") {
		t.Fatalf("got %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("handwritten output changed")
	}
}
