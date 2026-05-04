package layout

import (
	"path/filepath"
	"testing"
)

func TestDirPathHelpers(t *testing.T) {
	dir := NewDir(filepath.Join("workspace", "services", "api.v2"))

	if got := dir.Base(); got != "api.v2" {
		t.Fatalf("Base() = %q, want %q", got, "api.v2")
	}
	if got := dir.Stem(); got != "api" {
		t.Fatalf("Stem() = %q, want %q", got, "api")
	}
}

func TestFilePathHelpers(t *testing.T) {
	file := NewFile(filepath.Join("workspace", "configs", "archive.tar.gz"))

	if got := file.Base(); got != "archive.tar.gz" {
		t.Fatalf("Base() = %q, want %q", got, "archive.tar.gz")
	}
	if got := file.Ext(); got != ".gz" {
		t.Fatalf("Ext() = %q, want %q", got, ".gz")
	}
	if got := file.Stem(); got != "archive.tar" {
		t.Fatalf("Stem() = %q, want %q", got, "archive.tar")
	}
}

func TestPathHelpersPreserveDotfilesAndExtensionlessNames(t *testing.T) {
	file := NewFile(filepath.Join("workspace", ".env"))
	dir := NewDir(".config")

	if got := file.Ext(); got != "" {
		t.Fatalf("Ext() for dotfile = %q, want empty", got)
	}
	if got := file.Stem(); got != ".env" {
		t.Fatalf("Stem() for dotfile = %q, want %q", got, ".env")
	}
	if got := dir.Stem(); got != ".config" {
		t.Fatalf("Stem() for dot-dir = %q, want %q", got, ".config")
	}
}
