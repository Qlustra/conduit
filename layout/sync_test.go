package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatSaveAndSyncMarkSynced(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))
	f.Set(map[string]string{"a": "b"})

	if err := f.Save(DefaultContext); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if got := f.DiskState(); got != DiskPresent {
		t.Fatalf("DiskState() after Save = %v, want %v", got, DiskPresent)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Save = %v, want %v", got, MemorySynced)
	}

	f.Set(map[string]string{"a": "c"})
	if got := f.MemoryState(); got != MemoryDirty {
		t.Fatalf("MemoryState() after Set = %v, want %v", got, MemoryDirty)
	}

	if _, err := f.Sync(DefaultContext); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != `{"a":"c"}` {
		t.Fatalf("file contents = %q, want %q", got, `{"a":"c"}`)
	}
}

func TestFormatSyncWithoutContentIsNoop(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))

	if _, err := f.Sync(DefaultContext); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if _, err := os.Stat(f.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat() error = %v, want not exist", err)
	}
}

func TestFormatSyncIfDirtySkipsLoadedState(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))

	if err := os.WriteFile(f.Path(), []byte(`{"a":"disk"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if loaded, err := f.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	} else if !loaded {
		t.Fatal("Load() loaded = false, want true")
	}

	ctx := DefaultContext
	ctx.SyncPolicy = SyncIfDirty

	if _, err := f.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := f.MemoryState(); got != MemoryLoaded {
		t.Fatalf("MemoryState() after skipped Sync = %v, want %v", got, MemoryLoaded)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != `{"a":"disk"}` {
		t.Fatalf("file contents = %q, want %q", got, `{"a":"disk"}`)
	}
}

func TestFormatSyncIfUnsyncedWritesLoadedAndSkipsSynced(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))

	if err := os.WriteFile(f.Path(), []byte(`{"a":"disk"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if loaded, err := f.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	} else if !loaded {
		t.Fatal("Load() loaded = false, want true")
	}

	ctx := DefaultContext
	ctx.SyncPolicy = SyncIfUnsynced

	if _, err := f.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}

	f.Set(map[string]string{"a": "dirty"})
	if _, err := f.Sync(ctx); err != nil {
		t.Fatalf("Sync() after Set() error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after dirty Sync = %v, want %v", got, MemorySynced)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != `{"a":"dirty"}` {
		t.Fatalf("file contents after dirty Sync = %q, want %q", got, `{"a":"dirty"}`)
	}

	if _, err := f.Sync(ctx); err != nil {
		t.Fatalf("Sync() for synced state error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after skipped synced Sync = %v, want %v", got, MemorySynced)
	}

	data, err = os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != `{"a":"dirty"}` {
		t.Fatalf("file contents after skipped synced Sync = %q, want %q", got, `{"a":"dirty"}`)
	}
}

func TestFormatSyncDefaultsToRewriteWhenPolicyUnset(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))
	f.Set(map[string]string{"a": "b"})

	ctx := Context{
		DirMode:  0o755,
		FileMode: 0o644,
	}

	if _, err := f.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}
}

func TestFormatSyncIfMissingSupportsInitializationFlow(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))

	if err := f.LoadOrInit(map[string]string{"name": "default"}); err != nil {
		t.Fatalf("LoadOrInit() error = %v", err)
	}
	if got := f.DiskState(); got != DiskMissing {
		t.Fatalf("DiskState() after missing LoadOrInit = %v, want %v", got, DiskMissing)
	}
	if got := f.MemoryState(); got != MemoryDirty {
		t.Fatalf("MemoryState() after missing LoadOrInit = %v, want %v", got, MemoryDirty)
	}

	ctx := DefaultContext
	ctx.SyncPolicy = SyncIfMissing

	result, err := f.Sync(ctx)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result != SyncWritten {
		t.Fatalf("Sync() result = %v, want %v", result, SyncWritten)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}
	if got := f.DiskState(); got != DiskPresent {
		t.Fatalf("DiskState() after Sync = %v, want %v", got, DiskPresent)
	}

	result, err = f.Sync(ctx)
	if err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}
	if result != SyncSkippedPolicy {
		t.Fatalf("second Sync() result = %v, want %v", result, SyncSkippedPolicy)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != `{"name":"default"}` {
		t.Fatalf("file contents = %q, want %q", got, `{"name":"default"}`)
	}
}

func TestFormatSyncDiskFilterDefaultsMemoryMaskToRewrite(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))

	if err := os.WriteFile(f.Path(), []byte(`{"a":"disk"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if loaded, err := f.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	} else if !loaded {
		t.Fatal("Load() loaded = false, want true")
	}

	ctx := DefaultContext
	ctx.SyncPolicy = SyncOnDiskPresent

	result, err := f.Sync(ctx)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result != SyncWritten {
		t.Fatalf("Sync() result = %v, want %v", result, SyncWritten)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}
}

func TestFormatSyncIfMissingSkipsWhenDiskPresent(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))

	if err := os.WriteFile(f.Path(), []byte(`{"a":"disk"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if loaded, err := f.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	} else if !loaded {
		t.Fatal("Load() loaded = false, want true")
	}

	ctx := DefaultContext
	ctx.SyncPolicy = SyncIfMissing

	result, err := f.Sync(ctx)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result != SyncSkippedPolicy {
		t.Fatalf("Sync() result = %v, want %v", result, SyncSkippedPolicy)
	}
	if got := f.MemoryState(); got != MemoryLoaded {
		t.Fatalf("MemoryState() after skipped Sync = %v, want %v", got, MemoryLoaded)
	}
}

func TestFormatSyncIfMissingSkipsWhenDiskStateUnknown(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))
	f.Set(map[string]string{"a": "planned"})

	ctx := DefaultContext
	ctx.SyncPolicy = SyncIfMissing

	result, err := f.Sync(ctx)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result != SyncSkippedPolicy {
		t.Fatalf("Sync() result = %v, want %v", result, SyncSkippedPolicy)
	}
	if got := f.DiskState(); got != DiskUnknown {
		t.Fatalf("DiskState() after skipped Sync = %v, want %v", got, DiskUnknown)
	}
	if got := f.MemoryState(); got != MemoryDirty {
		t.Fatalf("MemoryState() after skipped Sync = %v, want %v", got, MemoryDirty)
	}
}
