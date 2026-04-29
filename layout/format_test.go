package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatComposePathResetsState(t *testing.T) {
	var f testMapFile

	f.Set(map[string]string{"a": "b"})
	f.disk = DiskPresent

	f.ComposePath("tmp/example.json")

	if got := f.Path(); got != filepath.Clean("tmp/example.json") {
		t.Fatalf("Path() = %q", got)
	}
	if f.HasContent() {
		t.Fatalf("HasContent() = true, want false")
	}
	if got := f.DiskState(); got != DiskUnknown {
		t.Fatalf("DiskState() = %v, want %v", got, DiskUnknown)
	}
	if got := f.MemoryState(); got != MemoryUnknown {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryUnknown)
	}
}

func TestFormatSetClearAndGetTrackMemoryState(t *testing.T) {
	var f testMapFile

	f.Set(map[string]string{"a": "b"})

	value, ok := f.Get()
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if got := value["a"]; got != "b" {
		t.Fatalf("Get()[\"a\"] = %q, want %q", got, "b")
	}
	if !f.IsDirty() {
		t.Fatalf("IsDirty() = false, want true")
	}
	if !f.HasBeenLoaded() {
		t.Fatalf("HasBeenLoaded() = false, want true")
	}

	f.Clear()

	if _, ok := f.Get(); ok {
		t.Fatalf("Get() ok = true, want false after Clear()")
	}
	if f.HasContent() {
		t.Fatalf("HasContent() = true, want false after Clear()")
	}
	if got := f.MemoryState(); got != MemoryUnknown {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryUnknown)
	}
}

func TestFormatMustGetPanicsWithoutContent(t *testing.T) {
	var f testMapFile

	defer func() {
		if recover() == nil {
			t.Fatal("MustGet() did not panic")
		}
	}()

	_ = f.MustGet()
}

func TestFormatDeleteClearsContentAndMarksMissing(t *testing.T) {
	var f testMapFile
	f.ComposePath(filepath.Join(t.TempDir(), "state.json"))
	f.Set(map[string]string{"a": "b"})

	if err := f.Save(DefaultContext); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := f.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := os.Stat(f.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat() error = %v, want not exist", err)
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
