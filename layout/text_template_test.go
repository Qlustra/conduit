package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTextTemplateComposePathResetsStateAndContext(t *testing.T) {
	var f TextTemplate[string]

	f.Set("stale")
	f.SetContext("ctx")
	f.disk = DiskPresent

	f.ComposePath("tmp/example.txt")

	if got := f.Path(); got != filepath.Clean("tmp/example.txt") {
		t.Fatalf("Path() = %q, want %q", got, filepath.Clean("tmp/example.txt"))
	}
	if f.HasContent() {
		t.Fatalf("HasContent() = true, want false")
	}
	if f.HasContext() {
		t.Fatalf("HasContext() = true, want false")
	}
	if got := f.DiskState(); got != DiskUnknown {
		t.Fatalf("DiskState() = %v, want %v", got, DiskUnknown)
	}
	if got := f.MemoryState(); got != MemoryUnknown {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryUnknown)
	}
}

func TestTextTemplateContextHelpers(t *testing.T) {
	var f TextTemplate[int]

	if f.HasContext() {
		t.Fatalf("HasContext() = true, want false")
	}
	if _, ok := f.GetContext(); ok {
		t.Fatalf("GetContext() ok = true, want false")
	}

	f.SetContext(42)

	value, ok := f.GetContext()
	if !ok {
		t.Fatalf("GetContext() ok = false, want true")
	}
	if value != 42 {
		t.Fatalf("GetContext() = %d, want %d", value, 42)
	}
	if got := f.MustContext(); got != 42 {
		t.Fatalf("MustContext() = %d, want %d", got, 42)
	}

	f.ClearContext()

	if f.HasContext() {
		t.Fatalf("HasContext() = true, want false after ClearContext()")
	}
}

func TestTextTemplateMustContextPanicsWithoutContext(t *testing.T) {
	var f TextTemplate[string]

	defer func() {
		if recover() == nil {
			t.Fatal("MustContext() did not panic")
		}
	}()

	_ = f.MustContext()
}

func TestTextTemplateLoadAndSyncMirrorFormatBehavior(t *testing.T) {
	var f TextTemplate[string]
	f.ComposePath(filepath.Join(t.TempDir(), "state.txt"))

	if err := os.WriteFile(f.Path(), []byte("disk"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	loaded, err := f.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded {
		t.Fatalf("Load() loaded = false, want true")
	}
	if got := f.MustGet(); got != "disk" {
		t.Fatalf("MustGet() = %q, want %q", got, "disk")
	}
	if got := f.MemoryState(); got != MemoryLoaded {
		t.Fatalf("MemoryState() after Load = %v, want %v", got, MemoryLoaded)
	}

	f.Set("memory")
	if _, err := f.Sync(DefaultContext); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := f.MemoryState(); got != MemorySynced {
		t.Fatalf("MemoryState() after Sync = %v, want %v", got, MemorySynced)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != "memory" {
		t.Fatalf("file contents = %q, want %q", got, "memory")
	}
}
