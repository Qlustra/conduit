package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverLayoutDiscoversSlotEntriesWithoutLoadingContent(t *testing.T) {
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

	if _, err := manager.Discover(DefaultContext); err != nil {
		t.Fatalf("Discover() error = %v", err)
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

func TestDiscoverLayoutDiscoversFileSlotEntriesWithoutLoadingContent(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(base, "configs", "nested"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.Symlink(cached.Path(), filepath.Join(base, "configs", "linked.json")); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if _, err := manager.Discover(DefaultContext); err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if got := cached.DiskState(); got != DiskPresent {
		t.Fatalf("cached.DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := cached.MemoryState(); got != MemoryDirty {
		t.Fatalf("cached.MemoryState() = %v, want %v", got, MemoryDirty)
	}
	value, ok := cached.Get()
	if !ok {
		t.Fatalf("cached.Get() ok = false, want true")
	}
	if got := value["cached"]; got != "value" {
		t.Fatalf("cached.Get()[\"cached\"] = %q, want %q", got, "value")
	}

	if keys := layout.Configs.Keys(); len(keys) != 2 || keys[0] != "api.json" || keys[1] != "worker.json" {
		t.Fatalf("Configs.Keys() = %v, want [api.json worker.json]", keys)
	}

	discovered, ok := layout.Configs.Get("worker.json")
	if !ok {
		t.Fatalf("Configs.Get(\"worker.json\") = false, want true")
	}
	if discovered.HasContent() {
		t.Fatalf("discovered.HasContent() = true, want false")
	}
	if got := discovered.DiskState(); got != DiskPresent {
		t.Fatalf("discovered.DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := discovered.MemoryState(); got != MemoryUnknown {
		t.Fatalf("discovered.MemoryState() = %v, want %v", got, MemoryUnknown)
	}
	if _, ok := layout.Configs.Get("nested"); ok {
		t.Fatalf("Configs.Get(\"nested\") = true, want false")
	}
	if _, ok := layout.Configs.Get("linked.json"); ok {
		t.Fatalf("Configs.Get(\"linked.json\") = true, want false")
	}
}
