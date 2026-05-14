package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDeepPolicyCanLimitMaterializedNodeKinds(t *testing.T) {
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
	ctx.EnsurePolicy = EnsureDirs
	ctx.Reporter = &report

	if _, err := EnsureDeep(&layout, ctx); err != nil {
		t.Fatalf("EnsureDeep() error = %v", err)
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

func TestEnsurePolicyHelpers(t *testing.T) {
	policy := EnsureNone.
		Allow(EnsureDirs | EnsureSyncables).
		Deny(EnsureDirs)

	if policy != EnsureSyncables {
		t.Fatalf("policy = %v, want EnsureSyncables", policy)
	}
	if policy.Has(EnsureSyncables) != true {
		t.Fatal("policy.Has(EnsureSyncables) = false, want true")
	}
	if policy.Has(EnsureDirs) != false {
		t.Fatal("policy.Has(EnsureDirs) = true, want false")
	}

	policy = EnsureScaffold.Deny(EnsureDirs | EnsureFiles | EnsureExecs)
	if policy != EnsureNone {
		t.Fatalf("policy after Deny(all scaffold bits) = %v, want EnsureNone", policy)
	}

	policy = EnsureNone.Allow(EnsureDirs | EnsurePolicy(1<<7))
	if policy != EnsureDirs {
		t.Fatalf("policy with unknown bits allowed = %v, want EnsureDirs", policy)
	}
}

func TestEnsureDeepPolicyCanEnsureSyncablesWithoutRawFiles(t *testing.T) {
	type root struct {
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
	ctx.EnsurePolicy = EnsureSyncables
	ctx.Reporter = &report

	if _, err := EnsureDeep(&layout, ctx); err != nil {
		t.Fatalf("EnsureDeep() error = %v", err)
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

func TestSlotSyncDeepEnsurePhaseRespectsEnsurePolicy(t *testing.T) {
	type item struct {
		Root   Dir         `layout:"."`
		Config testMapFile `layout:"config.json"`
	}

	type root struct {
		Root  Dir         `layout:"."`
		Items Slot[*item] `layout:"items"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	api, err := layout.Items.At("api")
	if err != nil {
		t.Fatalf("Items.At() error = %v", err)
	}
	api.Config.Set(map[string]string{"name": "api"})

	var report Report
	ctx := DefaultContext
	ctx.EnsurePolicy = EnsureDirs
	ctx.Reporter = &report

	if _, err := SyncDeep(&layout, ctx); err != nil {
		t.Fatalf("SyncDeep() error = %v", err)
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

func TestFileSlotSyncDeepEnsurePhaseRespectsEnsurePolicy(t *testing.T) {
	type root struct {
		Configs FileSlot[*testMapFile] `layout:"configs"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	cfg, err := layout.Configs.At("api.json")
	if err != nil {
		t.Fatalf("Configs.At() error = %v", err)
	}
	cfg.Set(map[string]string{"name": "api"})

	var report Report
	ctx := DefaultContext
	ctx.EnsurePolicy = EnsureDirs
	ctx.Reporter = &report

	if _, err := SyncDeep(&layout, ctx); err != nil {
		t.Fatalf("SyncDeep() error = %v", err)
	}

	if _, err := os.Stat(cfg.Path()); err != nil {
		t.Fatalf("os.Stat(config) error = %v", err)
	}

	assertHasEntry(t, report.Entries(), Entry{Op: OpEnsure, Path: cfg.Path(), Result: EnsureSkippedPolicy})
	assertHasEntry(t, report.Entries(), Entry{Op: OpSync, Path: cfg.Path(), Result: SyncWritten})
}

func assertHasEntry(t *testing.T, entries []Entry, want Entry) {
	t.Helper()

	for _, got := range entries {
		if got.Op == want.Op && got.Path == want.Path && got.Result == want.Result {
			if want.Err == nil || got.Err != nil {
				return
			}
		}
	}

	t.Fatalf("missing entry %#v in %#v", want, entries)
}
