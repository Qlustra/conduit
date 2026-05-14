package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatLoadMissingUpdatesState(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "missing.json"))
	f.Set(map[string]string{"stale": "value"})

	loaded, err := f.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded {
		t.Fatalf("Load() loaded = true, want false")
	}
	if f.HasContent() {
		t.Fatalf("HasContent() = true, want false")
	}
	if got := f.DiskState(); got != DiskMissing {
		t.Fatalf("DiskState() = %v, want %v", got, DiskMissing)
	}
	if got := f.MemoryState(); got != MemoryUnknown {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryUnknown)
	}
}

func TestFormatLoadPresentUpdatesState(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "present.json"))

	if err := os.WriteFile(f.Path(), []byte(`{"name":"value"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	loaded, err := f.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded {
		t.Fatalf("Load() loaded = false, want true")
	}
	if got := f.DiskState(); got != DiskPresent {
		t.Fatalf("DiskState() = %v, want %v", got, DiskPresent)
	}
	if got := f.MemoryState(); got != MemoryLoaded {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryLoaded)
	}

	value, ok := f.Get()
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if got := value["name"]; got != "value" {
		t.Fatalf("Get()[\"name\"] = %q, want %q", got, "value")
	}
}

func TestFormatLoadOrInitUsesDefaultOnlyWhenMissing(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		var f testMapFile
		f.ComposePath(filepath.Join(t.TempDir(), "missing.json"))

		if err := f.LoadOrInit(map[string]string{"name": "default"}); err != nil {
			t.Fatalf("LoadOrInit() error = %v", err)
		}

		value, ok := f.Get()
		if !ok {
			t.Fatalf("Get() ok = false, want true")
		}
		if got := value["name"]; got != "default" {
			t.Fatalf("Get()[\"name\"] = %q, want %q", got, "default")
		}
		if got := f.MemoryState(); got != MemoryDirty {
			t.Fatalf("MemoryState() = %v, want %v", got, MemoryDirty)
		}
	})

	t.Run("present", func(t *testing.T) {
		var f testMapFile
		f.ComposePath(filepath.Join(t.TempDir(), "present.json"))
		if err := os.WriteFile(f.Path(), []byte(`{"name":"disk"}`), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		if err := f.LoadOrInit(map[string]string{"name": "default"}); err != nil {
			t.Fatalf("LoadOrInit() error = %v", err)
		}

		value, ok := f.Get()
		if !ok {
			t.Fatalf("Get() ok = false, want true")
		}
		if got := value["name"]; got != "disk" {
			t.Fatalf("Get()[\"name\"] = %q, want %q", got, "disk")
		}
		if got := f.MemoryState(); got != MemoryLoaded {
			t.Fatalf("MemoryState() = %v, want %v", got, MemoryLoaded)
		}
	})
}

func TestLoadDeepLoadsFileSlotEntries(t *testing.T) {
	type root struct {
		Configs FileSlot[*testMapFile] `layout:"configs"`
	}

	var layout root
	base := t.TempDir()

	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

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

	if _, err := LoadDeep(&layout, DefaultContext); err != nil {
		t.Fatalf("LoadDeep() error = %v", err)
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
