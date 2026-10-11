package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileLinkValidateRejectsDirectoryTarget(t *testing.T) {
	var link FileLink
	base := t.TempDir()
	link.ComposePath(filepath.Join(base, "config.link"))

	targetDir := filepath.Join(base, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	link.SetTarget("target")

	err := link.Validate(ValidateOptions{})
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "is not a file") {
		t.Fatalf("Validate() error = %v, want file-link validation error", err)
	}
}
