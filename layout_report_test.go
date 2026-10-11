package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureLayoutReportsVisitedNodes(t *testing.T) {
	type root struct {
		Assets Dir         `layout:"dir" path:"assets"`
		Readme File        `layout:"file" path:"README.md"`
		Config testMapFile `layout:"content" path:"config.json"`
	}

	base := t.TempDir()

	manager, _ := composeTree[root](t, base)

	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report

	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpEnsure, Path: filepath.Join(base, "assets"), Result: EnsureEnsured},
		{Op: OpEnsure, Path: filepath.Join(base, "README.md"), Result: EnsureEnsured},
		{Op: OpEnsure, Path: filepath.Join(base, "config.json"), Result: EnsureEnsured},
	})
}

func TestLoadLayoutReportsTypedRawAndSlotNodes(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"content" path:"config.json"`
	}

	type root struct {
		Raw      File        `layout:"file" path:"raw.txt"`
		Config   testMapFile `layout:"content" path:"config.json"`
		Services Slot[*item] `layout:"directories" path:"services"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	if err := os.WriteFile(layout.Raw.Path(), []byte("raw"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(raw) error = %v", err)
	}
	if err := os.WriteFile(layout.Config.Path(), []byte(`{"name":"root"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(config) error = %v", err)
	}

	svcPath := filepath.Join(base, "services", "api", "config.json")
	if err := os.MkdirAll(filepath.Dir(svcPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(svcPath, []byte(`{"name":"api"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(service) error = %v", err)
	}

	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report

	if _, err := manager.Load(ctx); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpLoad, Path: filepath.Join(base, "raw.txt"), Result: LoadNotApplicable},
		{Op: OpLoad, Path: filepath.Join(base, "config.json"), Result: LoadLoaded},
		{Op: OpLoad, Path: filepath.Join(base, "services", "api", "config.json"), Result: LoadLoaded},
		{Op: OpLoad, Path: filepath.Join(base, "services"), Result: LoadTraversed},
	})
}

func TestDiscoverAndScanReportStates(t *testing.T) {
	t.Run("discover", func(t *testing.T) {
		manager, tree := composeTree[reportFileTree](t, t.TempDir())
		f := &tree.Config

		if err := os.WriteFile(f.Path(), []byte(`{"name":"value"}`), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := manager.Discover(ctx); err != nil {
			t.Fatalf("Discover() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpDiscover, Path: f.Path(), Result: DiscoverPresent},
		})
	})

	t.Run("scan", func(t *testing.T) {
		manager, tree := composeTree[reportFileTree](t, t.TempDir())
		f := &tree.Config

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := manager.Scan(ctx); err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpScan, Path: f.Path(), Result: ScanMissing},
		})
	})
}

func TestSyncLayoutReportsWrittenAndSkippedStates(t *testing.T) {
	t.Run("written", func(t *testing.T) {
		manager, tree := composeTree[reportFileTree](t, t.TempDir())
		f := &tree.Config
		f.Set(map[string]string{"name": "value"})

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := manager.Sync(ctx); err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpSync, Path: f.Path(), Result: SyncWritten},
		})
	})

	t.Run("skipped_no_content", func(t *testing.T) {
		manager, tree := composeTree[reportFileTree](t, t.TempDir())
		f := &tree.Config

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := manager.Sync(ctx); err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpSync, Path: f.Path(), Result: SyncSkippedNoContent},
		})
	})

	t.Run("skipped_policy", func(t *testing.T) {
		manager, tree := composeTree[reportFileTree](t, t.TempDir())
		f := &tree.Config

		if err := os.WriteFile(f.Path(), []byte(`{"name":"disk"}`), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}
		if _, err := f.Load(); err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		var report Report
		ctx := DefaultContext
		ctx.SyncPolicy = SyncIfDirty
		ctx.Reporter = &report

		if _, err := manager.Sync(ctx); err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpSync, Path: f.Path(), Result: SyncSkippedPolicy},
		})
	})
}

func TestLoadLayoutReportStopsOnFirstError(t *testing.T) {
	type root struct {
		First  testMapFile       `layout:"content" path:"first.json"`
		Broken testLoadErrorFile `layout:"content" path:"broken.json"`
		Last   testMapFile       `layout:"content" path:"last.json"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	if err := os.WriteFile(layout.First.Path(), []byte(`{"name":"first"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(first) error = %v", err)
	}
	if err := os.WriteFile(layout.Last.Path(), []byte(`{"name":"last"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(last) error = %v", err)
	}

	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report

	_, err := manager.Load(ctx)
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "load failed") {
		t.Fatalf("Load() error = %v, want load failed", err)
	}

	entries := report.Entries()
	if len(entries) != 2 {
		t.Fatalf("len(report.Entries()) = %d, want 2", len(entries))
	}
	if entries[0].Path != layout.First.Path() || entries[0].Result != LoadLoaded || entries[0].Err != nil {
		t.Fatalf("first entry = %#v, want loaded %q", entries[0], layout.First.Path())
	}
	if entries[1].Path != layout.Broken.Path() || entries[1].Result != LoadFailed || entries[1].Err == nil {
		t.Fatalf("second entry = %#v, want failed %q", entries[1], layout.Broken.Path())
	}
	if report.HasErrors() != true {
		t.Fatal("report.HasErrors() = false, want true")
	}
}

type reportFileTree struct {
	Config testMapFile `layout:"content" path:"config.json"`
}
