package conduit_test

import (
	"errors"

	. "github.com/qlustra/conduit/layout"
	"github.com/qlustra/conduit/path"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateLayoutCallsValidatorsAndReports(t *testing.T) {
	type root struct {
		Raw    File              `layout:"file" path:"raw.txt"`
		Config testValidatorFile `layout:"content" path:"config.json"`
	}

	var calls int

	base := t.TempDir()
	manager, layout := composeTree[root](t, base)
	layout.Config.calls = &calls

	var report Report
	opts := ValidateOptions{Reporter: &report}

	if _, err := manager.Validate(opts); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("validator calls = %d, want 1", calls)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpValidate, Path: filepath.Join(base, "raw.txt"), Result: ValidateOK},
		{Op: OpValidate, Path: filepath.Join(base, "config.json"), Result: ValidateOK},
	})
}

func TestValidateLayoutStopsOnFirstError(t *testing.T) {
	type root struct {
		First  testValidatorFile `layout:"content" path:"first.json"`
		Broken testValidatorFile `layout:"content" path:"broken.json"`
		Last   testValidatorFile `layout:"content" path:"last.json"`
	}

	var firstCalls, brokenCalls, lastCalls int

	base := t.TempDir()
	manager, layout := composeTree[root](t, base)
	layout.First.calls = &firstCalls
	layout.Broken.calls = &brokenCalls
	layout.Broken.err = errors.New("validate failed")
	layout.Last.calls = &lastCalls

	var report Report
	opts := ValidateOptions{Reporter: &report}

	_, err := manager.Validate(opts)
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "validate failed") {
		t.Fatalf("Validate() error = %v, want validate failed", err)
	}

	if firstCalls != 1 || brokenCalls != 1 || lastCalls != 0 {
		t.Fatalf("validator calls = first:%d broken:%d last:%d, want 1:1:0", firstCalls, brokenCalls, lastCalls)
	}

	assertEntries(t, report.Entries(), []Entry{
		{Op: OpValidate, Path: filepath.Join(base, "first.json"), Result: ValidateOK},
		{Op: OpValidate, Path: filepath.Join(base, "broken.json"), Result: ValidateFailed, Err: errors.New("want non-nil")},
	})
}

func TestValidateLayoutOnlyValidatesCachedSlotItems(t *testing.T) {
	type item struct {
		Config testValidatorFile `layout:"content" path:"config.json"`
	}

	type root struct {
		Items Slot[*item] `layout:"directories" path:"items"`
	}

	base := t.TempDir()
	manager, layout := composeTree[root](t, base)

	if err := os.MkdirAll(filepath.Join(base, "items", "worker"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	api, err := layout.Items.ComposeMember("api")
	if err != nil {
		t.Fatalf("Items.ComposeMember() error = %v", err)
	}

	var apiCalls int
	api.Config.calls = &apiCalls

	var report Report
	opts := ValidateOptions{Reporter: &report}

	if _, err := manager.Validate(opts); err != nil {
		t.Fatalf("Validate() error = %v", err)
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
		Run Exec `layout:"exec" path:"bin/run"`
	}

	base := t.TempDir()
	manager, layout := composeTree[root](t, base)

	if err := os.MkdirAll(filepath.Dir(layout.Run.Path()), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(layout.Run.Path(), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := manager.Validate(ValidateOptions{})
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "not an executable regular file") {
		t.Fatalf("Validate() error = %v, want executable validation error", err)
	}
}

func TestValidateLayoutRejectsSymlinkParentByDefault(t *testing.T) {
	type root struct {
		Config File `layout:"file" path:"config.json"`
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

	manager, _ := composeTree[root](t, linkBase)

	_, err := manager.Validate(ValidateOptions{})
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("Validate() error = %v, want parent-symlink error", err)
	}
}

func TestValidateLayoutCanFollowSymlinkParentWhenEnabled(t *testing.T) {
	type root struct {
		Config File `layout:"file" path:"config.json"`
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

	manager, _ := composeTree[root](t, linkBase)

	opts := ValidateOptions{PathSafetyPolicy: path.PathSafetyFollowSymlinks}
	if _, err := manager.Validate(opts); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateLayoutExecRejectsSymlinkLeaf(t *testing.T) {
	type root struct {
		Run Exec `layout:"exec" path:"run.sh"`
	}

	base := t.TempDir()
	targetPath := filepath.Join(base, "target.sh")

	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile(target) error = %v", err)
	}

	manager, layout := composeTree[root](t, base)

	if err := os.Symlink(targetPath, layout.Run.Path()); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	_, err := manager.Validate(ValidateOptions{})
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "is a symlink, not a file") {
		t.Fatalf("Validate() error = %v, want symlink-leaf error", err)
	}
}
