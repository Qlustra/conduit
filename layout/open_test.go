package layout

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileOpenReadReadsExistingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	handle, err := file.OpenRead(DefaultContext, OpenExisting)
	if err != nil {
		t.Fatalf("OpenRead() error = %v", err)
	}
	defer handle.Close()

	data, err := io.ReadAll(handle)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("read content = %q, want %q", data, "payload")
	}
}

func TestFileOpenReadCanCreateMissingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "nested", "payload.txt"))

	handle, err := file.OpenRead(DefaultContext, OpenOrCreate)
	if err != nil {
		t.Fatalf("OpenRead(OpenOrCreate) error = %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	info, err := os.Stat(file.Path())
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("created file size = %d, want 0", info.Size())
	}
}

func TestFileOpenRewriteCreatesParentsAndTruncates(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "nested", "payload.txt"))
	if err := os.MkdirAll(filepath.Dir(file.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(file.Path(), []byte("stale"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	handle, err := file.OpenRewrite(DefaultContext, OpenOrCreate)
	if err != nil {
		t.Fatalf("OpenRewrite() error = %v", err)
	}
	if _, err := handle.WriteString("fresh"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "fresh" {
		t.Fatalf("file content = %q, want %q", got, "fresh")
	}
}

func TestFileOpenExclusiveFailsWhenFileExists(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	handle, err := file.OpenWrite(DefaultContext, OpenOrCreateExclusive)
	if err == nil {
		handle.Close()
		t.Fatal("OpenWrite(OpenOrCreateExclusive) error = nil, want non-nil")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("OpenWrite(OpenOrCreateExclusive) error = %v, want os.ErrExist", err)
	}
}

func TestFileOpenAppendAlwaysAppends(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	handle, err := file.OpenAppend(DefaultContext, OpenExisting)
	if err != nil {
		t.Fatalf("OpenAppend() error = %v", err)
	}
	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if _, err := handle.WriteString("beta"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "alphabeta" {
		t.Fatalf("file content = %q, want %q", got, "alphabeta")
	}
}

func TestFileOpenRewriteAppendTruncatesThenAppends(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "app.log"))
	if err := os.WriteFile(file.Path(), []byte("stale log"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	handle, err := file.OpenRewriteAppend(DefaultContext, OpenExisting)
	if err != nil {
		t.Fatalf("OpenRewriteAppend() error = %v", err)
	}
	if _, err := handle.WriteString("alpha"); err != nil {
		t.Fatalf("WriteString(alpha) error = %v", err)
	}
	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if _, err := handle.WriteString("beta"); err != nil {
		t.Fatalf("WriteString(beta) error = %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "alphabeta" {
		t.Fatalf("file content = %q, want %q", got, "alphabeta")
	}
}

func TestFileOpenReadWriteSupportsReadAndWrite(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	handle, err := file.OpenReadWrite(DefaultContext, OpenExisting)
	if err != nil {
		t.Fatalf("OpenReadWrite() error = %v", err)
	}
	defer handle.Close()

	buf := make([]byte, 5)
	if _, err := io.ReadFull(handle, buf); err != nil {
		t.Fatalf("ReadFull() error = %v", err)
	}
	if string(buf) != "alpha" {
		t.Fatalf("read content = %q, want %q", buf, "alpha")
	}
	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if _, err := handle.WriteString("omega"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
}

func TestFileOpenRejectsUnsupportedPolicy(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))

	_, err := file.OpenRead(DefaultContext, OpenPolicy(99))
	if err == nil {
		t.Fatal("OpenRead(unsupported policy) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unsupported open policy") {
		t.Fatalf("OpenRead(unsupported policy) error = %v, want unsupported policy", err)
	}
}

func TestFileOpenRejectsSymlinkLeaf(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target.txt")
	link := NewFile(filepath.Join(base, "payload.txt"))

	if err := os.WriteFile(target, []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(target, link.Path()); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if handle, err := link.OpenRead(DefaultContext, OpenExisting); err == nil {
		handle.Close()
		t.Fatal("OpenRead() error = nil, want non-nil for symlink leaf")
	}
}

func TestFileOpenRejectsSymlinkParentByDefault(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")
	file := NewFile(filepath.Join(linkParent, "payload.txt"))

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if handle, err := file.OpenWrite(DefaultContext, OpenOrCreate); err == nil {
		handle.Close()
		t.Fatal("OpenWrite() error = nil, want non-nil for symlink parent")
	}
	if _, err := os.Stat(filepath.Join(realDir, "payload.txt")); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(real payload) error = %v, want not-exist", err)
	}
}

func TestFileOpenCanFollowSymlinkParentWhenEnabled(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")
	file := NewFile(filepath.Join(linkParent, "payload.txt"))

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	ctx := DefaultContext
	ctx.PathSafetyPolicy = PathSafetyFollowSymlinks
	handle, err := file.OpenWrite(ctx, OpenOrCreate)
	if err != nil {
		t.Fatalf("OpenWrite() error = %v", err)
	}
	if _, err := handle.WriteString("payload"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(realDir, "payload.txt"))
	if err != nil {
		t.Fatalf("os.ReadFile(real payload) error = %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("real payload = %q, want %q", got, "payload")
	}
}
