package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultContextUsesSafePathPolicy(t *testing.T) {
	if got := DefaultContext.PathSafetyPolicy; got != PathSafetyRejectSymlinkParents {
		t.Fatalf("DefaultContext.PathSafetyPolicy = %v, want %v", got, PathSafetyRejectSymlinkParents)
	}
}

func TestFileWriteBytesRejectsSymlinkLeaf(t *testing.T) {
	base := t.TempDir()
	targetPath := filepath.Join(base, "target.txt")
	linkPath := filepath.Join(base, "payload.txt")

	if err := os.WriteFile(targetPath, []byte("original"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	file := NewFile(linkPath)
	if err := file.WriteBytes([]byte("updated"), DefaultContext); err == nil {
		t.Fatal("WriteBytes() error = nil, want non-nil for symlink leaf")
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("os.ReadFile(target) error = %v", err)
	}
	if string(got) != "original" {
		t.Fatalf("target content = %q, want %q", got, "original")
	}
}

func TestDirDeleteIfExistsRejectsSymlinkLeaf(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkPath := filepath.Join(base, "cache")

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	dir := NewDir(linkPath)
	if err := dir.DeleteIfExists(DefaultContext); err == nil {
		t.Fatal("DeleteIfExists() error = nil, want non-nil for symlink leaf")
	}

	if _, err := os.Stat(realDir); err != nil {
		t.Fatalf("os.Stat(real) error = %v", err)
	}
}

func TestFileWriteBytesRejectsSymlinkParentByDefault(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	file := NewFile(filepath.Join(linkParent, "payload.txt"))
	if err := file.WriteBytes([]byte("blocked"), DefaultContext); err == nil {
		t.Fatal("WriteBytes() error = nil, want non-nil for symlink parent")
	}

	if _, err := os.Stat(filepath.Join(realDir, "payload.txt")); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(real payload) error = %v, want not-exist", err)
	}
}

func TestFileWriteBytesCanFollowSymlinkParentWhenEnabled(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	ctx := DefaultContext
	ctx.PathSafetyPolicy = PathSafetyFollowSymlinks

	file := NewFile(filepath.Join(linkParent, "payload.txt"))
	if err := file.WriteBytes([]byte("allowed"), ctx); err != nil {
		t.Fatalf("WriteBytes() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(realDir, "payload.txt"))
	if err != nil {
		t.Fatalf("os.ReadFile(real payload) error = %v", err)
	}
	if string(got) != "allowed" {
		t.Fatalf("real payload content = %q, want %q", got, "allowed")
	}
}

func TestExecEnsureRejectsSymlinkLeaf(t *testing.T) {
	base := t.TempDir()
	targetPath := filepath.Join(base, "target.sh")
	linkPath := filepath.Join(base, "tool.sh")

	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	execFile := NewExec(linkPath)
	if err := execFile.Ensure(DefaultContext); err == nil {
		t.Fatal("Ensure() error = nil, want non-nil for symlink leaf")
	}
}

func TestLinkSyncRejectsSymlinkParentByDefault(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	var link Link
	link.ComposePath(filepath.Join(linkParent, "config.link"))
	link.SetTarget("configs/app.yaml")

	result, err := link.Sync(DefaultContext)
	if err == nil {
		t.Fatal("Sync() error = nil, want non-nil for symlink parent")
	}
	if result != SyncFailed {
		t.Fatalf("Sync() result = %v, want %v", result, SyncFailed)
	}
}

func TestFileCopyToPathRejectsSymlinkSourceLeaf(t *testing.T) {
	base := t.TempDir()
	targetPath := filepath.Join(base, "real-source.txt")
	linkPath := filepath.Join(base, "source.txt")
	dstPath := filepath.Join(base, "dest.txt")

	if err := os.WriteFile(targetPath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	src := NewFile(linkPath)
	if err := src.CopyToPath(dstPath, DefaultCopyOptions); err == nil {
		t.Fatal("CopyToPath() error = nil, want non-nil for symlink source leaf")
	}
}

func TestFileCopyToPathRejectsSymlinkDestinationParentByDefault(t *testing.T) {
	base := t.TempDir()
	srcPath := filepath.Join(base, "source.txt")
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")

	if err := os.WriteFile(srcPath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(source) error = %v", err)
	}
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	src := NewFile(srcPath)
	dstPath := filepath.Join(linkParent, "dest.txt")
	if err := src.CopyToPath(dstPath, DefaultCopyOptions); err == nil {
		t.Fatal("CopyToPath() error = nil, want non-nil for symlink destination parent")
	}
}

func TestFileCopyToPathCanFollowSymlinkDestinationParentWhenEnabled(t *testing.T) {
	base := t.TempDir()
	srcPath := filepath.Join(base, "source.txt")
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")

	if err := os.WriteFile(srcPath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(source) error = %v", err)
	}
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	src := NewFile(srcPath)
	dstPath := filepath.Join(linkParent, "dest.txt")
	opts := DefaultCopyOptions
	opts.PathSafetyPolicy = PathSafetyFollowSymlinks

	if err := src.CopyToPath(dstPath, opts); err != nil {
		t.Fatalf("CopyToPath() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(realDir, "dest.txt"))
	if err != nil {
		t.Fatalf("os.ReadFile(real dest) error = %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("real dest content = %q, want %q", got, "payload")
	}
}
