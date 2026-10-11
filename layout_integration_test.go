package conduit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/qlustra/conduit/layout"
)

func TestUnloadClearsContentAndRetainsSlotMembership(t *testing.T) {
	type child struct {
		Config testMapFile `layout:"content" path:"config.json"`
	}
	type tree struct {
		Config    testMapFile            `layout:"content" path:"config.json"`
		Children  Slot[*child]           `layout:"directories" path:"children"`
		Documents FileSlot[*testMapFile] `layout:"documents" path:"documents"`
		Links     LinkSlot[*Link]        `layout:"links" path:"links"`
	}
	base := t.TempDir()
	manager, model := composeTree[tree](t, base)
	childNode := model.Children.MustComposeMember("cached")
	document := model.Documents.MustComposeMember("cached.json")
	link := model.Links.MustComposeMember("README")
	model.Config.Set(map[string]string{"name": "root"})
	childNode.Config.Set(map[string]string{"name": "child"})
	document.Set(map[string]string{"name": "document"})
	link.SetTarget("missing-target")
	if _, err := manager.Sync(DefaultContext); err != nil {
		t.Fatal(err)
	}
	if err := manager.Unload(); err != nil {
		t.Fatal(err)
	}
	if model.Config.HasContent() || childNode.Config.HasContent() || document.HasContent() || link.HasContent() {
		t.Fatal("cached content remains after Unload")
	}
	if model.Children.Len() != 1 || model.Documents.Len() != 1 || model.Links.Len() != 1 {
		t.Fatal("Unload evicted collection membership")
	}
	if cached, _ := model.Children.Get("cached"); cached != childNode {
		t.Fatal("Unload replaced the cached member")
	}
	if bytes, err := os.ReadFile(document.Path()); err != nil || !strings.Contains(string(bytes), "document") {
		t.Fatalf("Unload changed disk content: %q, %v", bytes, err)
	}
	if target, err := os.Readlink(link.Path()); err != nil || target != "missing-target" {
		t.Fatalf("Unload changed symlink: %q, %v", target, err)
	}
}

func TestLoadAndDiscoverAccumulateFreshMembersWithoutVisitingStaleCache(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"content" path:"config.json"`
	}
	type tree struct {
		Items     Slot[*item]            `layout:"directories" path:"items"`
		Documents FileSlot[*testMapFile] `layout:"documents" path:"documents"`
		Links     LinkSlot[*Link]        `layout:"links" path:"links"`
	}
	for _, operation := range []string{"load", "discover"} {
		t.Run(operation, func(t *testing.T) {
			base := t.TempDir()
			manager, model := composeTree[tree](t, base)
			writeMembers := func(name string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(base, "items", name), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(base, "documents"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(base, "links"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(base, "items", name, "config.json"), []byte(`{"name":"disk"}`), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(base, "documents", name+".json"), []byte(`{"name":"disk"}`), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("missing-target", filepath.Join(base, "links", name)); err != nil {
					t.Fatal(err)
				}
			}
			writeMembers("old")
			if _, err := manager.Load(DefaultContext); err != nil {
				t.Fatal(err)
			}
			oldItem, _ := model.Items.Get("old")
			oldDocument, _ := model.Documents.Get("old.json")
			oldLink, _ := model.Links.Get("old")
			oldItem.Config.Set(map[string]string{"name": "memory"})
			oldDocument.Set(map[string]string{"name": "memory"})
			oldLink.SetTarget("memory-target")
			for _, path := range []string{filepath.Join(base, "items", "old"), oldDocument.Path(), oldLink.Path()} {
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
			}
			writeMembers("fresh")
			// A directory collection admits physical directories only, including when
			// a symlink resolves to a directory.
			if err := os.WriteFile(filepath.Join(base, "items", "plain"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("fresh", filepath.Join(base, "items", "alias")); err != nil {
				t.Fatal(err)
			}
			run := manager.Load
			if operation == "discover" {
				run = manager.Discover
			}
			if _, err := run(DefaultContext); err != nil {
				t.Fatal(err)
			}
			if model.Items.Len() != 2 || model.Documents.Len() != 2 || model.Links.Len() != 2 {
				t.Fatal("fresh members replaced stale membership or admitted wrong entry kinds")
			}
			if got, _ := model.Items.Get("old"); got != oldItem {
				t.Fatal("cached directory identity changed")
			}
			if oldItem.Config.MustGet()["name"] != "memory" || oldDocument.MustGet()["name"] != "memory" || oldLink.MustTarget() != "memory-target" {
				t.Fatal("stale members received the filesystem operation")
			}
			if oldItem.Config.DiskState() != DiskPresent || oldDocument.DiskState() != DiskPresent || oldLink.DiskState() != DiskPresent {
				t.Fatal("stale members were rescanned")
			}
			freshDocument, _ := model.Documents.Get("fresh.json")
			freshLink, _ := model.Links.Get("fresh")
			if freshDocument.HasContent() != (operation == "load") || freshLink.HasContent() != (operation == "load") {
				t.Fatal("Load/Discover content boundary changed")
			}
			if freshDocument.DiskState() != DiskPresent || freshLink.DiskState() != DiskPresent {
				t.Fatal("fresh member disk state was not observed")
			}
		})
	}
}

func TestEnsurePreparesEmptyAndNilDirectoryMemberRoots(t *testing.T) {
	type tree struct {
		Items Slot[*struct{ Untagged bool }] `layout:"directories" path:"items"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	model.Items.MustComposeMember("empty")
	model.Items.Put("nil", nil)
	if _, err := manager.Ensure(DefaultContext); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"empty", "nil"} {
		if info, err := os.Stat(filepath.Join(model.Items.Path(), name)); err != nil || !info.IsDir() {
			t.Fatalf("member root %q: %v", name, err)
		}
	}
}

