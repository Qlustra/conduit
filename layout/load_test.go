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
