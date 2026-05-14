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

func TestDirEmptyRemovesChildrenAndPreservesDirectory(t *testing.T) {
	root := NewDir(t.TempDir())

	if err := os.MkdirAll(root.Dir("nested").Dir("deeper").Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(nested) error = %v", err)
	}
	if err := os.WriteFile(root.File("nested/deeper/payload.txt").Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(payload) error = %v", err)
	}
	if err := os.WriteFile(root.File("top.txt").Path(), []byte("top"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(top) error = %v", err)
	}

	if err := root.Empty(); err != nil {
		t.Fatalf("Empty() error = %v", err)
	}

	info, err := os.Stat(root.Path())
	if err != nil {
		t.Fatalf("os.Stat(root) error = %v", err)
	}
	if !info.IsDir() {
		t.Fatal("root after Empty() is not a directory")
	}

	entries, err := root.List()
	if err != nil {
		t.Fatalf("List() after Empty() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(List()) after Empty() = %d, want 0", len(entries))
	}
}

func TestDirEmptyMissingDirectoryIsNoOp(t *testing.T) {
	root := NewDir(filepath.Join(t.TempDir(), "missing"))

	if err := root.Empty(); err != nil {
		t.Fatalf("Empty() on missing dir error = %v", err)
	}
}

func TestDirEmptyRejectsNonDirectoryPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payload.txt")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	root := NewDir(path)
	if err := root.Empty(); err == nil {
		t.Fatal("Empty() on file path error = nil, want non-nil")
	}
}

func TestDirEmptyRejectsSymlinkRoot(t *testing.T) {
	base := t.TempDir()
	target := NewDir(filepath.Join(base, "target"))
	root := NewDir(filepath.Join(base, "workspace"))

	if err := os.MkdirAll(target.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(target) error = %v", err)
	}
	if err := os.WriteFile(target.File("payload.txt").Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(target payload) error = %v", err)
	}
	if err := os.Symlink(target.Path(), root.Path()); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if err := root.Empty(); err == nil {
		t.Fatal("Empty() on symlink root error = nil, want non-nil")
	}

	if _, err := os.Stat(target.File("payload.txt").Path()); err != nil {
		t.Fatalf("os.Stat(target payload) error = %v", err)
	}
}

func TestDirEmptyRemovesSymlinkEntriesWithoutFollowingThem(t *testing.T) {
	base := t.TempDir()
	root := NewDir(filepath.Join(base, "workspace"))
	target := NewDir(filepath.Join(base, "target"))

	if err := os.MkdirAll(root.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(root) error = %v", err)
	}
	if err := os.MkdirAll(target.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(target) error = %v", err)
	}
	if err := os.WriteFile(target.File("payload.txt").Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(target payload) error = %v", err)
	}
	if err := os.Symlink(target.Path(), root.Dir("linked").Path()); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if err := root.Empty(); err != nil {
		t.Fatalf("Empty() error = %v", err)
	}

	if _, err := os.Lstat(root.Dir("linked").Path()); !os.IsNotExist(err) {
		t.Fatalf("os.Lstat(linked) err = %v, want not-exist", err)
	}
	if _, err := os.Stat(target.File("payload.txt").Path()); err != nil {
		t.Fatalf("os.Stat(target payload) error = %v", err)
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
