package conduit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qlustra/conduit/layout"
)

type contractGoFile struct {
	layout.File
}

func TestFileContractComposesAndEnsuresBuiltinsAndWrappers(t *testing.T) {
	type tree struct {
		Plain   layout.File     `layout:"file" path:"plain.txt"`
		Pointer *layout.File    `layout:"file" path:"pointer.txt"`
		Go      contractGoFile  `layout:"file,required" path:"main.go"`
		GoPtr   *contractGoFile `layout:"file" path:"nested/test.go"`
	}
	base := t.TempDir()
	manager, model := composeTree[tree](t, base)
	if model.Pointer == nil || model.GoPtr == nil {
		t.Fatal("Compose() did not allocate pointer file carriers")
	}
	nodes := []struct {
		file layout.File
		path string
	}{
		{model.Plain, "plain.txt"},
		{*model.Pointer, "pointer.txt"},
		{model.Go.File, "main.go"},
		{model.GoPtr.File, "nested/test.go"},
	}
	for _, node := range nodes {
		if got, want := node.file.Path(), filepath.Join(base, node.path); got != want {
			t.Fatalf("Path() = %q, want %q", got, want)
		}
		if got, ok := node.file.DeclaredPath(); !ok || got != node.path {
			t.Fatalf("DeclaredPath() = %q, %v; want %q, true", got, ok, node.path)
		}
		if got, ok := node.file.ComposedBaseDir(); !ok || got.Path() != base {
			t.Fatalf("ComposedBaseDir() = %q, %v; want %q, true", got.Path(), ok, base)
		}
		if node.file.Exists() {
			t.Fatalf("Compose() created %q", node.file.Path())
		}
	}
	if _, err := manager.Ensure(layout.DefaultContext); err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		data, err := node.file.ReadBytes()
		if err != nil || len(data) != 0 {
			t.Fatalf("ReadBytes(%q) = %q, %v; want empty file", node.file.Path(), data, err)
		}
	}
}

type contractEnsureFile struct {
	layout.File
	calls int
}

func (f *contractEnsureFile) Ensure(ctx layout.Context) error {
	f.calls++
	return f.File.Ensure(ctx)
}

func TestFileContractEnsurePolicyGatesWrapperHook(t *testing.T) {
	type tree struct {
		Go contractEnsureFile `layout:"file" path:"main.go"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	var report layout.Report
	ctx := layout.DefaultContext
	ctx.EnsurePolicy = layout.EnsureNone
	ctx.Reporter = &report
	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if model.Go.calls != 0 || model.Go.Exists() {
		t.Fatal("Ensure() invoked the file hook or created a file with EnsureNone")
	}
	assertEntries(t, report.Entries(), []layout.Entry{
		{Op: layout.OpEnsure, Path: model.Go.Path(), Result: layout.EnsureSkippedPolicy},
	})
	ctx.EnsurePolicy = layout.EnsureFiles
	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if model.Go.calls != 1 || !model.Go.Exists() {
		t.Fatal("Ensure() did not invoke the file hook and create the file with EnsureFiles")
	}
}

func TestFileContractReportsWrappersAsNotApplicable(t *testing.T) {
	type tree struct {
		Plain layout.File     `layout:"file" path:"plain.txt"`
		Go    contractGoFile  `layout:"file" path:"main.go"`
		GoPtr *contractGoFile `layout:"file" path:"test.go"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	for _, test := range []struct {
		name      string
		operation func(layout.Context) (layout.ResultCode, error)
		op        layout.Operation
		result    layout.ResultCode
	}{
		{"scan", manager.Scan, layout.OpScan, layout.ScanNotApplicable},
		{"load", manager.Load, layout.OpLoad, layout.LoadNotApplicable},
		{"discover", manager.Discover, layout.OpDiscover, layout.DiscoverNotApplicable},
		{"sync", manager.Sync, layout.OpSync, layout.SyncNotApplicable},
	} {
		t.Run(test.name, func(t *testing.T) {
			var report layout.Report
			ctx := layout.DefaultContext
			ctx.Reporter = &report
			if _, err := test.operation(ctx); err != nil {
				t.Fatal(err)
			}
			assertEntries(t, report.Entries(), []layout.Entry{
				{Op: test.op, Path: model.Plain.Path(), Result: test.result},
				{Op: test.op, Path: model.Go.Path(), Result: test.result},
				{Op: test.op, Path: model.GoPtr.Path(), Result: test.result},
			})
		})
	}
	if _, err := os.Stat(model.Go.Path()); !os.IsNotExist(err) {
		t.Fatalf("operations created a raw file: %v", err)
	}
}

func TestFileContractRejectsOtherNodeKinds(t *testing.T) {
	type directory struct {
		Node layout.Dir `layout:"file" path:"node"`
	}
	type link struct {
		Node layout.Link `layout:"file" path:"node"`
	}
	type collection struct {
		Node layout.Slot[*struct{}] `layout:"file" path:"node"`
	}
	for _, test := range []struct {
		name    string
		compile func() error
	}{
		{"directory", compileRoleError[directory]},
		{"link", compileRoleError[link]},
		{"collection", compileRoleError[collection]},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.compile(); err == nil {
				t.Fatal("Compile() accepted a carrier without file capabilities")
			}
		})
	}
}

type contractPartialFile struct{}

func (contractPartialFile) Path() string               { return "" }
func (contractPartialFile) ReadBytes() ([]byte, error) { return nil, nil }

type contractIncompatibleFile struct {
	layout.File
}

func (contractIncompatibleFile) WriteBytes(string, layout.Context) error { return nil }

func TestFileContractRequiresCompleteFileAPI(t *testing.T) {
	type partial struct {
		Node contractPartialFile `layout:"file" path:"node"`
	}
	type incompatible struct {
		Node contractIncompatibleFile `layout:"file" path:"node"`
	}
	for _, test := range []struct {
		name    string
		compile func() error
	}{
		{"partial API", compileRoleError[partial]},
		{"incompatible override", compileRoleError[incompatible]},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.compile(); err == nil {
				t.Fatal("Compile() accepted a carrier without File's complete compatible API")
			}
		})
	}
}

func TestFileContractDoesNotPruneGroupsEmbeddingFile(t *testing.T) {
	type group struct {
		layout.File
		Config testMapFile `layout:"content" path:"config.json"`
	}
	type tree struct {
		layout.File
		Group group `layout:"group" path:"nested"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	for _, test := range []struct {
		name      string
		operation func(layout.Context) (layout.ResultCode, error)
		op        layout.Operation
	}{
		{"scan", manager.Scan, layout.OpScan},
		{"load", manager.Load, layout.OpLoad},
		{"discover", manager.Discover, layout.OpDiscover},
		{"sync", manager.Sync, layout.OpSync},
	} {
		t.Run(test.name, func(t *testing.T) {
			var report layout.Report
			ctx := layout.DefaultContext
			ctx.Reporter = &report
			if _, err := test.operation(ctx); err != nil {
				t.Fatal(err)
			}
			entries := report.Entries()
			if len(entries) != 1 || entries[0].Op != test.op || entries[0].Path != model.Group.Config.Path() {
				t.Fatalf("operation did not reach the group's content: %#v", entries)
			}
		})
	}
}
