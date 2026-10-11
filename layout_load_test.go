package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLayoutLoadsFileSlotEntries(t *testing.T) {
	type root struct {
		Configs FileSlot[*testMapFile] `layout:"documents" path:"configs"`
	}

	base := t.TempDir()

	manager, layout := composeTree[root](t, base)

	if err := os.MkdirAll(filepath.Join(base, "configs"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "configs", "api.json"), []byte(`{"name":"api"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "configs", "worker.json"), []byte(`{"name":"worker"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "configs", "nested"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.Symlink(filepath.Join(base, "configs", "api.json"), filepath.Join(base, "configs", "linked.json")); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if _, err := manager.Load(DefaultContext); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if keys := layout.Configs.Keys(); len(keys) != 2 || keys[0] != "api.json" || keys[1] != "worker.json" {
		t.Fatalf("Configs.Keys() = %v, want [api.json worker.json]", keys)
	}

	api, ok := layout.Configs.Get("api.json")
	if !ok {
		t.Fatalf("Configs.Get(\"api.json\") = false, want true")
	}
	if !api.HasContent() {
		t.Fatal("api.HasContent() = false, want true")
	}
	value, ok := api.Get()
	if !ok {
		t.Fatalf("api.Get() ok = false, want true")
	}
	if got := value["name"]; got != "api" {
		t.Fatalf("api.Get()[\"name\"] = %q, want %q", got, "api")
	}
	if _, ok := layout.Configs.Get("nested"); ok {
		t.Fatalf("Configs.Get(\"nested\") = true, want false")
	}
	if _, ok := layout.Configs.Get("linked.json"); ok {
		t.Fatalf("Configs.Get(\"linked.json\") = true, want false")
	}
}
