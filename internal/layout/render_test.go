package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testRenderContext struct {
	Name  string
	Items []string
}

type testTemplateFile struct {
	TextTemplate[testRenderContext]
}

func (f *testTemplateFile) Render() (string, error) {
	ctx, ok := f.GetContext()
	if !ok {
		return "", fmt.Errorf("render context is not set")
	}

	return ctx.Name + ":" + strings.Join(ctx.Items, ","), nil
}

type testTemplatedFile struct {
	TextTemplate[testRenderContext]
}

func (f *testTemplatedFile) Template() string {
	return "{{ .Name }}:{{ range $i, $item := .Items }}{{ if $i }},{{ end }}{{ $item }}{{ end }}"
}

type testBothFile struct {
	TextTemplate[testRenderContext]
}

func (f *testBothFile) Template() string {
	return "template-path"
}

func (f *testBothFile) Render() (string, error) {
	return "custom-path", nil
}

type testBrokenRenderable struct {
	File
	rendered string
}

func (f *testBrokenRenderable) Render() (string, error) {
	return "broken", nil
}

func (f *testBrokenRenderable) SetRendered(value string) {
	f.rendered = value
}

type testRenderChild struct {
	Template testTemplateFile `layout:"child.txt"`
}

type testRenderLayout struct {
	Root     Dir                    `layout:"."`
	Template testTemplateFile       `layout:"README.md"`
	BuiltIn  testTemplatedFile      `layout:"BUILTIN.md"`
	Both     testBothFile           `layout:"BOTH.md"`
	Children Slot[*testRenderChild] `layout:"children"`
	Broken   *testBrokenRenderable  `layout:"broken.txt"`
}

func TestRenderDeepRendersTextTemplatesIntoMemory(t *testing.T) {
	var layout testRenderLayout
	if err := Compose(t.TempDir(), &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

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

	if err := RenderDeep(&layout); err != nil {
		t.Fatalf("RenderDeep() error = %v", err)
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

func TestRenderDeepOnlyRendersCachedSlotItems(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "children", "disk-only"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	var layout testRenderLayout
	if err := Compose(root, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

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

	cached := layout.Children.MustAt("cached")
	cached.Template.SetContext(testRenderContext{
		Name:  "cached",
		Items: []string{"x"},
	})

	if err := RenderDeep(&layout); err != nil {
		t.Fatalf("RenderDeep() error = %v", err)
	}

	if _, ok := layout.Children.Get("disk-only"); ok {
		t.Fatalf("slot item %q was discovered during RenderDeep()", "disk-only")
	}

	value, ok := cached.Template.Get()
	if !ok {
		t.Fatalf("cached.Template.Get() ok = false, want true")
	}
	if value != "cached:x" {
		t.Fatalf("cached.Template.Get() = %q, want %q", value, "cached:x")
	}
}

func TestRenderDeepReturnsRenderErrors(t *testing.T) {
	var layout testRenderLayout
	if err := Compose(t.TempDir(), &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	layout.BuiltIn.SetContext(testRenderContext{Name: "built-in"})
	layout.Both.SetContext(testRenderContext{Name: "ignored"})
	layout.Broken = nil

	err := RenderDeep(&layout)
	if err == nil {
		t.Fatal("RenderDeep() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "render context is not set") {
		t.Fatalf("RenderDeep() error = %q, want message containing %q", got, "render context is not set")
	}
}

func TestRenderDeepUsesRenderableSetter(t *testing.T) {
	var layout testRenderLayout
	if err := Compose(t.TempDir(), &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	layout.Template.SetContext(testRenderContext{Name: "root"})
	layout.BuiltIn.SetContext(testRenderContext{Name: "built-in"})
	layout.Both.SetContext(testRenderContext{Name: "ignored"})
	layout.Broken = &testBrokenRenderable{}

	if err := RenderDeep(&layout); err != nil {
		t.Fatalf("RenderDeep() error = %v", err)
	}
	if got := layout.Broken.rendered; got != "broken" {
		t.Fatalf("SetRendered() stored %q, want %q", got, "broken")
	}
}

func TestTextTemplateRenderTemplateReportsTemplateErrors(t *testing.T) {
	var f testTemplatedFile
	f.SetContext(testRenderContext{Name: "root"})

	_, err := f.RenderTemplate("{{ .Missing }}")
	if err == nil {
		t.Fatal("RenderTemplate() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "can't evaluate field Missing") {
		t.Fatalf("RenderTemplate() error = %q, want message containing %q", got, "can't evaluate field Missing")
	}
}
