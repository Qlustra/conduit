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

	if err := f.Sync(DefaultContext); err != nil {
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

	if err := f.Sync(DefaultContext); err != nil {
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

	if err := f.Sync(ctx); err != nil {
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

	if err := f.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}

	f.Set(map[string]string{"a": "dirty"})
	if err := f.Sync(ctx); err != nil {
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

	if err := f.Sync(ctx); err != nil {
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

	if err := f.Sync(ctx); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}
}
