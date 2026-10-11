package conduit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultLayoutAppliesDefaultsInMemory(t *testing.T) {
	manager, layout := composeTree[testDefaultLayout](t, t.TempDir())

	if err := manager.Default(); err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	if got := layout.Config.MustGet(); got != "default" {
		t.Fatalf("Config.MustGet() = %q, want %q", got, "default")
	}
	if got := layout.Template.MustContext().Name; got != "templated" {
		t.Fatalf("Template.MustContext().Name = %q, want %q", got, "templated")
	}
	if _, err := os.Stat(layout.Config.Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Stat() error = %v, want not exist", err)
	}
}

func TestDefaultLayoutPreservesExistingMemoryState(t *testing.T) {
	manager, layout := composeTree[testDefaultLayout](t, t.TempDir())

	layout.Config.Set("existing")
	layout.Template.SetContext(testRenderContext{Name: "existing"})

	if err := manager.Default(); err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	if got := layout.Config.MustGet(); got != "existing" {
		t.Fatalf("Config.MustGet() = %q, want %q", got, "existing")
	}
	if got := layout.Template.MustContext().Name; got != "existing" {
		t.Fatalf("Template.MustContext().Name = %q, want %q", got, "existing")
	}
}

func TestDefaultLayoutOnlyDefaultsCachedSlotItems(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "children", "disk-only"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	manager, layout := composeTree[testDefaultLayout](t, root)

	cached := layout.Children.MustComposeMember("cached")

	if err := manager.Default(); err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	if _, ok := layout.Children.Get("disk-only"); ok {
		t.Fatalf("slot item %q was discovered during Default()", "disk-only")
	}
	if got := cached.Config.MustGet(); got != "default" {
		t.Fatalf("cached.Config.MustGet() = %q, want %q", got, "default")
	}
}

func TestDefaultLayoutReturnsDefaultErrors(t *testing.T) {
	manager, _ := composeTree[testDefaultErrorLayout](t, t.TempDir())

	err := manager.Default()
	if err == nil {
		t.Fatal("Default() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "default failure") {
		t.Fatalf("Default() error = %q, want message containing %q", got, "default failure")
	}
}
