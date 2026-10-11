package conduit_test

import (
	. "github.com/qlustra/conduit/layout"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderLayoutRendersTextTemplatesIntoMemory(t *testing.T) {
	manager, layout := composeTree[testRenderLayout](t, t.TempDir())

	layout.Template.SetContext(testRenderContext{
		Name:  "root",
		Items: []string{"a", "b"},
	})
	layout.BuiltIn.SetContext(testRenderContext{
		Name:  "built-in",
		Items: []string{"x", "y"},
	})
	layout.Both.SetContext(testRenderContext{
		Name:  "ignored",
		Items: []string{"ignored"},
	})
	layout.Broken = nil

	if err := manager.Render(); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	value, ok := layout.Template.Get()
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if value != "root:a,b" {
		t.Fatalf("Get() = %q, want %q", value, "root:a,b")
	}
	if got := layout.Template.MemoryState(); got != MemoryDirty {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryDirty)
	}
	if got := layout.BuiltIn.MustGet(); got != "built-in:x,y" {
		t.Fatalf("BuiltIn.MustGet() = %q, want %q", got, "built-in:x,y")
	}
	if got := layout.Both.MustGet(); got != "custom-path" {
		t.Fatalf("Both.MustGet() = %q, want %q", got, "custom-path")
	}
}

func TestRenderLayoutOnlyRendersCachedSlotItems(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "children", "disk-only"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	manager, layout := composeTree[testRenderLayout](t, root)

	layout.Template.SetContext(testRenderContext{
		Name:  "root",
		Items: []string{"a"},
	})
	layout.BuiltIn.SetContext(testRenderContext{
		Name:  "built-in",
		Items: []string{"y"},
	})
	layout.Both.SetContext(testRenderContext{Name: "ignored"})
	layout.Broken = nil

	cached := layout.Children.MustComposeMember("cached")
	cached.Template.SetContext(testRenderContext{
		Name:  "cached",
		Items: []string{"x"},
	})

	if err := manager.Render(); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if _, ok := layout.Children.Get("disk-only"); ok {
		t.Fatalf("slot item %q was discovered during Render()", "disk-only")
	}

	value, ok := cached.Template.Get()
	if !ok {
		t.Fatalf("cached.Template.Get() ok = false, want true")
	}
	if value != "cached:x" {
		t.Fatalf("cached.Template.Get() = %q, want %q", value, "cached:x")
	}
}

func TestRenderLayoutReturnsRenderErrors(t *testing.T) {
	manager, layout := composeTree[testRenderLayout](t, t.TempDir())

	layout.BuiltIn.SetContext(testRenderContext{Name: "built-in"})
	layout.Both.SetContext(testRenderContext{Name: "ignored"})
	layout.Broken = nil

	err := manager.Render()
	if err == nil {
		t.Fatal("Render() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "render context is not set") {
		t.Fatalf("Render() error = %q, want message containing %q", got, "render context is not set")
	}
}

func TestRenderLayoutUsesRenderableSetter(t *testing.T) {
	manager, layout := composeTree[testRenderLayout](t, t.TempDir())

	layout.Template.SetContext(testRenderContext{Name: "root"})
	layout.BuiltIn.SetContext(testRenderContext{Name: "built-in"})
	layout.Both.SetContext(testRenderContext{Name: "ignored"})
	layout.Broken = &testBrokenRenderable{}

	if err := manager.Render(); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got := layout.Broken.rendered; got != "broken" {
		t.Fatalf("SetRendered() stored %q, want %q", got, "broken")
	}
}
