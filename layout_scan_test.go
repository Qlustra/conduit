package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestScanLayoutScansCachedFieldsAndDoesNotDiscoverSlotEntries(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"content" path:"config.json"`
	}

	type root struct {
		Services Slot[*item] `layout:"directories" path:"services"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	cached, err := layout.Services.ComposeMember("api")
	if err != nil {
		t.Fatalf("Services.ComposeMember() error = %v", err)
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

	if _, err := manager.Scan(DefaultContext); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if got := cached.Config.DiskState(); got != DiskPresent {
		t.Fatalf("cached.Config.DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := cached.Config.MemoryState(); got != MemoryDirty {
		t.Fatalf("cached.Config.MemoryState() = %v, want %v", got, MemoryDirty)
	}

	if keys := layout.Services.Keys(); len(keys) != 1 || keys[0] != "api" {
		t.Fatalf("Services.Keys() = %v, want [api]", keys)
	}
	if _, ok := layout.Services.Get("worker"); ok {
		t.Fatalf("Services.Get(\"worker\") = true, want false")
	}
}

func TestScanLayoutScansCachedFileSlotFieldsAndDoesNotDiscoverEntries(t *testing.T) {
	type root struct {
		Configs FileSlot[*testMapFile] `layout:"documents" path:"configs"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	cached, err := layout.Configs.ComposeMember("api.json")
	if err != nil {
		t.Fatalf("Configs.ComposeMember() error = %v", err)
	}
	cached.Set(map[string]string{"cached": "value"})

	if err := os.MkdirAll(filepath.Dir(cached.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(cached.Path(), []byte(`{"name":"api"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	diskOnlyPath := filepath.Join(base, "configs", "worker.json")
	if err := os.MkdirAll(filepath.Dir(diskOnlyPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(diskOnlyPath, []byte(`{"name":"worker"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if _, err := manager.Scan(DefaultContext); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if got := cached.DiskState(); got != DiskPresent {
		t.Fatalf("cached.DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := cached.MemoryState(); got != MemoryDirty {
		t.Fatalf("cached.MemoryState() = %v, want %v", got, MemoryDirty)
	}

	if keys := layout.Configs.Keys(); len(keys) != 1 || keys[0] != "api.json" {
		t.Fatalf("Configs.Keys() = %v, want [api.json]", keys)
	}
	if _, ok := layout.Configs.Get("worker.json"); ok {
		t.Fatalf("Configs.Get(\"worker.json\") = true, want false")
	}
}
