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
