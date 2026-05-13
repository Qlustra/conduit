package layout

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDirListReturnsSortedEntries(t *testing.T) {
	root := NewDir(t.TempDir())

	if err := os.WriteFile(root.File("b.txt").Path(), []byte("b"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(b.txt) error = %v", err)
	}
	if err := os.Mkdir(root.Dir("a").Path(), 0o755); err != nil {
		t.Fatalf("os.Mkdir(a) error = %v", err)
	}

	entries, err := root.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(List()) = %d, want %d", len(entries), 2)
	}
	if got := entries[0].Name(); got != "a" {
		t.Fatalf("List()[0].Name() = %q, want %q", got, "a")
	}
	if got := entries[1].Name(); got != "b.txt" {
		t.Fatalf("List()[1].Name() = %q, want %q", got, "b.txt")
	}
}

func TestDirChangeToChangesWorkingDirectory(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("os.Chdir(restore) error = %v", err)
		}
	}()

	target := NewDir(filepath.Join(t.TempDir(), "workspace"))
	if err := os.MkdirAll(target.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	if err := target.ChangeTo(); err != nil {
		t.Fatalf("ChangeTo() error = %v", err)
	}

	got, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() after ChangeTo() error = %v", err)
	}
	if got != target.Path() {
		t.Fatalf("cwd = %q, want %q", got, target.Path())
	}
}

func TestFileTruncateShrinksFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))

	if err := os.WriteFile(file.Path(), []byte("abcdef"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := file.Truncate(3); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}

	data, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if got := string(data); got != "abc" {
		t.Fatalf("truncated content = %q, want %q", got, "abc")
	}
}

func TestFileIsExecutableChecksRegularFiles(t *testing.T) {
	base := t.TempDir()
	execFile := NewFile(filepath.Join(base, "bin", "tool"))
	plainFile := NewFile(filepath.Join(base, "plain.txt"))
	dirHandle := NewFile(filepath.Join(base, "dir"))

	if err := os.MkdirAll(filepath.Dir(execFile.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(bin) error = %v", err)
	}
	if err := os.WriteFile(execFile.Path(), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile(exec) error = %v", err)
	}
	if err := os.WriteFile(plainFile.Path(), []byte("plain"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(plain) error = %v", err)
	}
	if err := os.Mkdir(dirHandle.Path(), 0o755); err != nil {
		t.Fatalf("os.Mkdir(dir) error = %v", err)
	}

	if !execFile.IsExecutable() {
		t.Fatal("IsExecutable() for executable file = false, want true")
	}
	if plainFile.IsExecutable() {
		t.Fatal("IsExecutable() for non-executable file = true, want false")
	}
	if dirHandle.IsExecutable() {
		t.Fatal("IsExecutable() for directory = true, want false")
	}
}

func TestFileAndDirChownCallThroughToOS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Chown is not supported on Windows")
	}

	base := t.TempDir()
	file := NewFile(filepath.Join(base, "payload.txt"))
	dir := NewDir(filepath.Join(base, "workspace"))

	if err := os.WriteFile(file.Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := os.Mkdir(dir.Path(), 0o755); err != nil {
		t.Fatalf("os.Mkdir() error = %v", err)
	}

	if err := file.Chown(-1, -1); err != nil {
		t.Fatalf("File.Chown() error = %v", err)
	}
	if err := dir.Chown(-1, -1); err != nil {
		t.Fatalf("Dir.Chown() error = %v", err)
	}
}
