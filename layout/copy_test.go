package layout

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileCopyIntoDirPreservesContentAndMode(t *testing.T) {
	base := t.TempDir()
	src := NewFile(filepath.Join(base, "source.txt"))
	dstDir := NewDir(filepath.Join(base, "out"))

	content := bytes.Repeat([]byte("copy me\n"), 256)
	if err := os.WriteFile(src.Path(), content, 0o640); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := src.CopyIntoDir(dstDir, DefaultCopyOptions); err != nil {
		t.Fatalf("CopyIntoDir() error = %v", err)
	}

	dst := dstDir.File(src.Base())
	got, err := os.ReadFile(dst.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatal("copied file content mismatch")
	}

	info, err := os.Stat(dst.Path())
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("copied file mode = %#o, want %#o", got, 0o640)
	}
}

func TestFileCopyToPathRejectsExistingDestinationByDefault(t *testing.T) {
	base := t.TempDir()
	src := NewFile(filepath.Join(base, "source.txt"))
	dst := filepath.Join(base, "dest.txt")

	if err := os.WriteFile(src.Path(), []byte("source"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(dst, []byte("dest"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dest) error = %v", err)
	}

	err := src.CopyToPath(dst, DefaultCopyOptions)
	if err == nil {
		t.Fatal("CopyToPath() error = nil, want non-nil")
	}
}

func TestFileCopyToFileCanReplaceDestination(t *testing.T) {
	base := t.TempDir()
	src := NewFile(filepath.Join(base, "source.txt"))
	dst := NewFile(filepath.Join(base, "dest.txt"))

	if err := os.WriteFile(src.Path(), []byte("source"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(dst.Path(), []byte("dest"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dest) error = %v", err)
	}

	opts := DefaultCopyOptions
	opts.Overwrite = CopyOverwriteReplace

	if err := src.CopyToFile(dst, opts); err != nil {
		t.Fatalf("CopyToFile() error = %v", err)
	}

	got, err := os.ReadFile(dst.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "source" {
		t.Fatalf("copied content = %q, want %q", got, "source")
	}
}

func TestDirCopyIntoDirPreservesSymlinksByDefault(t *testing.T) {
	base := t.TempDir()
	src := NewDir(filepath.Join(base, "project"))
	parent := NewDir(filepath.Join(base, "out"))

	if err := os.MkdirAll(filepath.Join(src.Path(), "nested"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(src.Path(), "nested", "config.yaml"), []byte("port: 8080\n"), 0o640); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := os.Symlink("nested/config.yaml", filepath.Join(src.Path(), "config.link")); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if err := src.CopyIntoDir(parent, DefaultCopyOptions); err != nil {
		t.Fatalf("CopyIntoDir() error = %v", err)
	}

	dst := parent.Dir(src.Base())
	linkPath := filepath.Join(dst.Path(), "config.link")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("os.Readlink() error = %v", err)
	}
	if target != "nested/config.yaml" {
		t.Fatalf("os.Readlink() = %q, want %q", target, "nested/config.yaml")
	}

	fileData, err := os.ReadFile(filepath.Join(dst.Path(), "nested", "config.yaml"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(fileData) != "port: 8080\n" {
		t.Fatalf("copied file content = %q, want %q", fileData, "port: 8080\n")
	}
}

func TestDirCopyToDirCanFollowSymlinks(t *testing.T) {
	base := t.TempDir()
	src := NewDir(filepath.Join(base, "project"))
	dst := NewDir(filepath.Join(base, "snapshot"))

	if err := os.MkdirAll(filepath.Join(src.Path(), "nested"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(src.Path(), "nested", "config.yaml"), []byte("name: api\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := os.Symlink("nested/config.yaml", filepath.Join(src.Path(), "config.link")); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	opts := DefaultCopyOptions
	opts.Symlinks = CopySymlinkFollow

	if err := src.CopyToDir(dst, opts); err != nil {
		t.Fatalf("CopyToDir() error = %v", err)
	}

	linkInfo, err := os.Lstat(filepath.Join(dst.Path(), "config.link"))
	if err != nil {
		t.Fatalf("os.Lstat() error = %v", err)
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatal("copied entry is still a symlink, want regular file")
	}

	got, err := os.ReadFile(filepath.Join(dst.Path(), "config.link"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "name: api\n" {
		t.Fatalf("copied file content = %q, want %q", got, "name: api\n")
	}
}

func TestDirCopyToPathRejectsSymlinksWhenConfigured(t *testing.T) {
	base := t.TempDir()
	src := NewDir(filepath.Join(base, "project"))

	if err := os.MkdirAll(src.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.Symlink("missing.txt", filepath.Join(src.Path(), "config.link")); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	opts := DefaultCopyOptions
	opts.Symlinks = CopySymlinkReject

	err := src.CopyToPath(filepath.Join(base, "snapshot"), opts)
	if err == nil {
		t.Fatal("CopyToPath() error = nil, want non-nil")
	}
}

func TestDirCopyToPathCanReplaceExistingTree(t *testing.T) {
	base := t.TempDir()
	src := NewDir(filepath.Join(base, "project"))
	dst := filepath.Join(base, "snapshot")

	if err := os.MkdirAll(filepath.Join(src.Path(), "nested"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(src.Path(), "nested", "fresh.txt"), []byte("fresh"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(source) error = %v", err)
	}

	if err := os.MkdirAll(filepath.Join(dst, "stale"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(dest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "stale", "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dest) error = %v", err)
	}

	opts := DefaultCopyOptions
	opts.Overwrite = CopyOverwriteReplace

	if err := src.CopyToPath(dst, opts); err != nil {
		t.Fatalf("CopyToPath() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "stale", "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(stale file) error = %v, want not-exist", err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "nested", "fresh.txt"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "fresh" {
		t.Fatalf("copied file content = %q, want %q", got, "fresh")
	}
}

func TestDirCopyToPathRejectsDestinationInsideSource(t *testing.T) {
	base := t.TempDir()
	src := NewDir(filepath.Join(base, "project"))

	if err := os.MkdirAll(src.Path(), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	err := src.CopyToPath(filepath.Join(src.Path(), "nested", "copy"), DefaultCopyOptions)
	if err == nil {
		t.Fatal("CopyToPath() error = nil, want non-nil")
	}
}
