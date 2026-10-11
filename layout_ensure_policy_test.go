package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLayoutPolicyCanLimitMaterializedNodeKinds(t *testing.T) {
	type root struct {
		Assets Dir         `layout:"dir" path:"assets"`
		Readme File        `layout:"file" path:"README.md"`
		Config testMapFile `layout:"content" path:"config.json"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	var report Report
	ctx := DefaultContext
	ctx.EnsurePolicy = EnsureDirs
	ctx.Reporter = &report

	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	if _, err := os.Stat(layout.Assets.Path()); err != nil {
		t.Fatalf("os.Stat(assets) error = %v", err)
	}
	if _, err := os.Stat(layout.Readme.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(readme) err = %v, want not-exist", err)
	}
	if _, err := os.Stat(layout.Config.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(config) err = %v, want not-exist", err)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpEnsure, Path: layout.Assets.Path(), Result: EnsureEnsured},
		{Op: OpEnsure, Path: layout.Readme.Path(), Result: EnsureSkippedPolicy},
		{Op: OpEnsure, Path: layout.Config.Path(), Result: EnsureSkippedPolicy},
	})
}

func TestEnsureLayoutPolicyCanEnsureSyncablesWithoutRawFiles(t *testing.T) {
	type root struct {
		Readme File        `layout:"file" path:"README.md"`
		Config testMapFile `layout:"content" path:"config.json"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	var report Report
	ctx := DefaultContext
	ctx.EnsurePolicy = EnsureSyncables
	ctx.Reporter = &report

	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	if _, err := os.Stat(layout.Readme.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(readme) err = %v, want not-exist", err)
	}
	if _, err := os.Stat(layout.Config.Path()); err != nil {
		t.Fatalf("os.Stat(config) error = %v", err)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpEnsure, Path: layout.Readme.Path(), Result: EnsureSkippedPolicy},
		{Op: OpEnsure, Path: layout.Config.Path(), Result: EnsureEnsured},
	})
}

func TestLayoutEnsureThenSyncDirectoryMembers(t *testing.T) {
	type item struct {
		Root   Dir         `layout:"dir" path:"."`
		Config testMapFile `layout:"content" path:"config.json"`
	}

	type root struct {
		Root  Dir         `layout:"dir" path:"."`
		Items Slot[*item] `layout:"directories" path:"items"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	api, err := layout.Items.ComposeMember("api")
	if err != nil {
		t.Fatalf("Items.ComposeMember() error = %v", err)
	}
	api.Config.Set(map[string]string{"name": "api"})

	var report Report
	ctx := DefaultContext
	ctx.EnsurePolicy = EnsureDirs
	ctx.Reporter = &report

	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(base, "items", "api")); err != nil {
		t.Fatalf("os.Stat(item root) error = %v", err)
	}
	if _, err := os.Stat(api.Config.Path()); err != nil {
		t.Fatalf("os.Stat(config) error = %v", err)
	}

	assertHasEntry(t, report.Entries(), Entry{Op: OpEnsure, Path: filepath.Join(base, "items", "api"), Result: EnsureEnsured})
	assertHasEntry(t, report.Entries(), Entry{Op: OpEnsure, Path: api.Config.Path(), Result: EnsureSkippedPolicy})
	assertHasEntry(t, report.Entries(), Entry{Op: OpSync, Path: api.Config.Path(), Result: SyncWritten})
}

func TestLayoutEnsureThenSyncDocumentMembers(t *testing.T) {
	type root struct {
		Configs FileSlot[*testMapFile] `layout:"documents" path:"configs"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	cfg, err := layout.Configs.ComposeMember("api.json")
	if err != nil {
		t.Fatalf("Configs.ComposeMember() error = %v", err)
	}
	cfg.Set(map[string]string{"name": "api"})

	var report Report
	ctx := DefaultContext
	ctx.EnsurePolicy = EnsureDirs
	ctx.Reporter = &report

	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if _, err := os.Stat(cfg.Path()); err != nil {
		t.Fatalf("os.Stat(config) error = %v", err)
	}

	assertHasEntry(t, report.Entries(), Entry{Op: OpEnsure, Path: cfg.Path(), Result: EnsureSkippedPolicy})
	assertHasEntry(t, report.Entries(), Entry{Op: OpSync, Path: cfg.Path(), Result: SyncWritten})
}
