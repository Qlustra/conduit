package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriteBytesDirectKeepsExistingBehavior(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "direct.txt"))

	if err := file.WriteBytes([]byte("direct"), DefaultContext); err != nil {
		t.Fatalf("WriteBytes() error = %v", err)
	}
	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "direct" {
		t.Fatalf("content = %q, want direct", got)
	}
}

func TestFileWriteBytesAtomicReplace(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "atomic.txt"))
	if err := file.WriteBytes([]byte("old"), DefaultContext); err != nil {
		t.Fatalf("initial WriteBytes() error = %v", err)
	}

	ctx := DefaultContext
	ctx.WritePolicy = WriteAtomicReplace
	if err := file.WriteBytes([]byte("new"), ctx); err != nil {
		t.Fatalf("atomic WriteBytes() error = %v", err)
	}

	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("content = %q, want new", got)
	}
}

func TestFileWriteBytesAtomicRequiresTempDirWhenConfigured(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "atomic.txt"))
	ctx := DefaultContext
	ctx.WritePolicy = WriteAtomicReplace
	ctx.TempFilePlacement = TempFileDir

	err := file.WriteBytes([]byte("new"), ctx)
	if err == nil {
		t.Fatal("WriteBytes() error = nil, want missing temp dir error")
	}
	if !strings.Contains(err.Error(), "TempFileDir") {
		t.Fatalf("WriteBytes() error = %v, want TempFileDir guidance", err)
	}
}

func TestFileWriteBytesAtomicCustomTempDir(t *testing.T) {
	base := t.TempDir()
	file := NewFile(filepath.Join(base, "out", "atomic.txt"))
	tempDir := NewDir(filepath.Join(base, "tmp"))
	ctx := DefaultContext
	ctx.WritePolicy = WriteAtomicReplace
	ctx.TempFilePlacement = TempFileDir
	ctx.TempDir = tempDir

	if err := file.WriteBytes([]byte("custom-temp"), ctx); err != nil {
		t.Fatalf("WriteBytes() error = %v", err)
	}
	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "custom-temp" {
		t.Fatalf("content = %q, want custom-temp", got)
	}
	entries, err := os.ReadDir(tempDir.Path())
	if err != nil {
		t.Fatalf("os.ReadDir(tempDir) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("temp dir entries = %d, want cleanup", len(entries))
	}
}

func TestFileWriteBytesAtomicAdjacentPlacementCleansTemp(t *testing.T) {
	base := t.TempDir()
	file := NewFile(filepath.Join(base, "atomic.txt"))
	ctx := DefaultContext
	ctx.WritePolicy = WriteAtomicReplace
	ctx.TempFilePlacement = TempFileAdjacent

	if err := file.WriteBytes([]byte("adjacent"), ctx); err != nil {
		t.Fatalf("WriteBytes() error = %v", err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("os.ReadDir(base) error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "atomic.txt" {
		t.Fatalf("entries = %+v, want only destination", entries)
	}
}

func TestFileWriteBytesRejectsUnsupportedAtomicOptions(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "atomic.txt"))
	ctx := DefaultContext
	ctx.WritePolicy = WritePolicy(99)
	err := file.WriteBytes([]byte("data"), ctx)
	if err == nil || !strings.Contains(err.Error(), "unsupported write policy") {
		t.Fatalf("WriteBytes() error = %v, want unsupported write policy", err)
	}

	ctx = DefaultContext
	ctx.WritePolicy = WriteAtomicReplace
	ctx.TempFilePlacement = TempFilePlacement(99)
	err = file.WriteBytes([]byte("data"), ctx)
	if err == nil || !strings.Contains(err.Error(), "unsupported temp file placement") {
		t.Fatalf("WriteBytes() error = %v, want unsupported temp file placement", err)
	}
}

func TestAtomicReplaceErrorIncludesCorrectiveActions(t *testing.T) {
	err := atomicReplaceError("/workspace/out.txt", "/tmp/conduit-out", os.ErrInvalid)
	text := err.Error()
	for _, want := range []string{
		"atomic replace failed",
		"TempFileAdjacent",
		"TempFileDir",
		"WriteDirect",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("atomicReplaceError() = %q, want %q", text, want)
		}
	}
}
