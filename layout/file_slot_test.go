package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSlotEntriesReturnSortedSnapshot(t *testing.T) {
	var slot FileSlot[int]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	slot.Put("worker.json", 2)
	slot.Put("api.json", 1)

	entries := slot.Entries()
	if len(entries) != 2 {
		t.Fatalf("len(Entries()) = %d, want 2", len(entries))
	}
	if entries[0].Name != "api.json" || entries[0].Item != 1 {
		t.Fatalf("Entries()[0] = {%q, %d}, want {%q, %d}", entries[0].Name, entries[0].Item, "api.json", 1)
	}
	if entries[1].Name != "worker.json" || entries[1].Item != 2 {
		t.Fatalf("Entries()[1] = {%q, %d}, want {%q, %d}", entries[1].Name, entries[1].Item, "worker.json", 2)
	}

	slot.Remove("api.json")
	if len(entries) != 2 {
		t.Fatalf("len(Entries()) after Remove() = %d, want snapshot to stay 2", len(entries))
	}
}

func TestFileSlotAllIteratesSortedCachedItems(t *testing.T) {
	var slot FileSlot[int]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	slot.Put("worker.json", 2)
	slot.Put("api.json", 1)

	var gotNames []string
	var gotItems []int
	for name, item := range slot.All() {
		gotNames = append(gotNames, name)
		gotItems = append(gotItems, item)
	}

	if len(gotNames) != 2 {
		t.Fatalf("len(names from All()) = %d, want 2", len(gotNames))
	}
	if gotNames[0] != "api.json" || gotItems[0] != 1 {
		t.Fatalf("first All() item = {%q, %d}, want {%q, %d}", gotNames[0], gotItems[0], "api.json", 1)
	}
	if gotNames[1] != "worker.json" || gotItems[1] != 2 {
		t.Fatalf("second All() item = {%q, %d}, want {%q, %d}", gotNames[1], gotItems[1], "worker.json", 2)
	}
}

func TestFileSlotLenTracksCachedItems(t *testing.T) {
	var slot FileSlot[int]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	if got := slot.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}

	slot.Put("api.json", 1)
	slot.Put("worker.json", 2)
	if got := slot.Len(); got != 2 {
		t.Fatalf("Len() after Put() = %d, want 2", got)
	}

	slot.Remove("api.json")
	if got := slot.Len(); got != 1 {
		t.Fatalf("Len() after Remove() = %d, want 1", got)
	}

	slot.Clear()
	if got := slot.Len(); got != 0 {
		t.Fatalf("Len() after Clear() = %d, want 0", got)
	}
}

func TestFileSlotHasAndRequireRejectSymlinkChildren(t *testing.T) {
	var slot FileSlot[*testMapFile]
	base := t.TempDir()
	slot.ComposePath(filepath.Join(base, "configs"))

	if err := os.MkdirAll(slot.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(slot root) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "target.json"), []byte(`{"name":"target"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(filepath.Join(base, "target.json"), filepath.Join(slot.Path(), "linked.json")); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if slot.Has("linked.json") {
		t.Fatal("Has(linked.json) = true, want false for symlink")
	}
	if _, err := slot.Require("linked.json"); err == nil {
		t.Fatal("Require(linked.json) error = nil, want non-nil")
	}
}

func TestFileSlotRejectsNamesOutsideDirectChildren(t *testing.T) {
	var slot FileSlot[*testMapFile]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	invalid := []string{"", ".", "..", "nested/api.json", "../api.json", `/tmp/api.json`}
	for _, name := range invalid {
		if slot.Has(name) {
			t.Fatalf("Has(%q) = true, want false", name)
		}
		if _, err := slot.ComposeMember(name); err == nil {
			t.Fatalf("ComposeMember(%q) error = nil, want non-nil", name)
		}
		if _, err := slot.Require(name); err == nil {
			t.Fatalf("Require(%q) error = nil, want non-nil", name)
		}
		if err := slot.Delete(name, DefaultContext); err == nil {
			t.Fatalf("Delete(%q) error = nil, want non-nil", name)
		}
	}
}
