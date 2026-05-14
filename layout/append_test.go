package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileAppendBytesCreatesParentDirs(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "nested", "payload.txt"))

	if err := file.AppendBytes([]byte("alpha"), 0o755, 0o644); err != nil {
		t.Fatalf("AppendBytes() error = %v", err)
	}

	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "alpha" {
		t.Fatalf("appended content = %q, want %q", got, "alpha")
	}
}

func TestFileAppendStringAppendsToExistingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))

	if err := os.WriteFile(file.Path(), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := file.AppendString("beta", 0o755, 0o644); err != nil {
		t.Fatalf("AppendString() error = %v", err)
	}

	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "alphabeta" {
		t.Fatalf("appended content = %q, want %q", got, "alphabeta")
	}
}

func TestFileAppendFileAppendsSourceContent(t *testing.T) {
	base := t.TempDir()
	dst := NewFile(filepath.Join(base, "dest.txt"))
	src := NewFile(filepath.Join(base, "source.txt"))

	if err := os.WriteFile(dst.Path(), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dest) error = %v", err)
	}
	if err := os.WriteFile(src.Path(), []byte("beta"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(src) error = %v", err)
	}
	if err := dst.AppendFile(src, 0o755, 0o644); err != nil {
		t.Fatalf("AppendFile() error = %v", err)
	}

	got, err := os.ReadFile(dst.Path())
	if err != nil {
		t.Fatalf("os.ReadFile(dest) error = %v", err)
	}
	if string(got) != "alphabeta" {
		t.Fatalf("appended content = %q, want %q", got, "alphabeta")
	}
}

func TestFileAppendFilesPreservesOrder(t *testing.T) {
	base := t.TempDir()
	dst := NewFile(filepath.Join(base, "dest.txt"))
	first := NewFile(filepath.Join(base, "one.txt"))
	second := NewFile(filepath.Join(base, "two.txt"))

	if err := os.WriteFile(first.Path(), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(first) error = %v", err)
	}
	if err := os.WriteFile(second.Path(), []byte("two\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(second) error = %v", err)
	}
	if err := dst.AppendFiles(0o755, 0o644, first, second); err != nil {
		t.Fatalf("AppendFiles() error = %v", err)
	}

	got, err := os.ReadFile(dst.Path())
	if err != nil {
		t.Fatalf("os.ReadFile(dest) error = %v", err)
	}
	if string(got) != "one\ntwo\n" {
		t.Fatalf("appended content = %q, want %q", got, "one\ntwo\n")
	}
}

func TestFileAppendFilesReturnsPartialWriteOnError(t *testing.T) {
	base := t.TempDir()
	dst := NewFile(filepath.Join(base, "dest.txt"))
	first := NewFile(filepath.Join(base, "one.txt"))
	missing := NewFile(filepath.Join(base, "missing.txt"))

	if err := os.WriteFile(first.Path(), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(first) error = %v", err)
	}

	err := dst.AppendFiles(0o755, 0o644, first, missing)
	if err == nil {
		t.Fatal("AppendFiles() error = nil, want non-nil")
	}

	got, readErr := os.ReadFile(dst.Path())
	if readErr != nil {
		t.Fatalf("os.ReadFile(dest) error = %v", readErr)
	}
	if string(got) != "one\n" {
		t.Fatalf("partial content = %q, want %q", got, "one\n")
	}
}

func TestFileAppendFileRejectsSelfAppend(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))

	if err := os.WriteFile(file.Path(), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	err := file.AppendFile(file, 0o755, 0o644)
	if err == nil {
		t.Fatal("AppendFile() error = nil, want non-nil")
	}
}

func TestFileAppendReaderRejectsNilSource(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))

	err := file.AppendReader(nil, 0o755, 0o644)
	if err == nil {
		t.Fatal("AppendReader() error = nil, want non-nil")
	}
}

func TestFileAppendFilesWithNoSourcesIsNoOp(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))

	if err := file.AppendFiles(0o755, 0o644); err != nil {
		t.Fatalf("AppendFiles() error = %v", err)
	}
	if file.Exists() {
		t.Fatal("AppendFiles() created destination for empty input")
	}
}
