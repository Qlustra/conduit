package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSlotAtCachesComposedItems(t *testing.T) {
	var slot FileSlot[*testMapFile]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	first, err := slot.At("api.json")
	if err != nil {
		t.Fatalf("At() error = %v", err)
	}
	second, err := slot.At("api.json")
	if err != nil {
		t.Fatalf("At() error = %v", err)
	}

	if first != second {
		t.Fatalf("At() returned different cached items for the same key")
	}
	if got := first.Path(); got != filepath.Join(slot.Path(), "api.json") {
		t.Fatalf("Path() = %q", got)
	}
	if keys := slot.Keys(); len(keys) != 1 || keys[0] != "api.json" {
		t.Fatalf("Keys() = %v, want [api.json]", keys)
	}
}

func TestFileSlotAddEnsuresDeclaredFile(t *testing.T) {
	var slot FileSlot[*testMapFile]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	added, err := slot.Add("api.json", DefaultContext)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if _, err := os.Stat(slot.Path()); err != nil {
		t.Fatalf("os.Stat(slot root) error = %v", err)
	}
	if _, err := os.Stat(added.Path()); err != nil {
		t.Fatalf("os.Stat(item file) error = %v", err)
	}
	if cached, ok := slot.Get("api.json"); !ok || cached != added {
		t.Fatalf("Get(\"api.json\") did not return the cached item")
	}
}

func TestFileSlotAddCachedItemRecreatesMissingFile(t *testing.T) {
	var slot FileSlot[*testMapFile]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	added, err := slot.Add("api.json", DefaultContext)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := os.Remove(added.Path()); err != nil {
		t.Fatalf("os.Remove(item file) error = %v", err)
	}

	again, err := slot.Add("api.json", DefaultContext)
	if err != nil {
		t.Fatalf("second Add() error = %v", err)
	}

	if again != added {
		t.Fatalf("second Add() returned a different cached item")
	}
	if _, err := os.Stat(added.Path()); err != nil {
		t.Fatalf("os.Stat(item file) error = %v", err)
	}
}

func TestFileSlotDeleteRemovesDiskAndCache(t *testing.T) {
	var slot FileSlot[*testMapFile]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	added, err := slot.Add("api.json", DefaultContext)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := slot.Delete("api.json", DefaultContext); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := os.Stat(added.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(item file) error = %v, want not-exist", err)
	}
	if _, ok := slot.Get("api.json"); ok {
		t.Fatalf("Get(\"api.json\") ok = true after Delete(), want false")
	}
}

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

func TestFileSlotRequireNeedsExistingFile(t *testing.T) {
	var slot FileSlot[*testMapFile]
	slot.ComposePath(filepath.Join(t.TempDir(), "configs"))

	if _, err := slot.Require("missing.json"); err == nil {
		t.Fatal("Require() error = nil, want error for missing file")
	}

	if err := os.MkdirAll(slot.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(slot root) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(slot.Path(), "api.json"), []byte(`{"name":"api"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	item, err := slot.Require("api.json")
	if err != nil {
		t.Fatalf("Require() error = %v", err)
	}
	if got := item.Path(); got != filepath.Join(slot.Path(), "api.json") {
		t.Fatalf("Path() = %q", got)
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
		if _, err := slot.At(name); err == nil {
			t.Fatalf("At(%q) error = nil, want non-nil", name)
		}
		if _, err := slot.Add(name, DefaultContext); err == nil {
			t.Fatalf("Add(%q) error = nil, want non-nil", name)
		}
		if _, err := slot.Require(name); err == nil {
			t.Fatalf("Require(%q) error = nil, want non-nil", name)
		}
		if err := slot.Delete(name, DefaultContext); err == nil {
			t.Fatalf("Delete(%q) error = nil, want non-nil", name)
		}
	}
}
