package layout

import (
	"path/filepath"
	"testing"
)

func TestFormatDiscoverUpdatesDiskStateAndPreservesMemory(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "missing.json"))
	f.Set(map[string]string{"stale": "value"})

	state, err := f.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if state != DiskMissing {
		t.Fatalf("Discover() state = %v, want %v", state, DiskMissing)
	}
	if got := f.DiskState(); got != DiskMissing {
		t.Fatalf("DiskState() = %v, want %v", got, DiskMissing)
	}
	if got := f.MemoryState(); got != MemoryDirty {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryDirty)
	}

	value, ok := f.Get()
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if got := value["stale"]; got != "value" {
		t.Fatalf("Get()[\"stale\"] = %q, want %q", got, "value")
	}
}
