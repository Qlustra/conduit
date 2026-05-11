package layout

import (
	"os"
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

func TestDiscoverDeepDiscoversSlotEntriesWithoutLoadingContent(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}

	type root struct {
		Services Slot[*item] `layout:"services"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	cached, err := layout.Services.At("api")
	if err != nil {
		t.Fatalf("Services.At() error = %v", err)
	}
	cached.Config.Set(map[string]string{"cached": "value"})

	if err := os.MkdirAll(filepath.Dir(cached.Config.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(cached.Config.Path(), []byte(`{"name":"api"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	diskOnlyPath := filepath.Join(base, "services", "worker", "config.json")
	if err := os.MkdirAll(filepath.Dir(diskOnlyPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(diskOnlyPath, []byte(`{"name":"worker"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if _, err := DiscoverDeep(&layout, DefaultContext); err != nil {
		t.Fatalf("DiscoverDeep() error = %v", err)
	}

	if got := cached.Config.DiskState(); got != DiskPresent {
		t.Fatalf("cached.Config.DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := cached.Config.MemoryState(); got != MemoryDirty {
		t.Fatalf("cached.Config.MemoryState() = %v, want %v", got, MemoryDirty)
	}
	value, ok := cached.Config.Get()
	if !ok {
		t.Fatalf("cached.Config.Get() ok = false, want true")
	}
	if got := value["cached"]; got != "value" {
		t.Fatalf("cached.Config.Get()[\"cached\"] = %q, want %q", got, "value")
	}

	if keys := layout.Services.Keys(); len(keys) != 2 || keys[0] != "api" || keys[1] != "worker" {
		t.Fatalf("Services.Keys() = %v, want [api worker]", keys)
	}

	discovered, ok := layout.Services.Get("worker")
	if !ok {
		t.Fatalf("Services.Get(\"worker\") = false, want true")
	}
	if discovered.Config.HasContent() {
		t.Fatalf("discovered.Config.HasContent() = true, want false")
	}
	if got := discovered.Config.DiskState(); got != DiskPresent {
		t.Fatalf("discovered.Config.DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := discovered.Config.MemoryState(); got != MemoryUnknown {
		t.Fatalf("discovered.Config.MemoryState() = %v, want %v", got, MemoryUnknown)
	}
}
