package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"testing"
)

func TestSlotComposeMemberCachesComposedItems(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"content" path:"config.json"`
	}

	type tree struct {
		Members Slot[*item] `layout:"directories" path:"services"`
	}
	_, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

	first, err := slot.ComposeMember("api")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}
	second, err := slot.ComposeMember("api")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}

	if first != second {
		t.Fatalf("ComposeMember() returned different cached items for the same key")
	}
	if got := first.Config.Path(); got != filepath.Join(slot.Path(), "api", "config.json") {
		t.Fatalf("Config.Path() = %q", got)
	}
	if keys := slot.Keys(); len(keys) != 1 || keys[0] != "api" {
		t.Fatalf("Keys() = %v, want [api]", keys)
	}
}

func TestSlotDeleteRemovesDiskAndCache(t *testing.T) {
	type item struct {
		Config testMapFile `layout:"content" path:"config.json"`
	}

	type tree struct {
		Members Slot[*item] `layout:"directories" path:"services"`
	}
	manager, model := composeTree[tree](t, t.TempDir())
	slot := &model.Members

	added, err := slot.ComposeMember("api")
	if err != nil {
		t.Fatalf("ComposeMember() error = %v", err)
	}

	if _, err := manager.Ensure(DefaultContext); err != nil {
		t.Fatal(err)
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
func TestComposeMemberDoesNotCreateFilesystemEntries(t *testing.T) {
	type tree struct {
		Directories Slot[*struct{}]        `layout:"directories" path:"directories"`
		Documents   FileSlot[*testMapFile] `layout:"documents" path:"documents"`
		Links       LinkSlot[*Link]        `layout:"links" path:"links"`
	}
	base := filepath.Join(t.TempDir(), "uncreated")
	_, model := composeTree[tree](t, base)
	if _, err := model.Directories.ComposeMember("service"); err != nil {
		t.Fatal(err)
	}
	if _, err := model.Documents.ComposeMember("config.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := model.Links.ComposeMember("README"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("composition created filesystem root: %v", err)
	}
}
