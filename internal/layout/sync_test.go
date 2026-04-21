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
