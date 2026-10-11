package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSlotComposeMemberCachesComposedItems(t *testing.T) {
	type tree struct {
		Members FileSlot[*testMapFile] `layout:"documents" path:"configs"`
	}
	_, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

	first, err := slot.ComposeMember("api.json")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}
	second, err := slot.ComposeMember("api.json")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}

	if first != second {
		t.Fatalf("ComposeMember() returned different cached items for the same key")
	}
	if got := first.Path(); got != filepath.Join(slot.Path(), "api.json") {
		t.Fatalf("Path() = %q", got)
	}
	if keys := slot.Keys(); len(keys) != 1 || keys[0] != "api.json" {
		t.Fatalf("Keys() = %v, want [api.json]", keys)
	}
}

func TestFileSlotDeleteRemovesDiskAndCache(t *testing.T) {
	type tree struct {
		Members FileSlot[*testMapFile] `layout:"documents" path:"configs"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

	added, err := slot.ComposeMember("api.json")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}

	if _, err := manager.Ensure(DefaultContext); err != nil {
		t.Fatal(err)
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

func TestFileSlotRequireNeedsExistingFile(t *testing.T) {
	type tree struct {
		Members FileSlot[*testMapFile] `layout:"documents" path:"configs"`
	}
	_, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

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
