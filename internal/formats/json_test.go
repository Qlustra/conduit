package formats

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qlustra/conduit/internal/layout"
)

func TestJSONFileSaveAndLoad(t *testing.T) {
	var f JSONFile[map[string]string]
	f.ComposePath(filepath.Join(t.TempDir(), "config.json"))
	f.Set(map[string]string{"name": "value"})

	if err := f.Save(layout.DefaultContext); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != "{\n  \"name\": \"value\"\n}\n" {
		t.Fatalf("file contents = %q", got)
	}

	var loaded JSONFile[map[string]string]
	loaded.ComposePath(f.Path())
	ok, err := loaded.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok {
		t.Fatalf("Load() ok = false, want true")
	}
	if got := loaded.MustGet()["name"]; got != "value" {
		t.Fatalf("MustGet()[\"name\"] = %q, want %q", got, "value")
	}
}
