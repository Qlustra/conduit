package layout

import (
	"path/filepath"
	"testing"
)

type testJSONFile struct {
	testMapFile
}

func TestDirPathHelpers(t *testing.T) {
	dir := NewDir(filepath.Join("workspace", "services", "api.v2"))

	if got := dir.Base(); got != "api.v2" {
		t.Fatalf("Base() = %q, want %q", got, "api.v2")
	}
	if got := dir.Stem(); got != "api" {
		t.Fatalf("Stem() = %q, want %q", got, "api")
	}
	if got := dir.ParentPath(); got != filepath.Join("workspace", "services") {
		t.Fatalf("ParentPath() = %q, want %q", got, filepath.Join("workspace", "services"))
	}
	if got := dir.ParentDir().Path(); got != filepath.Join("workspace", "services") {
		t.Fatalf("ParentDir().Path() = %q, want %q", got, filepath.Join("workspace", "services"))
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
	if got := file.ParentPath(); got != filepath.Join("workspace", "configs") {
		t.Fatalf("ParentPath() = %q, want %q", got, filepath.Join("workspace", "configs"))
	}
	if got := file.ParentDir().Path(); got != filepath.Join("workspace", "configs") {
		t.Fatalf("ParentDir().Path() = %q, want %q", got, filepath.Join("workspace", "configs"))
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

func TestComposedPathsAreUnavailableWithoutComposition(t *testing.T) {
	dir := NewDir("workspace")
	file := NewFile(filepath.Join("workspace", "config.yaml"))
	slot := NewSlot[*struct{}](dir)
	fileSlot := NewFileSlot[testJSONFile](dir)

	if _, ok := dir.ComposedBaseDir(); ok {
		t.Fatal("Dir.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := dir.ComposedRelativePath(); ok {
		t.Fatal("Dir.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := dir.JoinComposedPath("child"); ok {
		t.Fatal("Dir.JoinComposedPath() ok = true, want false")
	}

	if _, ok := file.ComposedBaseDir(); ok {
		t.Fatal("File.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := file.ComposedRelativePath(); ok {
		t.Fatal("File.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := file.JoinComposedPath("child"); ok {
		t.Fatal("File.JoinComposedPath() ok = true, want false")
	}
	if _, ok := file.ParentDir().ComposedRelativePath(); ok {
		t.Fatal("File.ParentDir().ComposedRelativePath() ok = true, want false")
	}

	if _, ok := slot.ComposedBaseDir(); ok {
		t.Fatal("Slot.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := slot.ComposedRelativePath(); ok {
		t.Fatal("Slot.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := slot.JoinComposedPath("child"); ok {
		t.Fatal("Slot.JoinComposedPath() ok = true, want false")
	}
	if _, ok := fileSlot.ComposedBaseDir(); ok {
		t.Fatal("FileSlot.ComposedBaseDir() ok = true, want false")
	}
	if _, ok := fileSlot.ComposedRelativePath(); ok {
		t.Fatal("FileSlot.ComposedRelativePath() ok = true, want false")
	}
	if _, ok := fileSlot.JoinComposedPath("child"); ok {
		t.Fatal("FileSlot.JoinComposedPath() ok = true, want false")
	}
}
