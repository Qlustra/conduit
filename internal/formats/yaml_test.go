package formats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qlustra/conduit/internal/layout"
)

func TestYAMLFileSaveAndLoad(t *testing.T) {
	var f YAMLFile[map[string]string]
	f.ComposePath(filepath.Join(t.TempDir(), "config.yaml"))
	f.Set(map[string]string{"name": "value"})

	if err := f.Save(layout.DefaultContext); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(f.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "name: value") {
		t.Fatalf("file contents = %q, want YAML key/value", string(data))
	}

	var loaded YAMLFile[map[string]string]
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
