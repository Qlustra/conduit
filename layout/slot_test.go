package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlotAtCachesComposedItems(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}

	var slot Slot[*item]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	first, err := slot.At("api")
	if err != nil {
		t.Fatalf("At() error = %v", err)
	}
	second, err := slot.At("api")
	if err != nil {
		t.Fatalf("At() error = %v", err)
	}

	if first != second {
		t.Fatalf("At() returned different cached items for the same key")
	}
	if got := first.Config.Path(); got != filepath.Join(slot.Path(), "api", "config.json") {
		t.Fatalf("Config.Path() = %q", got)
	}
	if keys := slot.Keys(); len(keys) != 1 || keys[0] != "api" {
		t.Fatalf("Keys() = %v, want [api]", keys)
	}
}

func TestSlotAddEnsuresItemRootAndDeclaredFiles(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}

	var slot Slot[*item]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	added, err := slot.Add("api", DefaultContext)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(slot.Path(), "api")); err != nil {
		t.Fatalf("os.Stat(item root) error = %v", err)
	}
	if _, err := os.Stat(added.Config.Path()); err != nil {
		t.Fatalf("os.Stat(config file) error = %v", err)
	}
	if cached, ok := slot.Get("api"); !ok || cached != added {
		t.Fatalf("Get(\"api\") did not return the cached item")
	}
}

func TestSlotAddCachedItemOnlyRecreatesItemDir(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}

	var slot Slot[*item]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	added, err := slot.Add("api", DefaultContext)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	itemDir := filepath.Join(slot.Path(), "api")
	if err := os.RemoveAll(itemDir); err != nil {
		t.Fatalf("RemoveAll(itemDir) error = %v", err)
	}

	again, err := slot.Add("api", DefaultContext)
	if err != nil {
		t.Fatalf("second Add() error = %v", err)
	}

	if again != added {
		t.Fatalf("second Add() returned a different cached item")
	}
	if _, err := os.Stat(itemDir); err != nil {
		t.Fatalf("os.Stat(item root) error = %v", err)
	}
	if _, err := os.Stat(added.Config.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(config file) error = %v, want not-exist", err)
	}
}

func TestSlotDeleteRemovesDiskAndCache(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}

	var slot Slot[*item]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	added, err := slot.Add("api", DefaultContext)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := slot.Delete("api", DefaultContext); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(slot.Path(), "api")); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(item root) error = %v, want not-exist", err)
	}
	if _, ok := slot.Get("api"); ok {
		t.Fatalf("Get(\"api\") ok = true after Delete(), want false")
	}
	if _, err := os.Stat(added.Config.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(config file) error = %v, want not-exist", err)
	}
}

func TestSlotEntriesReturnSortedSnapshot(t *testing.T) {
	type item struct {
		Name string
	}

	var slot Slot[*item]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	worker := &item{Name: "worker"}
	api := &item{Name: "api"}

	slot.Put("worker", worker)
	slot.Put("api", api)

	entries := slot.Entries()
	if len(entries) != 2 {
		t.Fatalf("len(Entries()) = %d, want 2", len(entries))
	}
	if entries[0].Name != "api" || entries[0].Item != api {
		t.Fatalf("Entries()[0] = {%q, %p}, want {%q, %p}", entries[0].Name, entries[0].Item, "api", api)
	}
	if entries[1].Name != "worker" || entries[1].Item != worker {
		t.Fatalf("Entries()[1] = {%q, %p}, want {%q, %p}", entries[1].Name, entries[1].Item, "worker", worker)
	}

	slot.Remove("api")
	if len(entries) != 2 {
		t.Fatalf("len(Entries()) after Remove() = %d, want snapshot to stay 2", len(entries))
	}
}

func TestSlotAllIteratesSortedCachedItems(t *testing.T) {
	type item struct {
		Name string
	}

	var slot Slot[*item]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	worker := &item{Name: "worker"}
	api := &item{Name: "api"}

	slot.Put("worker", worker)
	slot.Put("api", api)

	var gotNames []string
	var gotItems []*item
	for name, item := range slot.All() {
		gotNames = append(gotNames, name)
		gotItems = append(gotItems, item)
	}

	if len(gotNames) != 2 {
		t.Fatalf("len(names from All()) = %d, want 2", len(gotNames))
	}
	if gotNames[0] != "api" || gotItems[0] != api {
		t.Fatalf("first All() item = {%q, %p}, want {%q, %p}", gotNames[0], gotItems[0], "api", api)
	}
	if gotNames[1] != "worker" || gotItems[1] != worker {
		t.Fatalf("second All() item = {%q, %p}, want {%q, %p}", gotNames[1], gotItems[1], "worker", worker)
	}
}

func TestSlotLenTracksCachedItems(t *testing.T) {
	var slot Slot[int]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	if got := slot.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}

	slot.Put("api", 1)
	slot.Put("worker", 2)
	if got := slot.Len(); got != 2 {
		t.Fatalf("Len() after Put() = %d, want 2", got)
	}

	slot.Remove("api")
	if got := slot.Len(); got != 1 {
		t.Fatalf("Len() after Remove() = %d, want 1", got)
	}

	slot.Clear()
	if got := slot.Len(); got != 0 {
		t.Fatalf("Len() after Clear() = %d, want 0", got)
	}
}

func TestSlotRejectsNamesOutsideDirectChildren(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"config.json"`
	}

	var slot Slot[*item]
	slot.ComposePath(filepath.Join(t.TempDir(), "services"))

	invalid := []string{"", ".", "..", "nested/api", "../api", `/tmp/api`}
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
