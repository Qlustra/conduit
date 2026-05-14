package layout

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testValidatorFile struct {
	testMapFile
	calls *int
	err   error
}

func (f *testValidatorFile) Validate(opts ValidateOptions) error {
	if f.calls != nil {
		*f.calls = *f.calls + 1
	}
	return f.err
}

func TestValidateDeepCallsValidatorsAndReports(t *testing.T) {
	type root struct {
		Raw    File              `layout:"raw.txt"`
		Config testValidatorFile `layout:"config.json"`
	}

	var calls int
	var layout root
	layout.Config.calls = &calls

	base := t.TempDir()
	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	var report Report
	opts := ValidateOptions{Reporter: &report}

	if _, err := ValidateDeep(&layout, opts); err != nil {
		t.Fatalf("ValidateDeep() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("validator calls = %d, want 1", calls)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpValidate, Path: filepath.Join(base, "raw.txt"), Result: ValidateOK},
		{Op: OpValidate, Path: filepath.Join(base, "config.json"), Result: ValidateOK},
	})
}

func TestValidateDeepStopsOnFirstError(t *testing.T) {
	type root struct {
		First  testValidatorFile `layout:"first.json"`
		Broken testValidatorFile `layout:"broken.json"`
		Last   testValidatorFile `layout:"last.json"`
	}

	var firstCalls, brokenCalls, lastCalls int
	var layout root
	layout.First.calls = &firstCalls
	layout.Broken.calls = &brokenCalls
	layout.Broken.err = errors.New("validate failed")
	layout.Last.calls = &lastCalls

	base := t.TempDir()
	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	var report Report
	opts := ValidateOptions{Reporter: &report}

	_, err := ValidateDeep(&layout, opts)
	if err == nil {
		t.Fatal("ValidateDeep() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "validate failed") {
		t.Fatalf("ValidateDeep() error = %v, want validate failed", err)
	}

	if firstCalls != 1 || brokenCalls != 1 || lastCalls != 0 {
		t.Fatalf("validator calls = first:%d broken:%d last:%d, want 1:1:0", firstCalls, brokenCalls, lastCalls)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpValidate, Path: filepath.Join(base, "first.json"), Result: ValidateOK},
		{Op: OpValidate, Path: filepath.Join(base, "broken.json"), Result: ValidateFailed, Err: errors.New("want non-nil")},
	})
}

func TestValidateDeepOnlyValidatesCachedSlotItems(t *testing.T) {
	type item struct {
		Config testValidatorFile `layout:"config.json"`
	}

	type root struct {
		Items Slot[*item] `layout:"items"`
	}

	var layout root
	base := t.TempDir()
	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "items", "worker"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	api, err := layout.Items.At("api")
	if err != nil {
		t.Fatalf("Items.At() error = %v", err)
	}

	var apiCalls int
	api.Config.calls = &apiCalls

	var report Report
	opts := ValidateOptions{Reporter: &report}

	if _, err := ValidateDeep(&layout, opts); err != nil {
		t.Fatalf("ValidateDeep() error = %v", err)
	}

	if apiCalls != 1 {
		t.Fatalf("api validator calls = %d, want 1", apiCalls)
	}
	if layout.Items.Len() != 1 {
		t.Fatalf("Items.Len() = %d, want 1 cached item", layout.Items.Len())
	}
	if layout.Items.Has("worker") != true {
		t.Fatal("Items.Has(worker) = false, want true for on-disk item")
	}
	if _, ok := layout.Items.Get("worker"); ok {
		t.Fatal("Items.Get(worker) = ok, want uncached")
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpValidate, Path: filepath.Join(base, "items", "api", "config.json"), Result: ValidateOK},
		{Op: OpValidate, Path: filepath.Join(base, "items"), Result: ValidateTraversed},
	})
}

func TestExecValidateRejectsNonExecutableFile(t *testing.T) {
	type root struct {
		Run Exec `layout:"bin/run"`
	}

	var layout root
	base := t.TempDir()
	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(layout.Run.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(layout.Run.Path(), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := ValidateDeep(&layout, ValidateOptions{})
	if err == nil {
		t.Fatal("ValidateDeep() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "not an executable regular file") {
		t.Fatalf("ValidateDeep() error = %v, want executable validation error", err)
	}
}

func TestFileLinkValidateRejectsDirectoryTarget(t *testing.T) {
	var link FileLink
	base := t.TempDir()
	link.ComposePath(filepath.Join(base, "config.link"))

	targetDir := filepath.Join(base, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	link.SetTarget("target")

	_, err := ValidateDeep(&link, ValidateOptions{})
	if err == nil {
		t.Fatal("ValidateDeep() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "is not a file") {
		t.Fatalf("ValidateDeep() error = %v, want file-link validation error", err)
	}
}

func TestValidateDeepRejectsSymlinkParentByDefault(t *testing.T) {
	type root struct {
		Config File `layout:"config.json"`
	}

	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkBase := filepath.Join(base, "alias")

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkBase); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	var layout root
	if err := Compose(linkBase, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	_, err := ValidateDeep(&layout, ValidateOptions{})
	if err == nil {
		t.Fatal("ValidateDeep() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("ValidateDeep() error = %v, want parent-symlink error", err)
	}
}

func TestValidateDeepCanFollowSymlinkParentWhenEnabled(t *testing.T) {
	type root struct {
		Config File `layout:"config.json"`
	}

	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	linkBase := filepath.Join(base, "alias")

	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(real) error = %v", err)
	}
	if err := os.Symlink(realDir, linkBase); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	var layout root
	if err := Compose(linkBase, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	opts := ValidateOptions{PathSafetyPolicy: PathSafetyFollowSymlinks}
	if _, err := ValidateDeep(&layout, opts); err != nil {
		t.Fatalf("ValidateDeep() error = %v", err)
	}
}

func TestValidateDeepExecRejectsSymlinkLeaf(t *testing.T) {
	type root struct {
		Run Exec `layout:"run.sh"`
	}

	base := t.TempDir()
	targetPath := filepath.Join(base, "target.sh")

	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile(target) error = %v", err)
	}

	var layout root
	if err := Compose(base, &layout); err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if err := os.Symlink(targetPath, layout.Run.Path()); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	_, err := ValidateDeep(&layout, ValidateOptions{})
	if err == nil {
		t.Fatal("ValidateDeep() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "is a symlink, not a file") {
		t.Fatalf("ValidateDeep() error = %v, want symlink-leaf error", err)
	}
}
