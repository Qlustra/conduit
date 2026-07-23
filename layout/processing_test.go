package layout

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectReaderRejectsNilInputs(t *testing.T) {
	if err := InspectReader(nil, func(src io.Reader) error { return nil }); err == nil {
		t.Fatal("InspectReader(nil, fn) error = nil, want non-nil")
	}

	if err := InspectReader(strings.NewReader("payload"), nil); err == nil {
		t.Fatal("InspectReader(src, nil) error = nil, want non-nil")
	}
}

func TestMatchReaderRejectsNilInputs(t *testing.T) {
	if _, err := MatchReader(nil, func(src io.Reader) (bool, error) { return false, nil }); err == nil {
		t.Fatal("MatchReader(nil, fn) error = nil, want non-nil")
	}

	if _, err := MatchReader(strings.NewReader("payload"), nil); err == nil {
		t.Fatal("MatchReader(src, nil) error = nil, want non-nil")
	}
}

func TestInspectTokensReaderRejectsNilInputsAndOptions(t *testing.T) {
	if err := InspectTokensReader(nil, TokenOptions{}, func(token string) error { return nil }); err == nil {
		t.Fatal("InspectTokensReader(nil, opts, fn) error = nil, want non-nil")
	}

	if err := InspectTokensReader(strings.NewReader("payload"), TokenOptions{}, nil); err == nil {
		t.Fatal("InspectTokensReader(src, opts, nil) error = nil, want non-nil")
	}

	err := InspectTokensReader(strings.NewReader("payload"), TokenOptions{MaxTokenSize: -1}, func(token string) error { return nil })
	if err == nil {
		t.Fatal("InspectTokensReader() error = nil, want non-nil for negative max token size")
	}
}

func TestMatchTokensReaderRejectsNilInputsAndOptions(t *testing.T) {
	if _, err := MatchTokensReader(nil, TokenOptions{}, func(token string) (bool, error) { return false, nil }); err == nil {
		t.Fatal("MatchTokensReader(nil, opts, fn) error = nil, want non-nil")
	}

	if _, err := MatchTokensReader(strings.NewReader("payload"), TokenOptions{}, nil); err == nil {
		t.Fatal("MatchTokensReader(src, opts, nil) error = nil, want non-nil")
	}

	_, err := MatchTokensReader(strings.NewReader("payload"), TokenOptions{MaxTokenSize: -1}, func(token string) (bool, error) { return false, nil })
	if err == nil {
		t.Fatal("MatchTokensReader() error = nil, want non-nil for negative max token size")
	}
}

