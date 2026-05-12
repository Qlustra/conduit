package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkComposeAndTypedTargets(t *testing.T) {
	type project struct {
		Root   Dir      `layout:"."`
		Readme FileLink `layout:"README.link"`
		Assets DirLink  `layout:"assets.link"`
	}

	var layout project
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	layout.Readme.SetTarget("docs/README.md")
	layout.Assets.SetTarget("../shared/assets")

	if got := layout.Readme.Path(); got != filepath.Join(base, "README.link") {
		t.Fatalf("Readme.Path() = %q", got)
	}
	if got, ok := layout.Readme.DeclaredPath(); !ok || got != "README.link" {
		t.Fatalf("Readme.DeclaredPath() = (%q, %t), want (%q, true)", got, ok, "README.link")
	}
	if got, ok := layout.Assets.ComposedRelativePath(); !ok || got != "assets.link" {
		t.Fatalf("Assets.ComposedRelativePath() = (%q, %t), want (%q, true)", got, ok, "assets.link")
	}

	readmeTarget, ok := layout.Readme.TargetFile()
	if !ok {
		t.Fatal("Readme.TargetFile() ok = false, want true")
	}
	if got := readmeTarget.Path(); got != filepath.Join(base, "docs", "README.md") {
		t.Fatalf("Readme.TargetFile().Path() = %q, want %q", got, filepath.Join(base, "docs", "README.md"))
	}

	assetsTarget, ok := layout.Assets.TargetDir()
	if !ok {
		t.Fatal("Assets.TargetDir() ok = false, want true")
	}
	if got := assetsTarget.Path(); got != filepath.Join(filepath.Dir(base), "shared", "assets") {
		t.Fatalf("Assets.TargetDir().Path() = %q, want %q", got, filepath.Join(filepath.Dir(base), "shared", "assets"))
	}
}

func TestLinkLoadMissingUpdatesState(t *testing.T) {
	var link Link
	link.ComposePath(filepath.Join(t.TempDir(), "missing.link"))
	link.SetTarget("stale.txt")

	loaded, err := link.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded {
		t.Fatalf("Load() loaded = true, want false")
	}
	if link.HasTarget() {
		t.Fatal("HasTarget() = true, want false")
	}
	if got := link.DiskState(); got != DiskMissing {
		t.Fatalf("DiskState() = %v, want %v", got, DiskMissing)
	}
	if got := link.MemoryState(); got != MemoryUnknown {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryUnknown)
	}
}

func TestLinkLoadDanglingSymlink(t *testing.T) {
	var link Link
	link.ComposePath(filepath.Join(t.TempDir(), "config.link"))

	if err := os.MkdirAll(filepath.Dir(link.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.Symlink("missing/config.yaml", link.Path()); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	loaded, err := link.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded {
		t.Fatal("Load() loaded = false, want true")
	}
	if !link.Exists() {
		t.Fatal("Exists() = false, want true")
	}
	if got := link.MustTarget(); got != "missing/config.yaml" {
		t.Fatalf("MustTarget() = %q, want %q", got, "missing/config.yaml")
	}
	exists, err := link.TargetExists()
	if err != nil {
		t.Fatalf("TargetExists() error = %v", err)
	}
	if exists {
		t.Fatal("TargetExists() = true, want false")
	}
	dangling, err := link.IsDangling()
	if err != nil {
		t.Fatalf("IsDangling() error = %v", err)
	}
	if !dangling {
		t.Fatal("IsDangling() = false, want true")
	}
	if got := link.DiskState(); got != DiskPresent {
		t.Fatalf("DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := link.MemoryState(); got != MemoryLoaded {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryLoaded)
	}
}

func TestLinkScanRejectsNonSymlinkEntry(t *testing.T) {
	var link Link
	link.ComposePath(filepath.Join(t.TempDir(), "config.link"))

	if err := os.WriteFile(link.Path(), []byte("not a symlink"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	state, err := link.Scan()
	if err == nil {
		t.Fatal("Scan() error = nil, want non-nil")
	}
	if state != DiskUnknown {
		t.Fatalf("Scan() state = %v, want %v", state, DiskUnknown)
	}
	if got := link.DiskState(); got != DiskUnknown {
		t.Fatalf("DiskState() = %v, want %v", got, DiskUnknown)
	}
}

func TestLinkSyncCreatesSymlinkAndMarksSynced(t *testing.T) {
	var link Link
	link.ComposePath(filepath.Join(t.TempDir(), "config.link"))
	link.SetTarget("configs/app.yaml")

	result, err := link.Sync(DefaultContext)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result != SyncWritten {
		t.Fatalf("Sync() result = %v, want %v", result, SyncWritten)
	}

	target, err := os.Readlink(link.Path())
	if err != nil {
		t.Fatalf("os.Readlink() error = %v", err)
	}
	if target != "configs/app.yaml" {
		t.Fatalf("os.Readlink() = %q, want %q", target, "configs/app.yaml")
	}
	if got := link.DiskState(); got != DiskPresent {
		t.Fatalf("DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := link.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() = %v, want %v", got, MemorySynced)
	}
}

func TestLinkSyncWithoutTargetIsNoop(t *testing.T) {
	var link Link
	link.ComposePath(filepath.Join(t.TempDir(), "config.link"))

	result, err := link.Sync(DefaultContext)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result != SyncSkippedNoContent {
		t.Fatalf("Sync() result = %v, want %v", result, SyncSkippedNoContent)
	}
	if _, err := os.Lstat(link.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Lstat() error = %v, want not-exist", err)
	}
}

func TestLinkSyncRejectsNonSymlinkEntry(t *testing.T) {
	var link Link
	link.ComposePath(filepath.Join(t.TempDir(), "config.link"))
	link.SetTarget("configs/app.yaml")

	if err := os.WriteFile(link.Path(), []byte("not a symlink"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	result, err := link.Sync(DefaultContext)
	if err == nil {
		t.Fatal("Sync() error = nil, want non-nil")
	}
	if result != SyncFailed {
		t.Fatalf("Sync() result = %v, want %v", result, SyncFailed)
	}
}

func TestLinkDeleteRemovesSymlinkAndKeepsTargetKindChecksLocal(t *testing.T) {
	var link Link
	link.ComposePath(filepath.Join(t.TempDir(), "config.link"))
	link.SetTarget("configs/app.yaml")

	if _, err := link.Sync(DefaultContext); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if err := link.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Lstat(link.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Lstat() error = %v, want not-exist", err)
	}
	if link.HasTarget() {
		t.Fatal("HasTarget() = true, want false")
	}
	if got := link.DiskState(); got != DiskMissing {
		t.Fatalf("DiskState() = %v, want %v", got, DiskMissing)
	}
}

func TestEnsureDeepDoesNotMaterializeLinks(t *testing.T) {
	type project struct {
		Root   Dir  `layout:"."`
		Config File `layout:"config.yaml"`
		Link   Link `layout:"config.link"`
	}

	var layout project
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if _, err := EnsureDeep(&layout, DefaultContext); err != nil {
		t.Fatalf("EnsureDeep() error = %v", err)
	}

	if _, err := os.Stat(layout.Config.Path()); err != nil {
		t.Fatalf("os.Stat(config) error = %v", err)
	}
	if _, err := os.Lstat(layout.Link.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Lstat(link) error = %v, want not-exist", err)
	}
}
