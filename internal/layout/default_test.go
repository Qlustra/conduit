package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testDefaultFile struct {
	Format[string, textCodec]
}

func (f *testDefaultFile) Default() error {
	f.SetDefault("default")
	return nil
}

type testDefaultTemplate struct {
	TextTemplate[testRenderContext]
}

func (f *testDefaultTemplate) Default() error {
	f.SetDefaultContext(testRenderContext{
		Name:  "templated",
		Items: []string{"a"},
	})
	return nil
}

type testDefaultChild struct {
	Config testDefaultFile `layout:"config.txt"`
}

type testDefaultLayout struct {
	Root     Dir                     `layout:"."`
	Config   testDefaultFile         `layout:"config.txt"`
	Template testDefaultTemplate     `layout:"template.txt"`
	Children Slot[*testDefaultChild] `layout:"children"`
}

type testDefaultErrorFile struct {
	Format[string, textCodec]
}

func (f *testDefaultErrorFile) Default() error {
	return fmt.Errorf("default failure")
}

type testDefaultErrorLayout struct {
	Root   Dir                  `layout:"."`
	Broken testDefaultErrorFile `layout:"broken.txt"`
}

func TestFormatSetDefaultOnlyWhenContentMissing(t *testing.T) {
	var f testDefaultFile

	if applied := f.SetDefault("first"); !applied {
		t.Fatal("SetDefault() = false, want true")
	}
	if got := f.MustGet(); got != "first" {
		t.Fatalf("MustGet() = %q, want %q", got, "first")
	}
	if got := f.MemoryState(); got != MemoryDirty {
		t.Fatalf("MemoryState() = %v, want %v", got, MemoryDirty)
	}

	if applied := f.SetDefault("second"); applied {
		t.Fatal("SetDefault() = true, want false when content already exists")
	}
	if got := f.MustGet(); got != "first" {
		t.Fatalf("MustGet() after second default = %q, want %q", got, "first")
	}
}

func TestTextTemplateSetDefaultContextOnlyWhenContextMissing(t *testing.T) {
	var f TextTemplate[testRenderContext]

	if applied := f.SetDefaultContext(testRenderContext{Name: "first"}); !applied {
		t.Fatal("SetDefaultContext() = false, want true")
	}
	if got := f.MustContext().Name; got != "first" {
		t.Fatalf("MustContext().Name = %q, want %q", got, "first")
	}

	if applied := f.SetDefaultContext(testRenderContext{Name: "second"}); applied {
		t.Fatal("SetDefaultContext() = true, want false when context already exists")
	}
	if got := f.MustContext().Name; got != "first" {
		t.Fatalf("MustContext().Name after second default = %q, want %q", got, "first")
	}
}

func TestDefaultDeepAppliesDefaultsInMemory(t *testing.T) {
	var layout testDefaultLayout
	if err := Compose(t.TempDir(), &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if err := DefaultDeep(&layout); err != nil {
		t.Fatalf("DefaultDeep() error = %v", err)
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

func TestDefaultDeepPreservesExistingMemoryState(t *testing.T) {
	var layout testDefaultLayout
	if err := Compose(t.TempDir(), &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	layout.Config.Set("existing")
	layout.Template.SetContext(testRenderContext{Name: "existing"})

	if err := DefaultDeep(&layout); err != nil {
		t.Fatalf("DefaultDeep() error = %v", err)
	}

	if got := layout.Config.MustGet(); got != "existing" {
		t.Fatalf("Config.MustGet() = %q, want %q", got, "existing")
	}
	if got := layout.Template.MustContext().Name; got != "existing" {
		t.Fatalf("Template.MustContext().Name = %q, want %q", got, "existing")
	}
}

func TestDefaultDeepOnlyDefaultsCachedSlotItems(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "children", "disk-only"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	var layout testDefaultLayout
	if err := Compose(root, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	cached := layout.Children.MustAt("cached")

	if err := DefaultDeep(&layout); err != nil {
		t.Fatalf("DefaultDeep() error = %v", err)
	}

	if _, ok := layout.Children.Get("disk-only"); ok {
		t.Fatalf("slot item %q was discovered during DefaultDeep()", "disk-only")
	}
	if got := cached.Config.MustGet(); got != "default" {
		t.Fatalf("cached.Config.MustGet() = %q, want %q", got, "default")
	}
}

func TestDefaultDeepReturnsDefaultErrors(t *testing.T) {
	var layout testDefaultErrorLayout
	if err := Compose(t.TempDir(), &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	err := DefaultDeep(&layout)
	if err == nil {
		t.Fatal("DefaultDeep() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "default failure") {
		t.Fatalf("DefaultDeep() error = %q, want message containing %q", got, "default failure")
	}
}