func TestSyncDoesNotPrepareEmptyCollectionRoots(t *testing.T) {
	type tree struct {
		Items Slot[*struct{ Untagged bool }] `layout:"directories" path:"items"`
	}
	base := filepath.Join(t.TempDir(), "uncreated")
	manager, model := composeTree[tree](t, base)
	model.Items.MustComposeMember("empty")
	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report
	if _, err := manager.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("Sync scaffolded empty members: %v", err)
	}
	assertEntries(t, report.Entries(), []Entry{{Op: OpSync, Path: model.Items.Path(), Result: SyncTraversed}})
}

func TestNestedSlotRootIsValidatedBeforeItsMembers(t *testing.T) {
	type tree struct {
		Items Slot[*FileSlot[*testMapFile]] `layout:"directories" path:"items"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	documents := model.Items.MustComposeMember("documents")
	if err := os.MkdirAll(model.Items.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(documents.Path(), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	var report Report
	code, err := manager.Validate(ValidateOptions{Reporter: &report})
	if err == nil || code != ValidateFailed {
		t.Fatalf("Validate() = %v, %v; want failure", code, err)
	}
	assertEntries(t, report.Entries(), []Entry{
		{Op: OpValidate, Path: documents.Path(), Result: ValidateFailed, Err: err},
		{Op: OpValidate, Path: model.Items.Path(), Result: ValidateFailed, Err: err},
	})
}

func TestLoadFailureReportsMemberAndEnclosingCollections(t *testing.T) {
	type tree struct {
		Items Slot[*FileSlot[*testMapFile]] `layout:"directories" path:"items"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	documents := model.Items.MustComposeMember("documents")
	if err := os.MkdirAll(documents.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(documents.Path(), "a-broken.json")
	if err := os.WriteFile(broken, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(documents.Path(), "z-later.json"), []byte(`{"name":"later"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report
	code, err := manager.Load(ctx)
	if err == nil || code != LoadFailed {
		t.Fatalf("Load() = %v, %v; want failure", code, err)
	}
	if documents.Len() != 1 {
		t.Fatalf("Load composed members after failure: %v", documents.Keys())
	}
	assertEntries(t, report.Entries(), []Entry{
		{Op: OpLoad, Path: broken, Result: LoadFailed, Err: err},
		{Op: OpLoad, Path: documents.Path(), Result: LoadFailed, Err: err},
		{Op: OpLoad, Path: model.Items.Path(), Result: LoadFailed, Err: err},
	})
}

type nestedTemplate struct{ testTemplateFile }

func (f *nestedTemplate) Default() error {
	f.SetDefaultContext(testRenderContext{Name: "nested", Items: []string{"value"}})
	return nil
}

func TestNestedSlotsParticipateInCachedTreeOperations(t *testing.T) {
	type tree struct {
		Items Slot[*FileSlot[*nestedTemplate]] `layout:"directories" path:"items"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	documents := model.Items.MustComposeMember("documents")
	template := documents.MustComposeMember("rendered.txt")
	if err := manager.Default(); err != nil {
		t.Fatal(err)
	}
	if !template.HasContext() {
		t.Fatal("Default did not reach the nested slot member")
	}
	if err := manager.Render(); err != nil {
		t.Fatal(err)
	}
	if got := template.MustGet(); got != "nested:value" {
		t.Fatalf("Render() stored %q", got)
	}
	if _, err := manager.Sync(DefaultContext); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(template.Path()); err != nil {
		t.Fatal(err)
	}
	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report
	if _, err := manager.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if template.DiskState() != DiskMissing || template.MustGet() != "nested:value" {
		t.Fatal("Scan did not reach or preserve the nested content")
	}
	assertEntries(t, report.Entries(), []Entry{
		{Op: OpScan, Path: template.Path(), Result: ScanMissing},
		{Op: OpScan, Path: documents.Path(), Result: ScanTraversed},
		{Op: OpScan, Path: model.Items.Path(), Result: ScanTraversed},
	})
	if err := manager.Unload(); err != nil {
		t.Fatal(err)
	}
	if template.HasContent() {
		t.Fatal("Unload did not reach the nested slot member")
	}
	if documents.Len() != 1 || model.Items.Len() != 1 {
		t.Fatal("Unload removed nested membership")
	}
}
