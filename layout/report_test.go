package layout

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testLoadErrorFile struct {
	file File
}

func (f *testLoadErrorFile) ComposePath(path string) {
	f.file.ComposePath(path)
}

func (f testLoadErrorFile) Path() string {
	return f.file.Path()
}

func (f testLoadErrorFile) Exists() bool {
	return f.file.Exists()
}

func (f *testLoadErrorFile) Load() (bool, error) {
	return false, errors.New("load failed")
}

func (f testLoadErrorFile) HasContent() bool {
	return false
}

func (f *testLoadErrorFile) Unload() {}

func TestEnsureDeepReportsVisitedNodes(t *testing.T) {
	type root struct {
		Assets Dir         `layout:"assets"`
		Readme File        `layout:"README.md"`
		Config testMapFile `layout:"config.json"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report

	if _, err := EnsureDeep(&layout, ctx); err != nil {
		t.Fatalf("EnsureDeep() error = %v", err)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpEnsure, Path: filepath.Join(base, "assets"), Result: EnsureEnsured},
		{Op: OpEnsure, Path: filepath.Join(base, "README.md"), Result: EnsureEnsured},
		{Op: OpEnsure, Path: filepath.Join(base, "config.json"), Result: EnsureEnsured},
	})
}

func TestEnsureDeepHandlesBareDirAndFileValues(t *testing.T) {
	base := t.TempDir()
	dir := NewDir(filepath.Join(base, "assets"))
	file := NewFile(filepath.Join(base, "README.md"))

	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report

	if _, err := EnsureDeep(dir, ctx); err != nil {
		t.Fatalf("EnsureDeep(dir) error = %v", err)
	}
	if _, err := EnsureDeep(file, ctx); err != nil {
		t.Fatalf("EnsureDeep(file) error = %v", err)
	}

	if _, err := os.Stat(dir.Path()); err != nil {
		t.Fatalf("os.Stat(dir) error = %v", err)
	}
	if _, err := os.Stat(file.Path()); err != nil {
		t.Fatalf("os.Stat(file) error = %v", err)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpEnsure, Path: dir.Path(), Result: EnsureEnsured},
		{Op: OpEnsure, Path: file.Path(), Result: EnsureEnsured},
	})
}

func TestLoadDeepReportsTypedRawAndSlotNodes(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}

	type root struct {
		Raw      File        `layout:"raw.txt"`
		Config   testMapFile `layout:"config.json"`
		Services Slot[*item] `layout:"services"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

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

	if _, err := LoadDeep(&layout, ctx); err != nil {
		t.Fatalf("LoadDeep() error = %v", err)
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
		var f testMapFile
		f.ComposePath(filepath.Join(t.TempDir(), "config.json"))

		if err := os.WriteFile(f.Path(), []byte(`{"name":"value"}`), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := DiscoverDeep(&f, ctx); err != nil {
			t.Fatalf("DiscoverDeep() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpDiscover, Path: f.Path(), Result: DiscoverPresent},
		})
	})

	t.Run("scan", func(t *testing.T) {
		var f testMapFile
		f.ComposePath(filepath.Join(t.TempDir(), "missing.json"))

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := ScanDeep(&f, ctx); err != nil {
			t.Fatalf("ScanDeep() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpScan, Path: f.Path(), Result: ScanMissing},
		})
	})
}

func TestSyncDeepReportsWrittenAndSkippedStates(t *testing.T) {
	t.Run("written", func(t *testing.T) {
		var f testMapFile
		f.ComposePath(filepath.Join(t.TempDir(), "config.json"))
		f.Set(map[string]string{"name": "value"})

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := SyncDeep(&f, ctx); err != nil {
			t.Fatalf("SyncDeep() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpSync, Path: f.Path(), Result: SyncWritten},
		})
	})

	t.Run("skipped_no_content", func(t *testing.T) {
		var f testMapFile
		f.ComposePath(filepath.Join(t.TempDir(), "config.json"))

		var report Report
		ctx := DefaultContext
		ctx.Reporter = &report

		if _, err := SyncDeep(&f, ctx); err != nil {
			t.Fatalf("SyncDeep() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpSync, Path: f.Path(), Result: SyncSkippedNoContent},
		})
	})

	t.Run("skipped_policy", func(t *testing.T) {
		var f testMapFile
		f.ComposePath(filepath.Join(t.TempDir(), "config.json"))

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

		if _, err := SyncDeep(&f, ctx); err != nil {
			t.Fatalf("SyncDeep() error = %v", err)
		}

		assertEntries(t, report.Entries(), []Entry{
			{Op: OpSync, Path: f.Path(), Result: SyncSkippedPolicy},
		})
	})
}

func TestLoadDeepReportStopsOnFirstError(t *testing.T) {
	type root struct {
		First  testMapFile       `layout:"first.json"`
		Broken testLoadErrorFile `layout:"broken.json"`
		Last   testMapFile       `layout:"last.json"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if err := os.WriteFile(layout.First.Path(), []byte(`{"name":"first"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(first) error = %v", err)
	}
	if err := os.WriteFile(layout.Last.Path(), []byte(`{"name":"last"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(last) error = %v", err)
	}

	var report Report
	ctx := DefaultContext
	ctx.Reporter = &report

	_, err := LoadDeep(&layout, ctx)
	if err == nil {
		t.Fatal("LoadDeep() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "load failed") {
		t.Fatalf("LoadDeep() error = %v, want load failed", err)
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

func TestReportRenderTreeEmpty(t *testing.T) {
	var report Report

	if got := report.RenderTree(); got != "" {
		t.Fatalf("RenderTree() = %q, want empty string", got)
	}
}

func TestReportRenderTreeNestedSiblings(t *testing.T) {
	var report Report
	report.Record(Entry{
		Op:     OpEnsure,
		Path:   filepath.Join("workspace"),
		Result: EnsureEnsured,
	})
	report.Record(Entry{
		Op:     OpLoad,
		Path:   filepath.Join("workspace", "services"),
		Result: LoadTraversed,
	})
	report.Record(Entry{
		Op:     OpEnsure,
		Path:   filepath.Join("workspace", "services", "api"),
		Result: EnsureEnsured,
	})
	report.Record(Entry{
		Op:     OpLoad,
		Path:   filepath.Join("workspace", "services", "api", "config.json"),
		Result: LoadLoaded,
	})
	report.Record(Entry{
		Op:     OpLoad,
		Path:   filepath.Join("workspace", "services", "worker", "config.json"),
		Result: LoadMissing,
	})

	want := strings.Join([]string{
		"`- workspace [ensure:ensured]",
		"   `- services [load:traversed]",
		"      |- api [ensure:ensured]",
		"      |  `- config.json [load:loaded]",
		"      `- worker",
		"         `- config.json [load:missing]",
	}, "\n")

	if got := report.RenderTree(); got != want {
		t.Fatalf("RenderTree() =\n%s\nwant:\n%s", got, want)
	}
}

func TestReportRenderTreeMultipleEntriesSamePath(t *testing.T) {
	var report Report
	path := filepath.Join("workspace", "config.json")

	report.Record(Entry{
		Op:     OpLoad,
		Path:   path,
		Result: LoadLoaded,
	})
	report.Record(Entry{
		Op:     OpSync,
		Path:   path,
		Result: SyncWritten,
	})

	want := strings.Join([]string{
		"`- workspace",
		"   `- config.json [load:loaded] [sync:written]",
	}, "\n")

	if got := report.RenderTree(); got != want {
		t.Fatalf("RenderTree() =\n%s\nwant:\n%s", got, want)
	}
}

func TestReportRenderTreeAbsolutePaths(t *testing.T) {
	var report Report
	firstPath := filepath.Join(string(filepath.Separator), "srv", "workspace", "config.json")
	secondPath := filepath.Join(string(filepath.Separator), "var", "log", "app.log")

	report.Record(Entry{
		Op:     OpScan,
		Path:   firstPath,
		Result: ScanPresent,
	})
	report.Record(Entry{
		Op:     OpScan,
		Path:   secondPath,
		Result: ScanMissing,
	})

	want := strings.Join([]string{
		"`- /",
		"   |- srv",
		"   |  `- workspace",
		"   |     `- config.json [scan:present]",
		"   `- var",
		"      `- log",
		"         `- app.log [scan:missing]",
	}, "\n")

	if got := report.RenderTree(); got != want {
		t.Fatalf("RenderTree() =\n%s\nwant:\n%s", got, want)
	}
}

func assertEntries(t *testing.T, got []Entry, want []Entry) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(entries) = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i].Op != want[i].Op || got[i].Path != want[i].Path || got[i].Result != want[i].Result {
			t.Fatalf("entry[%d] = %#v, want %#v", i, got[i], want[i])
		}
		if want[i].Err == nil && got[i].Err != nil {
			t.Fatalf("entry[%d].Err = %v, want nil", i, got[i].Err)
		}
	}
}