func TestInspectBytesAndStringPassContent(t *testing.T) {
	var bytesGot string
	err := InspectBytes([]byte("alpha"), func(src io.Reader) error {
		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		bytesGot = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("InspectBytes() error = %v", err)
	}
	if bytesGot != "alpha" {
		t.Fatalf("InspectBytes() content = %q, want %q", bytesGot, "alpha")
	}

	var lines []string
	err = InspectString("one\ntwo\n", func(src io.Reader) error {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatalf("InspectString() error = %v", err)
	}
	if strings.Join(lines, ",") != "one,two" {
		t.Fatalf("InspectString() lines = %q, want %q", strings.Join(lines, ","), "one,two")
	}
}

func TestFileInspectReadsExistingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("  \nvalue\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var lines []string
	err := file.Inspect(DefaultContext, func(src io.Reader) error {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if strings.Join(lines, "|") != "  |value" {
		t.Fatalf("Inspect() lines = %q, want %q", strings.Join(lines, "|"), "  |value")
	}
}

func TestFileInspectReturnsCallbackError(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	wantErr := errors.New("stop")
	err := file.Inspect(DefaultContext, func(src io.Reader) error {
		_, _ = io.Copy(io.Discard, src)
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Inspect() error = %v, want %v", err, wantErr)
	}
}

func TestMatchBytesAndStringReturnBoolean(t *testing.T) {
	matched, err := MatchBytes([]byte(" \n value\n"), func(src io.Reader) (bool, error) {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) != "" {
				return true, nil
			}
		}
		return false, scanner.Err()
	})
	if err != nil {
		t.Fatalf("MatchBytes() error = %v", err)
	}
	if !matched {
		t.Fatal("MatchBytes() = false, want true")
	}

	matched, err = MatchString(" \n\t", func(src io.Reader) (bool, error) {
		data, err := io.ReadAll(src)
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(string(data)) != "", nil
	})
	if err != nil {
		t.Fatalf("MatchString() error = %v", err)
	}
	if matched {
		t.Fatal("MatchString() = true, want false")
	}
}

func TestInspectTokensStringUsesCustomSplit(t *testing.T) {
	var tokens []string
	err := InspectTokensString("alpha beta\ngamma", TokenOptions{Split: bufio.ScanWords}, func(token string) error {
		tokens = append(tokens, token)
		return nil
	})
	if err != nil {
		t.Fatalf("InspectTokensString() error = %v", err)
	}
	if strings.Join(tokens, ",") != "alpha,beta,gamma" {
		t.Fatalf("InspectTokensString() tokens = %q, want %q", strings.Join(tokens, ","), "alpha,beta,gamma")
	}
}

func TestInspectTokensStringCanRaiseMaxTokenSize(t *testing.T) {
	long := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	var got string

	err := InspectTokensString(long, TokenOptions{MaxTokenSize: len(long) + 1}, func(token string) error {
		got = token
		return nil
	})
	if err != nil {
		t.Fatalf("InspectTokensString() error = %v", err)
	}
	if got != long {
		t.Fatalf("InspectTokensString() token length = %d, want %d", len(got), len(long))
	}
}

func TestMatchTokensStringUsesCustomSplit(t *testing.T) {
	matched, err := MatchTokensString("alpha beta\ngamma", TokenOptions{Split: bufio.ScanWords}, func(token string) (bool, error) {
		return token == "gamma", nil
	})
	if err != nil {
		t.Fatalf("MatchTokensString() error = %v", err)
	}
	if !matched {
		t.Fatal("MatchTokensString() = false, want true")
	}
}

func TestInspectLinesStringPassesLines(t *testing.T) {
	var lines []string
	err := InspectLinesString("one\ntwo\n", func(line string) error {
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		t.Fatalf("InspectLinesString() error = %v", err)
	}
	if strings.Join(lines, ",") != "one,two" {
		t.Fatalf("InspectLinesString() lines = %q, want %q", strings.Join(lines, ","), "one,two")
	}
}

func TestMatchLinesStringReturnsBoolean(t *testing.T) {
	matched, err := MatchLinesString("# note\n\nvalue\n", func(line string) (bool, error) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("MatchLinesString() error = %v", err)
	}
	if !matched {
		t.Fatal("MatchLinesString() = false, want true")
	}
}

func TestFileMatchReadsExistingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("# note\n\nvalue\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	matched, err := file.Match(DefaultContext, func(src io.Reader) (bool, error) {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			return true, nil
		}
		return false, scanner.Err()
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if !matched {
		t.Fatal("Match() = false, want true")
	}
}

func TestFileMatchReturnsCallbackError(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("payload"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	wantErr := errors.New("boom")
	matched, err := file.Match(DefaultContext, func(src io.Reader) (bool, error) {
		_, _ = io.Copy(io.Discard, src)
		return false, wantErr
	})
	if matched {
		t.Fatal("Match() = true, want false on error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Match() error = %v, want %v", err, wantErr)
	}
}

func TestFileMatchIfExistsReturnsFalseForMissingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "missing.txt"))
	called := false

	matched, err := file.MatchIfExists(DefaultContext, func(src io.Reader) (bool, error) {
		called = true
		return true, nil
	})
	if err != nil {
		t.Fatalf("MatchIfExists() error = %v", err)
	}
	if matched {
		t.Fatal("MatchIfExists() = true, want false for missing file")
	}
	if called {
		t.Fatal("MatchIfExists() called matcher for missing file")
	}
}

func TestFileMatchIfExistsMatchesPresentFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("# note\n\nvalue\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	matched, err := file.MatchIfExists(DefaultContext, func(src io.Reader) (bool, error) {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			return true, nil
		}
		return false, scanner.Err()
	})
	if err != nil {
		t.Fatalf("MatchIfExists() error = %v", err)
	}
	if !matched {
		t.Fatal("MatchIfExists() = false, want true")
	}
}

func TestFileInspectTokensReadsExistingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "payload.txt"))
	if err := os.WriteFile(file.Path(), []byte("alpha beta\ngamma"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var tokens []string
	err := file.InspectTokens(DefaultContext, TokenOptions{Split: bufio.ScanWords}, func(token string) error {
		tokens = append(tokens, token)
		return nil
	})
	if err != nil {
		t.Fatalf("InspectTokens() error = %v", err)
	}
	if strings.Join(tokens, ",") != "alpha,beta,gamma" {
		t.Fatalf("InspectTokens() tokens = %q, want %q", strings.Join(tokens, ","), "alpha,beta,gamma")
	}
}

func TestFileMatchLinesIfExistsReturnsFalseForMissingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "missing.txt"))
	called := false

	matched, err := file.MatchLinesIfExists(DefaultContext, func(line string) (bool, error) {
		called = true
		return true, nil
	})
	if err != nil {
		t.Fatalf("MatchLinesIfExists() error = %v", err)
	}
	if matched {
		t.Fatal("MatchLinesIfExists() = true, want false for missing file")
	}
	if called {
		t.Fatal("MatchLinesIfExists() called matcher for missing file")
	}
}

func TestFileMatchLinesIfExistsRejectsNilCallbackForMissingFile(t *testing.T) {
	file := NewFile(filepath.Join(t.TempDir(), "missing.txt"))

	if _, err := file.MatchLinesIfExists(DefaultContext, nil); err == nil {
		t.Fatal("MatchLinesIfExists() error = nil, want non-nil for nil callback")
	}
}

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
