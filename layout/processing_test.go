package layout

import (
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConcatStringsUsesOptions(t *testing.T) {
	opts := ConcatOptions{
		Header:         []byte("begin\n"),
		Footer:         []byte("end\n"),
		Separator:      []byte("\n"),
		FinalSeparator: true,
		EntryPrefix:    []byte("["),
		EntrySuffix:    []byte("]"),
	}

	got := ConcatStrings(opts, "one", "two")
	want := "begin\n[one]\n[two]\nend\n"
	if got != want {
		t.Fatalf("ConcatStrings() = %q, want %q", got, want)
	}
}

func TestFileConcatFilesAllowsDestinationAsSource(t *testing.T) {
	base := t.TempDir()
	dst := NewFile(filepath.Join(base, "bundle.txt"))
	src := NewFile(filepath.Join(base, "extra.txt"))

	if err := os.WriteFile(dst.Path(), []byte("old"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dst) error = %v", err)
	}
	if err := os.WriteFile(src.Path(), []byte("new"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(src) error = %v", err)
	}

	opts := ConcatOptions{Separator: []byte("+"), EntryPrefix: []byte("<"), EntrySuffix: []byte(">")}
	if err := dst.ConcatFiles(DefaultContext, opts, dst, src); err != nil {
		t.Fatalf("ConcatFiles() error = %v", err)
	}

	got, err := os.ReadFile(dst.Path())
	if err != nil {
		t.Fatalf("os.ReadFile(dst) error = %v", err)
	}
	if string(got) != "<old>+<new>" {
		t.Fatalf("ConcatFiles() content = %q, want %q", got, "<old>+<new>")
	}
}

func TestFileConcatFilesLeavesDestinationOnReadError(t *testing.T) {
	base := t.TempDir()
	dst := NewFile(filepath.Join(base, "bundle.txt"))
	src := NewFile(filepath.Join(base, "one.txt"))
	missing := NewFile(filepath.Join(base, "missing.txt"))

	if err := os.WriteFile(dst.Path(), []byte("original"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dst) error = %v", err)
	}
	if err := os.WriteFile(src.Path(), []byte("one"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(src) error = %v", err)
	}

	err := dst.ConcatFiles(DefaultContext, ConcatOptions{}, src, missing)
	if err == nil {
		t.Fatal("ConcatFiles() error = nil, want non-nil")
	}

	got, readErr := os.ReadFile(dst.Path())
	if readErr != nil {
		t.Fatalf("os.ReadFile(dst) error = %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("destination after failed ConcatFiles = %q, want %q", got, "original")
	}
}

func TestFileConcatReadersRejectsNilSourceAndLeavesDestination(t *testing.T) {
	dst := NewFile(filepath.Join(t.TempDir(), "bundle.txt"))
	if err := os.WriteFile(dst.Path(), []byte("original"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dst) error = %v", err)
	}

	err := dst.ConcatReaders(DefaultContext, ConcatOptions{}, strings.NewReader("one"), nil)
	if err == nil {
		t.Fatal("ConcatReaders() error = nil, want non-nil")
	}

	got, readErr := os.ReadFile(dst.Path())
	if readErr != nil {
		t.Fatalf("os.ReadFile(dst) error = %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("destination after failed ConcatReaders = %q, want %q", got, "original")
	}
}

func TestHashHelpersAndFileHashHex(t *testing.T) {
	data := "payload"
	wantBytes := sha256.Sum256([]byte(data))
	wantHex := "239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5"

	if got := HashBytes([]byte(data), sha256.New()); string(got) != string(wantBytes[:]) {
		t.Fatalf("HashBytes() = %x, want %x", got, wantBytes)
	}
	if got := HashString(data, sha256.New()); string(got) != string(wantBytes[:]) {
		t.Fatalf("HashString() = %x, want %x", got, wantBytes)
	}
	gotHex, err := HashHexReader(strings.NewReader(data), sha256.New())
	if err != nil {
		t.Fatalf("HashHexReader() error = %v", err)
	}
	if gotHex != wantHex {
		t.Fatalf("HashHexReader() = %q, want %q", gotHex, wantHex)
	}

	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte(data), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	fileHex, err := file.HashHex(DefaultContext, sha256.New())
	if err != nil {
		t.Fatalf("HashHex() error = %v", err)
	}
	if fileHex != wantHex {
		t.Fatalf("HashHex() = %q, want %q", fileHex, wantHex)
	}
}

func TestTransformStringUsesStreamingCallback(t *testing.T) {
	got, err := TransformString("alpha", upperTransform)
	if err != nil {
		t.Fatalf("TransformString() error = %v", err)
	}
	if got != "ALPHA" {
		t.Fatalf("TransformString() = %q, want %q", got, "ALPHA")
	}
}

func TestFileTransformFileWritesAfterSuccess(t *testing.T) {
	base := t.TempDir()
	dst := NewFile(filepath.Join(base, "out.txt"))
	src := NewFile(filepath.Join(base, "in.txt"))

	if err := os.WriteFile(dst.Path(), []byte("original"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dst) error = %v", err)
	}
	if err := os.WriteFile(src.Path(), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(src) error = %v", err)
	}

	if err := dst.TransformFile(DefaultContext, src, upperTransform); err != nil {
		t.Fatalf("TransformFile() error = %v", err)
	}

	got, err := os.ReadFile(dst.Path())
	if err != nil {
		t.Fatalf("os.ReadFile(dst) error = %v", err)
	}
	if string(got) != "ALPHA" {
		t.Fatalf("TransformFile() content = %q, want %q", got, "ALPHA")
	}
}

func TestFileTransformFailureLeavesDestination(t *testing.T) {
	dst := NewFile(filepath.Join(t.TempDir(), "out.txt"))
	if err := os.WriteFile(dst.Path(), []byte("original"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dst) error = %v", err)
	}

	wantErr := errors.New("boom")
	err := dst.TransformString(DefaultContext, "alpha", func(dst io.Writer, src io.Reader) error {
		if _, err := dst.Write([]byte("partial")); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("TransformString() error = %v, want %v", err, wantErr)
	}

	got, readErr := os.ReadFile(dst.Path())
	if readErr != nil {
		t.Fatalf("os.ReadFile(dst) error = %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("destination after failed transform = %q, want %q", got, "original")
	}
}

func TestFileTransformSelfBuffersBeforeRewrite(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("alpha"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := file.Transform(DefaultContext, upperTransform); err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	got, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "ALPHA" {
		t.Fatalf("Transform() content = %q, want %q", got, "ALPHA")
	}
}

func TestProcessingFileSourceRejectsSymlinkParentByDefault(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkParent := filepath.Join(base, "alias")
	dst := NewFile(filepath.Join(base, "out.txt"))
	src := NewFile(filepath.Join(linkParent, "payload.txt"))

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "payload.txt"), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(real payload) error = %v", err)
	}
	if err := os.WriteFile(dst.Path(), []byte("original"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(dst) error = %v", err)
	}
	if err := os.Symlink(realDir, linkParent); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	if err := dst.TransformFile(DefaultContext, src, upperTransform); err == nil {
		t.Fatal("TransformFile() error = nil, want non-nil for symlink parent")
	}
	if err := dst.ConcatFiles(DefaultContext, ConcatOptions{}, src); err == nil {
		t.Fatal("ConcatFiles() error = nil, want non-nil for symlink parent")
	}
	if _, err := src.Hash(DefaultContext, sha256.New()); err == nil {
		t.Fatal("Hash() error = nil, want non-nil for symlink parent")
	}
}

func upperTransform(dst io.Writer, src io.Reader) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	_, err = dst.Write([]byte(strings.ToUpper(string(data))))
	return err
}
