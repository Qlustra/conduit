package layout

import (
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
